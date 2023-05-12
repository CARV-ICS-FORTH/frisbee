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

package chaos

import (
	"context"
	"reflect"
	"time"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/carv-ics-forth/frisbee/controllers/common/watchers"
	"github.com/carv-ics-forth/frisbee/pkg/grafana"
	"github.com/carv-ics-forth/frisbee/pkg/lifecycle"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=frisbee.dev,resources=chaos,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.dev,resources=chaos/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.dev,resources=chaos/finalizers,verbs=update

// +kubebuilder:rbac:groups=chaos-mesh.org,resources=*,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chaos-mesh.org,resources=*/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=chaos-mesh.org,resources=*/finalizers,verbs=update

// Controller reconciles a Reference object.
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
	var chaos v1alpha1.Chaos

	var requeue bool
	result, err := common.Reconcile(ctx, r, req, &chaos, &requeue)

	if requeue {
		return result, err
	}

	r.Logger.Info("-> Reconcile",
		"obj", client.ObjectKeyFromObject(&chaos),
		"phase", chaos.Status.Phase,
		"version", chaos.GetResourceVersion(),
	)

	defer func() {
		r.Logger.Info("<- Reconciler",
			"obj", client.ObjectKeyFromObject(&chaos),
			"phase", chaos.Status.Phase,
			"version", chaos.GetResourceVersion(),
		)
	}()

	/*
		2: Load CR's children and classify their current state (view)
		------------------------------------------------------------------
	*/
	if err := r.PopulateView(ctx, req.NamespacedName); err != nil {
		return lifecycle.Failed(ctx, r, &chaos, errors.Wrapf(err, "cannot populate view for '%s'", req))
	}

	/*
		3: Use the view to update the CR's lifecycle.
		------------------------------------------------------------------
		The Update serves as "journaling" for the upcoming operations,
		and as a roadblock for stall (queued) requests.
	*/
	if r.updateLifecycle(&chaos) {
		if err := common.UpdateStatus(ctx, r, &chaos); err != nil {
			// due to the multiple updates, it is possible for this function to
			// be in conflict. We fix this issue by re-queueing the request.
			return common.RequeueAfter(r, req, time.Second)
		}
	}

	/*
		4: Make the world matching what we want in our spec.
		------------------------------------------------------------------
	*/
	switch chaos.Status.Phase {
	case v1alpha1.PhaseUninitialized, v1alpha1.PhasePending:
		// Avoid re-scheduling a scheduled job
		if chaos.Status.LastScheduleTime != nil {
			return common.Stop(r, req)
		}

		// Build the job in kubernetes
		if err := r.runJob(ctx, &chaos); err != nil {
			return lifecycle.Failed(ctx, r, &chaos, errors.Wrapf(err, "injection failed"))
		}

		// Update the scheduling information
		chaos.Status.LastScheduleTime = &metav1.Time{Time: time.Now()}

		return lifecycle.Pending(ctx, r, &chaos, "injecting fault")

	case v1alpha1.PhaseRunning:
		// Nothing to do. Just wait for something to happen.

		return common.Stop(r, req)

	case v1alpha1.PhaseSuccess:
		r.HasSucceed(ctx, &chaos)

		return common.Stop(r, req)

	case v1alpha1.PhaseFailed:
		r.HasFailed(ctx, &chaos)

		return common.Stop(r, req)
	}

	panic(errors.New("This should never happen"))
}

func (r *Controller) PopulateView(ctx context.Context, req types.NamespacedName) error {
	r.view.Reset()

	// Because we use the unstructured type,  Get will return an empty if there is no object. In turn, the
	// client's parses will return the following error: "Object 'Kind' is missing in 'unstructured object has no kind'"
	// To avoid that, we ignore errors if the map is empty -- yielding the same behavior as empty, but valid objects.
	var networkChaosList GenericFaultList

	networkChaosList.SetGroupVersionKind(NetworkChaosGVK)
	{
		if err := common.ListChildren(ctx, r.GetClient(), &networkChaosList, req); err != nil {
			return errors.Wrapf(err, "cannot list children for '%s'", req)
		}

		for i, job := range networkChaosList.Items {
			r.view.ClassifyExternal(job.GetName(), &networkChaosList.Items[i], convertChaosLifecycle)
		}
	}

	var podChaosList GenericFaultList

	podChaosList.SetGroupVersionKind(PodChaosGVK)
	{
		if err := common.ListChildren(ctx, r.GetClient(), &podChaosList, req); err != nil {
			return errors.Wrapf(err, "cannot list children for '%s'", req)
		}

		for i, job := range podChaosList.Items {
			r.view.ClassifyExternal(job.GetName(), &podChaosList.Items[i], convertChaosLifecycle)
		}
	}

	var ioChaosList GenericFaultList

	ioChaosList.SetGroupVersionKind(IOChaosGVK)
	{
		if err := common.ListChildren(ctx, r.GetClient(), &ioChaosList, req); err != nil {
			return errors.Wrapf(err, "cannot list children for '%s'", req)
		}

		for i, job := range ioChaosList.Items {
			r.view.ClassifyExternal(job.GetName(), &ioChaosList.Items[i], convertChaosLifecycle)
		}
	}

	var kernelChaosList GenericFaultList

	kernelChaosList.SetGroupVersionKind(KernelChaosGVK)
	{
		if err := common.ListChildren(ctx, r.GetClient(), &kernelChaosList, req); err != nil {
			return errors.Wrapf(err, "cannot list children for '%s'", req)
		}

		for i, job := range kernelChaosList.Items {
			r.view.ClassifyExternal(job.GetName(), &kernelChaosList.Items[i], convertChaosLifecycle)
		}
	}

	var timeChaosList GenericFaultList

	timeChaosList.SetGroupVersionKind(TimeChaosGVK)
	{
		if err := common.ListChildren(ctx, r.GetClient(), &timeChaosList, req); err != nil {
			return errors.Wrapf(err, "cannot list children for '%s'", req)
		}

		for i, job := range timeChaosList.Items {
			r.view.ClassifyExternal(job.GetName(), &timeChaosList.Items[i], convertChaosLifecycle)
		}
	}

	return nil
}

func (r *Controller) HasSucceed(ctx context.Context, chaos *v1alpha1.Chaos) {
	r.Logger.Info("CleanOnSuccess",
		"obj", client.ObjectKeyFromObject(chaos).String(),
		"successfulJobs", r.view.ListSuccessfulJobs(),
	)

	for _, job := range r.view.GetSuccessfulJobs() {
		common.Delete(ctx, r, job)
	}
}

func (r *Controller) HasFailed(ctx context.Context, chaos *v1alpha1.Chaos) {
	r.Logger.Info("!! JobError",
		"obj", client.ObjectKeyFromObject(chaos).String(),
		"reason ", chaos.Status.Reason,
		"message", chaos.Status.Message,
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
	return "chaos.frisbee.dev/finalizer"
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
	controller := &Controller{
		Manager: mgr,
		Logger:  logger.WithName("chaos"),
		view:    &lifecycle.Classifier{},
	}

	gvk := v1alpha1.GroupVersion.WithKind("Chaos")

	var (
		networkChaos GenericFault
		podChaos     GenericFault
		// blockChaos Fault
		ioChaos     GenericFault
		kernelChaos GenericFault
		timeChaos   GenericFault
	)

	networkChaos.SetGroupVersionKind(NetworkChaosGVK)
	podChaos.SetGroupVersionKind(PodChaosGVK)
	// blockChaos.SetGroupVersionKind(BlockChaosGVK)
	ioChaos.SetGroupVersionKind(IOChaosGVK)
	kernelChaos.SetGroupVersionKind(KernelChaosGVK)
	timeChaos.SetGroupVersionKind(TimeChaosGVK)

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Chaos{}).
		Named("chaos").
		Owns(&networkChaos, watchers.WatchWithRangeAnnotations(controller, gvk, grafana.TagChaos)).
		Owns(&podChaos, watchers.WatchWithPointAnnotation(controller, gvk, grafana.TagChaos)).
		// Owns(&blockChaos, builder.WithPredicates(controller.Watchers())).
		Owns(&ioChaos, watchers.WatchWithRangeAnnotations(controller, gvk, grafana.TagChaos)).
		Owns(&kernelChaos, watchers.WatchWithPointAnnotation(controller, gvk, grafana.TagChaos)).
		Owns(&timeChaos, watchers.WatchWithPointAnnotation(controller, gvk, grafana.TagChaos)).
		Complete(controller)
}
