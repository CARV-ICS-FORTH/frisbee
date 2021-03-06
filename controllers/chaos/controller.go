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

package chaos

import (
	"context"
	"reflect"
	"time"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/carv-ics-forth/frisbee/controllers/common/lifecycle"
	"github.com/go-logr/logr"
	cmap "github.com/orcaman/concurrent-map"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
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

	gvk schema.GroupVersionKind

	chaosView lifecycle.Classifier

	// because the range annotator has state (uid), we need to save in the controller's store.
	regionAnnotations cmap.ConcurrentMap
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		1: Load CR by name.
		------------------------------------------------------------------
	*/
	var cr v1alpha1.Chaos

	var requeue bool
	result, err := common.Reconcile(ctx, r, req, &cr, &requeue)

	if requeue {
		return result, errors.Wrapf(err, "initialization error")
	}

	r.Logger.Info("-> Reconcile",
		"kind", reflect.TypeOf(cr),
		"name", cr.GetName(),
		"lifecycle", cr.Status.Phase,
		"version", cr.GetResourceVersion(),
	)

	defer func() {
		r.Logger.Info("<- Reconcile",
			"kind", reflect.TypeOf(cr),
			"name", cr.GetName(),
			"lifecycle", cr.Status.Phase,
			"version", cr.GetResourceVersion(),
		)
	}()

	/*
		2: Load CR's components.
		------------------------------------------------------------------

		Because we use the unstructured type,  Get will return an empty if there is no object. In turn, the
		client's parses will return the following error: "Object 'Kind' is missing in 'unstructured object has no kind'"
		To avoid that, we ignore errors if the map is empty -- yielding the same behavior as empty, but valid objects.
	*/

	// TODO: Make it to support multiple failures
	var fault GenericFault
	if err := getRawManifest(&cr, &fault); err != nil {
		return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "cannot get fault type for chaos '%s'", cr.GetName()))
	}

	{
		key := client.ObjectKeyFromObject(&cr)

		if err := r.GetClient().Get(ctx, key, &fault); client.IgnoreNotFound(err) != nil {
			return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "retrieve chaos"))
		}
	}

	/*
		3: Update the CR status using the data we've gathered
		------------------------------------------------------------------

		The Update at this step serves two functions.
		First, it is like "journaling" for the upcoming operations.
		Second, it is a roadblock for stall (queued) requests.

		However, due to the multiple updates, it is possible for this function to
		be in conflict. We fix this issue by re-queueing the request.
		We also suppress verbose error reporting as to avoid polluting the output.
	*/
	cr.SetReconcileStatus(calculateLifecycle(&cr, &fault))

	if err := common.UpdateStatus(ctx, r, &cr); err != nil {
		r.Info("Reschedule.", "object", cr.GetName(), "UpdateStatusErr", err)
		return common.RequeueAfter(time.Second)
	}

	/*
		4: Clean up the controller from finished jobs
		------------------------------------------------------------------

		First, we'll try to clean up old jobs, so that we don't leave too many lying
		around.
	*/
	if cr.Status.Phase.Is(v1alpha1.PhaseSuccess) {
		// Remove cr children once the cr is successfully complete.
		// We should not remove the cr descriptor itself, as we need to maintain its
		// status for higher-entities like the Scenario.
		common.Delete(ctx, r, &fault)

		return common.Stop()
	}

	if cr.Status.Phase.Is(v1alpha1.PhaseFailed) {
		r.Logger.Error(errors.New("Resource has failed"), "CleanOnFailure",
			"name", cr.GetName(),
			// "successfulJobs", r.view.ListSuccessfulJobs(),
			// "runningJobs", r.view.ListRunningJobs(),
			// "pendingJobs", r.view.ListPendingJobs(),
			"reason", cr.Status.Reason,
			"message", cr.Status.Message)

		return common.Stop()
	}

	/*
		5: Make the world matching what we want in our spec
		------------------------------------------------------------------

		Once we've updated our status, we can move on to ensuring that the status of
		the world matches what we want in our spec.

		We may delete the cr, add a pod, or wait for existing pod to change its status.
	*/
	if cr.Status.LastScheduleTime != nil {
		// next reconciliation cycle will be trigger by the watchers
		return common.Stop()
	}

	if err := r.inject(ctx, &cr); err != nil {
		return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "injection failed"))
	}

	/*
		6: Avoid double actions
		------------------------------------------------------------------

		If this process restarts at this point (after posting a job, but
		before updating the status), then we might try to start the job on
		the next time.  Actually, if we re-list the Jobs on the next cycle
		we might not see our own status update, and then post one again.
		So, we need to use the job name as a lock to prevent us from making the job twice.
	*/
	cr.Status.LastScheduleTime = &metav1.Time{Time: time.Now()}

	return lifecycle.Pending(ctx, r, &cr, "injecting fault")
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
	r := &Controller{
		Manager:           mgr,
		Logger:            logger.WithName("chaos"),
		gvk:               v1alpha1.GroupVersion.WithKind("Chaos"),
		regionAnnotations: cmap.New(),
	}

	var networkChaos GenericFault
	networkChaos.SetGroupVersionKind(NetworkChaosGVK)

	var podChaos GenericFault
	podChaos.SetGroupVersionKind(PodChaosGVK)

	// var blockChaos Fault
	// blockChaos.SetGroupVersionKind(BlockChaosGVK)

	var ioChaos GenericFault
	ioChaos.SetGroupVersionKind(IOChaosGVK)

	var kernelChaos GenericFault
	kernelChaos.SetGroupVersionKind(KernelChaosGVK)

	var timeChaos GenericFault
	timeChaos.SetGroupVersionKind(TimeChaosGVK)

	return ctrl.NewControllerManagedBy(mgr).
		Named("chaos").
		For(&v1alpha1.Chaos{}).
		Owns(&networkChaos, builder.WithPredicates(r.Watchers())).
		Owns(&podChaos, builder.WithPredicates(r.Watchers())).
		// Owns(&blockChaos, builder.WithPredicates(r.Watchers())).
		Owns(&ioChaos, builder.WithPredicates(r.Watchers())).
		Owns(&kernelChaos, builder.WithPredicates(r.Watchers())).
		Owns(&timeChaos, builder.WithPredicates(r.Watchers())).
		Complete(r)
}
