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

package chaos

import (
	"context"
	"reflect"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/utils"
	"github.com/fnikolai/frisbee/controllers/utils/lifecycle"
	"github.com/go-logr/logr"
	cmap "github.com/orcaman/concurrent-map"
	"github.com/pkg/errors"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=frisbee.io,resources=chaos,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.io,resources=chaos/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.io,resources=chaos/finalizers,verbs=update

// +kubebuilder:rbac:groups=chaos-mesh.org,resources=networkchaos,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chaos-mesh.org,resources=networkchaos/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=chaos-mesh.org,resources=networkchaos/finalizers,verbs=update

// Controller reconciles a Reference object
type Controller struct {
	ctrl.Manager
	logr.Logger

	// annotator sends annotations to grafana
	annotators cmap.ConcurrentMap
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		### 1: Load the chaos by name.
	*/
	var chaos v1alpha1.Chaos

	var ret bool
	result, err := utils.Reconcile(ctx, r, req, &chaos, &ret)
	if ret {
		return result, err
	}

	r.Logger.Info("-> Reconcile",
		"kind", reflect.TypeOf(chaos),
		"name", chaos.GetName(),
		"lifecycle", chaos.Status.Phase,
		"deleted", !chaos.GetDeletionTimestamp().IsZero(),
	)

	defer func() {
		r.Logger.Info("<- Reconcile",
			"kind", reflect.TypeOf(chaos),
			"name", chaos.GetName(),
			"lifecycle", chaos.Status.Phase,
			"deleted", !chaos.GetDeletionTimestamp().IsZero(),
		)
	}()

	/*
		### 2: Load the Chaos's components.

		Because we use the unstructured type,  Get will return an empty if there is no object. In turn, the
		client's parses will return the following error: "Object 'Kind' is missing in 'unstructured object has no kind'"
		To avoid that, we ignore errors if the map is empty -- yielding the same behavior as empty, but valid objects.
	*/
	handler := r.dispatch(&chaos)

	fault := handler.GetFault()
	{
		key := client.ObjectKeyFromObject(&chaos)

		err := r.GetClient().Get(ctx, key, fault)
		if err != nil && !k8errors.IsNotFound(err) {
			return lifecycle.Failed(ctx, r, &chaos, errors.Wrapf(err, "retrieve fault"))
		}
	}

	/*
		### 3: Calculate and update the service status

		Using the date we've gathered, we'll update the status of our CRD.
		Depending on the outcome, the execution may proceed or terminate.
	*/

	newStatus := CalculateLifecycle(fault)
	chaos.Status.Lifecycle = newStatus

	if _, err := utils.UpdateStatus(ctx, r, &chaos); err != nil {
		r.Logger.Error(err, "update status error")

		return lifecycle.Failed(ctx, r, &chaos, errors.Wrapf(err, "status update"))
	}

	/*
		### 4: Based on the current status, decide what to do next.

		We may inject a failure, revoke a failure, or wait ...
	*/

	switch newStatus.Phase {
	case v1alpha1.PhaseUninitialized:
		panic("this should never happen")
	case v1alpha1.PhaseInitializing:
		// ... proceed to inject the fault

	case v1alpha1.PhasePending, v1alpha1.PhaseRunning:
		// ... fault is already scheduled. nothing to do
		return utils.Stop()

	case v1alpha1.PhaseSuccess:
		if err := r.GetClient().Delete(ctx, &chaos); client.IgnoreNotFound(err) != nil {
			return lifecycle.Failed(ctx, r, &chaos, errors.Wrapf(err, "fault revoke"))
		}

		return utils.Stop()

	case v1alpha1.PhaseFailed:
		r.Logger.Error(errors.New(newStatus.Reason), "chaos failed", "cluster", chaos.GetName())

		return utils.Stop()
	}

	/*
		### 6: Inject the fault
	*/
	nextFault := handler.ConstructJob(ctx, &chaos)

	if err := utils.CreateUnlessExists(ctx, r, &nextFault); err != nil {
		return lifecycle.Failed(ctx, r, &chaos, errors.Wrapf(err, "injection failed"))
	}

	r.Logger.Info("Injected fault",
		"Chaos", chaos.GetName(),
		"type", handler.GetName(),
	)

	return utils.Stop()
}

func (r *Controller) Finalizer() string {
	return "chaos.frisbee.io/finalizer"
}

func (r *Controller) Finalize(obj client.Object) error {
	r.Logger.Info("Finalize", "kind", reflect.TypeOf(obj), "name", obj.GetName())

	return nil
}

func NewController(mgr ctrl.Manager, logger logr.Logger) error {
	r := &Controller{
		Manager:    mgr,
		Logger:     logger.WithName("chaos"),
		annotators: cmap.New(),
	}

	var fault Fault
	AsPartition(&fault)

	return ctrl.NewControllerManagedBy(mgr).
		Named("chaos").
		For(&v1alpha1.Chaos{}).
		Owns(&fault, builder.WithPredicates(r.Watchers())).
		Complete(r)
}

type Fault = unstructured.Unstructured

type chaoHandler interface {
	GetFault() *Fault

	GetName() string

	ConstructJob(ctx context.Context, obj *v1alpha1.Chaos) Fault

	// Inject(ctx context.Context, obj *v1alpha1.Chaos) error
	// WaitForDuration(ctx context.Context, obj *v1alpha1.Chaos) error
	// Revoke(ctx context.Context, obj *v1alpha1.Chaos) error
}

func (r *Controller) dispatch(chaos *v1alpha1.Chaos) chaoHandler {
	switch chaos.Spec.Type {
	case v1alpha1.FaultPartition:
		return &partition{spec: chaos.Spec.Partition}

	default:
		panic("should never happen")
	}
}
