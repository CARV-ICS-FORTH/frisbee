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

package cluster

import (
	"context"
	"reflect"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/controllers/common/lifecycle"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=frisbee.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.io,resources=clusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.io,resources=clusters/finalizers,verbs=update

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;

func NewController(mgr ctrl.Manager, logger logr.Logger) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Cluster{}).
		Named("cluster").
		Complete(&Reconciler{
			Manager: mgr,
			Logger:  logger.WithName("cluster"),
		})
}

// Reconciler reconciles a ByCluster object.
type Reconciler struct {
	ctrl.Manager
	logr.Logger
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var cluster v1alpha1.Cluster

	var ret bool
	result, err := common.Reconcile(ctx, r, req, &cluster, &ret)
	if ret {
		return result, err
	}

	r.Logger.Info("-> Reconcile", "kind", reflect.TypeOf(cluster), "name", cluster.GetName(), "lifecycle", cluster.Status.Phase)
	defer func() {
		r.Logger.Info("<- Reconcile", "kind", reflect.TypeOf(cluster), "name", cluster.GetName(), "lifecycle", cluster.Status.Phase)
	}()

	// reconciliation logic
	switch cluster.Status.Phase {
	case v1alpha1.PhaseUninitialized:
		return lifecycle.Pending(ctx, &cluster, "waiting for services to become ready")

	case v1alpha1.PhasePending:
		if err := r.prepare(ctx, &cluster); err != nil {
			return lifecycle.Failed(ctx, &cluster, err)
		}

		expected := cluster.Status.ExpectedServices
		if len(expected) == 0 {
			return lifecycle.Failed(ctx, &cluster, errors.New("no services are expected. stall condition ?"))
		}

		// start listening for events.
		// If any of the services is created, the cluster will go to the running phase.
		// If any of the services is failed, the cluster will go to the failed phase.
		// if all services are successfully terminated, the cluster will go to the success phase.
		err := lifecycle.New(
			lifecycle.Watch(&v1alpha1.Service{}, expected.GetNames()...),
			lifecycle.WithFilters(lifecycle.FilterByParent(cluster.GetUID())),
			lifecycle.WithAnnotator(&lifecycle.PointAnnotation{}), // Register event to grafana
			lifecycle.WithLogger(r.Logger),
			lifecycle.WithUpdateParentStatus(&cluster),
		).Run(ctx)

		if err != nil {
			return lifecycle.Failed(ctx, &cluster, err)
		}

		// start creating services
		if err := r.create(ctx, &cluster, expected); err != nil {
			return lifecycle.Failed(ctx, &cluster, err)
		}

		return common.Stop()

	case v1alpha1.PhaseRunning:
		return common.Stop()

	case v1alpha1.PhaseSuccess:
		// remove the cluster upon completion

		if err := r.GetClient().Delete(ctx, &cluster); err != nil {
			r.Logger.Error(err, "garbage collection failed", "cluster", cluster.GetName())
		}

		r.Logger.Info("garbage collection was complete", "cluster", cluster.GetName())

		return common.Stop()

	case v1alpha1.PhaseFailed:
		r.Logger.Info("Cluster has failed", "name", cluster.GetName())

		return common.Stop()

	case v1alpha1.PhaseChaos: // Invalid
		panic(errors.Errorf("invalid lifecycle phase %s", cluster.Status.Phase))

	default:
		panic(errors.Errorf("unknown lifecycle phase: %s", cluster.Status.Phase))
	}
}

func (r *Reconciler) Finalizer() string {
	return "clusters.frisbee.io/finalizer"
}

func (r *Reconciler) Finalize(obj client.Object) error {
	r.Logger.Info("Finalize", "kind", reflect.TypeOf(obj), "name", obj.GetName())

	return nil
}
