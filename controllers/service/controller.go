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

package service

import (
	"context"
	"reflect"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/controllers/common/lifecycle"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=frisbee.io,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.io,resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.io,resources=services/finalizers,verbs=update

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;

// Controller reconciles a Service object.
type Controller struct {
	ctrl.Manager
	logr.Logger

	// annotator sends annotations to grafana
	annotator lifecycle.Annotator
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		### 1: Load the service by name.
	*/
	var service v1alpha1.Service

	var ret bool
	result, err := common.Reconcile(ctx, r, req, &service, &ret)
	if ret {
		return result, err
	}

	r.Logger.Info("-> Reconcile",
		"kind", reflect.TypeOf(service),
		"name", service.GetName(),
		"lifecycle", service.Status.Phase,
		"deleted", !service.GetDeletionTimestamp().IsZero(),
	)

	defer func() {
		r.Logger.Info("<- Reconcile",
			"kind", reflect.TypeOf(service),
			"name", service.GetName(),
			"lifecycle", service.Status.Phase,
			"deleted", !service.GetDeletionTimestamp().IsZero(),
		)
	}()

	/*
		### 2: Load the service's components.

		In particular, the component is a pod with the same name as the service.
	*/

	var pod corev1.Pod
	{
		key := client.ObjectKeyFromObject(&service)

		if err := r.GetClient().Get(ctx, key, &pod); client.IgnoreNotFound(err) != nil {
			return lifecycle.Failed(ctx, r, &service, errors.Wrapf(err, "retrieve pod"))
		}
	}

	/*
		### 3: Calculate and update the service status

		Using the date we've gathered, we'll update the status of our CRD.
		Depending on the outcome, the execution may proceed or terminate.
	*/
	newStatus := calculateLifecycle(&service, &pod)
	service.Status.Lifecycle = newStatus

	if _, err := common.UpdateStatus(ctx, r, &service); err != nil {
		r.Logger.Error(err, "update status error")

		return lifecycle.Failed(ctx, r, &service, errors.Wrapf(err, "status update"))
	}

	/*
		### 4: Based on the current status, decide what to do next.

		We may delete the service, add a pod, or wait for existing pod to change its status.
	*/

	switch newStatus.Phase {
	case v1alpha1.PhaseUninitialized:
		panic("this should never happen")

	case v1alpha1.PhaseInitialized:
		// ... proceed to create pod

	case v1alpha1.PhasePending, v1alpha1.PhaseRunning:
		// ... Pod is already scheduled. nothing to do
		return common.Stop()

	case v1alpha1.PhaseSuccess:
		// Although we can remove a service when is complete, we recommend not to id as it will break the
		// cluster manager -- it needs to track the total number of created services.

		/*
			if err := r.GetClient().Delete(ctx, &service); client.IgnoreNotFound(err) != nil {
				return lifecycle.Failed(ctx, r, &service, errors.Wrapf(err, "service deletion"))
			}
		*/

		return common.Stop()

	case v1alpha1.PhaseFailed:
		r.Logger.Error(errors.New(newStatus.Reason), "cluster failed", "cluster", service.GetName())

		return common.Stop()
	}

	/*
		### 5: Create new Pod for the service
	*/

	if err := r.runJob(ctx, &service); err != nil {
		return lifecycle.Failed(ctx, r, &service, errors.Wrapf(err, "cannot create pod"))
	}

	r.Logger.Info("Create pod",
		"service", service.GetName(),
		"pod", pod.GetName(),
	)

	// exit and wait for watchers to trigger the next reconcile cycle
	return common.Stop()
}

/*
### Finalizers
*/

func (r *Controller) Finalizer() string {
	return "services.frisbee.io/finalizer"
}

func (r *Controller) Finalize(obj client.Object) error {
	r.Logger.Info("Finalize", "kind", reflect.TypeOf(obj), "name", obj.GetName())

	return nil
}

/*
### Setup
	Finally, we'll update our setup.

	We'll inform the manager that this controller owns some Services, so that it
	will automatically call Reconcile on the underlying Service when a Pod changes, is
	deleted, etc.
*/

func NewController(mgr ctrl.Manager, logger logr.Logger) error {
	r := &Controller{
		Manager:   mgr,
		Logger:    logger.WithName("service"),
		annotator: &lifecycle.PointAnnotation{},
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named("service").
		For(&v1alpha1.Service{}).
		Owns(&corev1.Pod{}, builder.WithPredicates(r.Watchers())).
		Complete(r)
}
