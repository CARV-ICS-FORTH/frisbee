/*
Copyright 2021 ICS-FORTH.

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
	"fmt"
	"reflect"
	"time"

	lifecycle2 "github.com/carv-ics-forth/frisbee/pkg/lifecycle"
	"k8s.io/apimachinery/pkg/types"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/go-logr/logr"
	cmap "github.com/orcaman/concurrent-map"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
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

	gvk schema.GroupVersionKind

	view *lifecycle2.Classifier

	// because the range annotator has state (uid), we need to save in the controller's store.
	regionAnnotations cmap.ConcurrentMap
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		1: Load CR by name and extract the Desired State
		------------------------------------------------------------------
	*/
	var cr v1alpha1.Service

	var requeue bool
	result, err := common.Reconcile(ctx, r, req, &cr, &requeue)

	if requeue {
		return result, err
	}

	r.Logger.Info("-> Reconcile",
		"obj", client.ObjectKeyFromObject(&cr),
		"phase", cr.Status.Phase,
		"version", cr.GetResourceVersion(),
	)

	defer func() {
		r.Logger.Info("<- Reconcile",
			"obj", client.ObjectKeyFromObject(&cr),
			"phase", cr.Status.Phase,
			"version", cr.GetResourceVersion(),
		)
	}()

	/*
		2: Load CR's children and classify their current state (view)
		------------------------------------------------------------------
	*/
	if err := r.PopulateView(ctx, req.NamespacedName); err != nil {
		return lifecycle2.Failed(ctx, r, &cr, errors.Wrapf(err, "cannot populate view for '%s'", req))
	}

	/*
		3: Use the view to update the CR's lifecycle.
		------------------------------------------------------------------
		The Update serves as "journaling" for the upcoming operations,
		and as a roadblock for stall (queued) requests.
	*/
	r.updateLifecycle(&cr)
	if err := common.UpdateStatus(ctx, r, &cr); err != nil {
		// due to the multiple updates, it is possible for this function to
		// be in conflict. We fix this issue by re-queueing the request.
		return common.RequeueAfter(time.Second)
	}

	/*
		4: Make the world matching what we want in our spec.
		------------------------------------------------------------------
	*/

	switch cr.Status.Phase {
	case v1alpha1.PhaseSuccess:
		if err := r.HasSucceed(ctx, &cr); err != nil {
			return common.RequeueAfter(time.Second)
		}

		return common.Stop()

	case v1alpha1.PhaseFailed:
		if err := r.HasFailed(ctx, &cr); err != nil {
			return common.RequeueAfter(time.Second)
		}

		return common.Stop()

	case v1alpha1.PhaseRunning:
		// Nothing to do. Just wait for something to happen.
		r.Logger.Info(".. Awaiting",
			"obj", client.ObjectKeyFromObject(&cr),
			cr.Status.Reason, cr.Status.Message,
		)

		return common.Stop()

	case v1alpha1.PhaseUninitialized:
		// Avoid re-scheduling a scheduled job
		if cr.Status.LastScheduleTime != nil {
			// next reconciliation cycle will be trigger by the watchers
			return common.Stop()
		}

		// Build the job in kubernetes
		if err := r.runJob(ctx, &cr); err != nil {
			return lifecycle2.Failed(ctx, r, &cr, err)
		}

		// Update the scheduling information
		cr.Status.LastScheduleTime = &metav1.Time{Time: time.Now()}

		return lifecycle2.Pending(ctx, r, &cr, "Submit pod create request")

	case v1alpha1.PhasePending:
		// Nothing to do
		return common.Stop()
	}

	panic(errors.New("This should never happen"))
}

func (r *Controller) PopulateView(ctx context.Context, req types.NamespacedName) error {
	r.view.Reset()

	var podJobs corev1.PodList
	{
		if err := common.ListChildren(ctx, r, &podJobs, req); err != nil {
			return errors.Wrapf(err, "cannot list children for '%s'", req)
		}

		for i, job := range podJobs.Items {
			r.view.ClassifyExternal(job.GetName(), &podJobs.Items[i], convertPodLifecycle)
		}
	}

	return nil
}

func (r *Controller) HasSucceed(ctx context.Context, cr *v1alpha1.Service) error {

	r.Logger.Info("CleanOnSuccess",
		"obj", client.ObjectKeyFromObject(cr).String(),
		"successfulJobs", r.view.ListSuccessfulJobs(),
	)

	for _, job := range r.view.GetSuccessfulJobs() {
		common.Delete(ctx, r, job)
	}

	// TODO: remove dns service.

	return nil
}

func (r *Controller) HasFailed(ctx context.Context, cr *v1alpha1.Service) error {

	r.Logger.Error(fmt.Errorf(cr.Status.Message), "!! "+cr.Status.Reason,
		"obj", client.ObjectKeyFromObject(cr).String())

	// Remove the non-failed components. Leave the failed jobs and system jobs for postmortem analysis.
	for _, job := range r.view.GetPendingJobs() {
		common.Delete(ctx, r, job)
	}

	for _, job := range r.view.GetRunningJobs() {
		common.Delete(ctx, r, job)
	}

	return nil
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
	r := &Controller{
		Manager:           mgr,
		Logger:            logger.WithName("service"),
		gvk:               v1alpha1.GroupVersion.WithKind("Service"),
		view:              &lifecycle2.Classifier{},
		regionAnnotations: cmap.New(),
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named("service").
		For(&v1alpha1.Service{}).
		Owns(&corev1.Pod{}, builder.WithPredicates(r.Watchers())).
		Complete(r)
}
