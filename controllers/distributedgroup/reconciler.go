// Licensed to FORTH/ICS under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. FORTH/ICS licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package distributedgroup

import (
	"context"
	"reflect"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/controllers/common/lifecycle"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=frisbee.io,resources=distributedgroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.io,resources=distributedgroups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.io,resources=distributedgroups/finalizers,verbs=update

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;

func NewController(mgr ctrl.Manager, logger logr.Logger) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.DistributedGroup{}).
		Named("distributedgroups").
		Complete(&Reconciler{
			Client: mgr.GetClient(),
			Logger: logger.WithName("distributedgroups"),
		})
}

// Reconciler reconciles a Templates object.
type Reconciler struct {
	client.Client
	logr.Logger
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var group v1alpha1.DistributedGroup

	var ret bool
	result, err := common.Reconcile(ctx, r, req, &group, &ret)
	if ret {
		return result, err
	}

	r.Logger.Info("-> Reconcile", "kind", reflect.TypeOf(group), "name", group.GetName(), "lifecycle", group.Status.Phase)
	defer func() {
		r.Logger.Info("<- Reconcile", "kind", reflect.TypeOf(group), "name", group.GetName(), "lifecycle", group.Status.Phase)
	}()

	// reconciliation logic
	switch group.Status.Phase {
	case v1alpha1.PhaseUninitialized:
		if err := r.prepare(ctx, &group); err != nil {
			return lifecycle.Failed(ctx, &group, err)
		}

		return lifecycle.Pending(ctx, &group, "waiting for services to become ready")

	case v1alpha1.PhasePending:
		expected := group.Status.ExpectedServices.GetNames()
		if len(expected) == 0 {
			return lifecycle.Failed(ctx, &group, errors.New("no services are expected. stall condition ?"))
		}

		logrus.Warn("Listening for children:", expected)

		// start listening for events.
		// If any of the services is created, the group will go to the running phase.
		// If any of the services is failed, the group will go to the failed phase.
		// if all services are successfully terminated, the group will go to the success phase.
		err := lifecycle.New(
			lifecycle.WatchExternal(&v1.Pod{}, lifecycle.Pod(), expected...),
			lifecycle.WithFilters(lifecycle.FilterByParent(&group)),
			lifecycle.WithAnnotator(&lifecycle.PointAnnotation{}), // Register event to grafana
			lifecycle.WithLogger(r.Logger),
			lifecycle.WithUpdateParent(group.DeepCopy()),
		).Run(ctx)

		if err != nil {
			return lifecycle.Failed(ctx, &group, err)
		}

		// start creating services
		if err := r.create(ctx, &group, group.Status.ExpectedServices); err != nil {
			return lifecycle.Failed(ctx, &group, err)
		}

		return common.Stop()

	case v1alpha1.PhaseRunning:
		return common.Stop()

	case v1alpha1.PhaseSuccess:
		// remove the group upon completion

		if err := r.Client.Delete(ctx, &group); err != nil {
			r.Logger.Error(err, "garbage collection failed", "group", group.GetName())
		}

		r.Logger.Info("garbage collection was complete", "group", group.GetName())

		return common.Stop()

	case v1alpha1.PhaseFailed:
		r.Logger.Info("DistributedGroup has failed", "name", group.GetName())

		return common.Stop()

	case v1alpha1.PhaseChaos: // Invalid
		panic(errors.Errorf("invalid lifecycle phase %s", group.Status.Phase))

	default:
		panic(errors.Errorf("unknown lifecycle phase: %s", group.Status.Phase))
	}
}

func (r *Reconciler) Finalizer() string {
	return "distributedgroups.frisbee.io/finalizer"
}

func (r *Reconciler) Finalize(obj client.Object) error {
	r.Logger.Info("Finalize", "kind", reflect.TypeOf(obj), "name", obj.GetName())

	return nil
}
