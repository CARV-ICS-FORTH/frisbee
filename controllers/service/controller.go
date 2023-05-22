/*
Copyright 2021-2023 ICS-FORTH.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package service

import (
	"context"
	"reflect"
	"time"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/carv-ics-forth/frisbee/controllers/common/watchers"
	"github.com/carv-ics-forth/frisbee/pkg/lifecycle"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=frisbee.dev,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.dev,resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.dev,resources=services/finalizers,verbs=update

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=pods/exec,verbs=get;list;watch;create;update;patch;delete

// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services/status,verbs=get;list;watch

// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims/status,verbs=get;list;watch

// +kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses/finalizers,verbs=update

// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/finalizers,verbs=update

// Controller reconciles a Service object.
type Controller struct {
	ctrl.Manager
	logr.Logger

	view *lifecycle.Classifier
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		1: Load CR by name and extract the Desired State
		------------------------------------------------------------------
	*/
	var service v1alpha1.Service

	var requeue bool
	result, err := common.Reconcile(ctx, r, req, &service, &requeue)

	if requeue {
		return result, err
	}

	r.Logger.Info("-> Reconcile",
		"obj", client.ObjectKeyFromObject(&service),
		"phase", service.Status.Phase,
		"version", service.GetResourceVersion(),
	)

	defer func() {
		r.Logger.Info("<- Reconciler",
			"obj", client.ObjectKeyFromObject(&service),
			"phase", service.Status.Phase,
			"version", service.GetResourceVersion(),
		)
	}()

	/*
		2: Load CR's children and classify their current state (view)
		------------------------------------------------------------------
	*/
	if err := r.PopulateView(ctx, req.NamespacedName); err != nil {
		return lifecycle.Failed(ctx, r, &service, errors.Wrapf(err, "cannot populate view for '%s'", req))
	}

	/*
		3: Use the view to update the CR's lifecycle.
		------------------------------------------------------------------
		The Update serves as "journaling" for the upcoming operations,
		and as a roadblock for stall (queued) requests.
	*/
	if r.updateLifecycle(&service) {
		if err := common.UpdateStatus(ctx, r, &service); err != nil {
			// due to the multiple updates, it is possible for this function to
			// be in conflict. We fix this issue by re-queueing the request.
			return common.RequeueAfter(r, req, time.Second)
		}
	}

	/*
		4: Make the world matching what we want in our spec.
		------------------------------------------------------------------
	*/

	switch service.Status.Phase {
	case v1alpha1.PhaseUninitialized:
		// Avoid re-scheduling a scheduled job
		if service.Status.LastScheduleTime != nil {
			// next reconciliation cycle will be trigger by the watchers
			return common.Stop(r, req)
		}

		// Build the job in kubernetes
		if err := r.runJob(ctx, &service); err != nil {
			return lifecycle.Failed(ctx, r, &service, err)
		}

		// Update the scheduling information
		service.Status.LastScheduleTime = &metav1.Time{Time: time.Now()}

		return lifecycle.Pending(ctx, r, &service, "Submit pod create request")

	case v1alpha1.PhasePending, v1alpha1.PhaseRunning:
		// Nothing to do. We are not waiting for Pod to begin.
		return common.Stop(r, req)

	case v1alpha1.PhaseSuccess:
		r.HasSucceed(ctx, &service)

		return common.Stop(r, req)

	case v1alpha1.PhaseFailed:
		r.HasFailed(ctx, &service)

		return common.Stop(r, req)
	}

	panic("this should never happen")
}

func (r *Controller) PopulateView(ctx context.Context, req types.NamespacedName) error {
	r.view.Reset()

	var podJobs corev1.PodList
	{
		if err := common.ListChildren(ctx, r.GetClient(), &podJobs, req); err != nil {
			return errors.Wrapf(err, "cannot list children for '%s'", req)
		}

		for i, job := range podJobs.Items {
			r.view.ClassifyExternal(job.GetName(), &podJobs.Items[i], convertPodLifecycle)
		}
	}

	return nil
}

func (r *Controller) HasSucceed(ctx context.Context, cr *v1alpha1.Service) {
	r.Logger.Info("CleanOnSuccess",
		"obj", client.ObjectKeyFromObject(cr).String(),
		"successfulJobs", r.view.ListSuccessfulJobs(),
	)

	for _, job := range r.view.GetSuccessfulJobs() {
		common.Delete(ctx, r, job)
	}
}

func (r *Controller) HasFailed(ctx context.Context, cr *v1alpha1.Service) {
	r.Logger.Info("!! JobError",
		"obj", client.ObjectKeyFromObject(cr).String(),
		"reason ", cr.Status.Reason,
		"message", cr.Status.Message,
	)

	// Remove the non-failed components. Leave the failed jobs and system jobs for postmortem analysis.
	for _, job := range r.view.GetPendingJobs() {
		common.Delete(ctx, r, job)
	}

	for _, job := range r.view.GetRunningJobs() {
		common.Delete(ctx, r, job)
	}
}

/*
### Finalizers
*/

func (r *Controller) Finalizer() string {
	return "services.frisbee.dev/finalizer"
}

func (r *Controller) Finalize(obj client.Object) error {
	r.Logger.Info("XX Finalize",
		"kind", reflect.TypeOf(obj),
		"name", obj.GetName(),
		"version", obj.GetResourceVersion(),
	)

	return nil
}

/*
### Setup
	Finally, we'll update our setup.

	We'll inform the manager that this controller owns some resources, so that it
	will automatically call Reconcile on the underlying controller when a resource changes, is
	deleted, etc.
*/

func NewController(mgr ctrl.Manager, logger logr.Logger) error {
	reconciler := &Controller{
		Manager: mgr,
		Logger:  logger.WithName("service"),
		view:    &lifecycle.Classifier{},
	}

	gvk := v1alpha1.GroupVersion.WithKind("Service")

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Service{}).
		Named("service").
		Owns(&corev1.Pod{}, watchers.Watch(reconciler, gvk)).
		Complete(reconciler)
}
