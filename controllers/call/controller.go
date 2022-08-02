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

package call

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"reflect"
	"time"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/carv-ics-forth/frisbee/controllers/common/expressions"
	"github.com/carv-ics-forth/frisbee/controllers/common/lifecycle"
	"github.com/carv-ics-forth/frisbee/controllers/common/scheduler"
	"github.com/carv-ics-forth/frisbee/controllers/common/watchers"
	"github.com/carv-ics-forth/frisbee/pkg/executor"
	"github.com/go-logr/logr"
	cmap "github.com/orcaman/concurrent-map"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=frisbee.dev,resources=calls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.dev,resources=calls/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.dev,resources=calls/finalizers,verbs=update

// +kubebuilder:rbac:groups=frisbee.dev,resources=virtualobjects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.dev,resources=virtualobjects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.dev,resources=virtualobjects/finalizers,verbs=update

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;list;watch

// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete

// Controller reconciles a Cluster object.
type Controller struct {
	ctrl.Manager
	logr.Logger

	gvk schema.GroupVersionKind

	view lifecycle.Classifier

	// executor is used to run commands directly into containers
	executor executor.Executor

	regionAnnotations cmap.ConcurrentMap
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current view of the cluster closer to the desired view.
func (r *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		1: Load CR by name and extract the Desired State
		------------------------------------------------------------------
	*/
	var cr v1alpha1.Call

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
		2: Load CR's children and classify their current state (view)
		------------------------------------------------------------------
	*/
	if err := r.PopulateView(ctx, req.NamespacedName); err != nil {
		return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "cannot get the cluster view for '%s'", req))
	}

	/*
		3: Use the view to update the CR's lifecycle.
		------------------------------------------------------------------
		The Update serves as "journaling" for the upcoming operations,
		and as a roadblock for stall (queued) requests.
	*/
	r.calculateLifecycle(&cr)

	if err := common.UpdateStatus(ctx, r, &cr); err != nil {
		// due to the multiple updates, it is possible for this function to
		// be in conflict. We fix this issue by re-queueing the request.
		return common.RequeueAfter(time.Second)
	}

	/*
		4: Make the world matching what we want in our spec.
		------------------------------------------------------------------
	*/

	if cr.Spec.Suspend != nil && *cr.Spec.Suspend {
		// If this object is suspended, we don't want to run any jobs, so we'll stop now.
		// This is useful if something's broken with the job we're running, and we want to
		// pause runs to investigate the cluster, without deleting the object.
		r.Logger.Info("Suspended",
			"resource", cr.GetName(),
			"reason", cr.Status.Reason,
			"message", cr.Status.Message,
		)

		return common.Stop()
	}

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
			"name", cr.GetName(),
			cr.Status.Reason, cr.Status.Message,
		)

		return common.Stop()

	case v1alpha1.PhaseUninitialized:
		if err := r.Initialize(ctx, &cr); err != nil {
			return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "cannot initialize"))
		}

		return lifecycle.Pending(ctx, r, &cr, "ready to start submitting jobs.")

	case v1alpha1.PhasePending:
		nextJob := cr.Status.ScheduledJobs + 1

		/*
			If all jobs are scheduled but are not in the Running phase, they may be in the Pending phase.
			In both cases, we have nothing else to do but waiting for the next reconciliation cycle.
		*/
		if cr.Spec.Until == nil && (nextJob >= len(cr.Status.QueuedJobs)) {
			r.Logger.Info(".. Awaiting",
				"name", cr.GetName(),
				cr.Status.Reason, cr.Status.Message,
			)

			return common.Stop()
		}

		// Get the next scheduled job
		if hasJob, requeue, err := scheduler.Schedule(ctx, r, &cr, cr.Spec.Schedule, cr.Status.LastScheduleTime, r.view); !hasJob {
			return requeue, err
		}

		// Build the job in kubernetes
		if err := r.runJob(ctx, &cr, nextJob); err != nil {
			return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "cannot create job"))
		}

		// Update the scheduling information
		cr.Status.ScheduledJobs = nextJob
		cr.Status.LastScheduleTime = &metav1.Time{Time: time.Now()}

		return lifecycle.Pending(ctx, r, &cr, "some jobs are still pending")
	}

	panic(errors.New("This should never happen"))
}

func (r *Controller) Initialize(ctx context.Context, cr *v1alpha1.Call) error {
	/*
		We construct a list of job specifications based on the CR's template.
		This list is used by the execution step to create the actual job.
		If the template is invalid, it should be captured at this stage.
	*/
	jobList, err := r.constructJobSpecList(ctx, cr)
	if err != nil {
		return errors.Wrapf(err, "cannot build joblist")
	}

	cr.Status.QueuedJobs = jobList
	cr.Status.ScheduledJobs = -1

	// Metrics-driven execution requires to set alerts on Grafana.
	if until := cr.Spec.Until; until != nil && until.HasMetricsExpr() {
		if err := expressions.SetAlert(ctx, cr, until.Metrics); err != nil {
			return errors.Wrapf(err, "spec.until")
		}
	}

	if schedule := cr.Spec.Schedule; schedule != nil && schedule.Event.HasMetricsExpr() {
		if err := expressions.SetAlert(ctx, cr, schedule.Event.Metrics); err != nil {
			return errors.Wrapf(err, "spec.schedule")
		}
	}

	if _, err := lifecycle.Pending(ctx, r, cr, "submitting job requests"); err != nil {
		return errors.Wrapf(err, "status update")
	}

	return nil
}

func (r *Controller) PopulateView(ctx context.Context, req types.NamespacedName) error {
	r.view.Reset()

	var streamJobs v1alpha1.VirtualObjectList
	{
		if err := common.ListChildren(ctx, r, &streamJobs, req); err != nil {
			return errors.Wrapf(err, "unable to list children for '%s'", req)
		}

		for i, job := range streamJobs.Items {
			r.view.Classify(job.GetName(), &streamJobs.Items[i])
		}
	}

	return nil
}

func (r *Controller) HasSucceed(ctx context.Context, cr *v1alpha1.Call) error {
	r.GetEventRecorderFor(cr.GetName()).Event(cr, corev1.EventTypeNormal,
		cr.Status.Lifecycle.Reason, cr.Status.Lifecycle.Message)

	r.Logger.Info("CleanOnSuccess",
		"name", cr.GetName(),
		"sucessfulJobs", r.view.ListSuccessfulJobs(),
	)

	/*
		Remove cr children once the cr is successfully complete.
		We should not remove the cr descriptor itself, as we need to maintain its
		status for higher-entities like the Scenario.
	*/
	for _, job := range r.view.GetSuccessfulJobs() {
		common.Delete(ctx, r, job)
	}

	return nil
}

func (r *Controller) HasFailed(ctx context.Context, cr *v1alpha1.Call) error {
	r.GetEventRecorderFor(cr.GetName()).Event(cr, corev1.EventTypeWarning,
		cr.Status.Lifecycle.Reason, cr.Status.Lifecycle.Message)

	r.Logger.Error(errors.New("Resource has failed"), "CleanOnFailure",
		"name", cr.GetName(),
		"successfulJobs", r.view.ListSuccessfulJobs(),
		"runningJobs", r.view.ListRunningJobs(),
		"pendingJobs", r.view.ListPendingJobs(),
		"reason", cr.Status.Reason,
		"message", cr.Status.Message)

	// Remove the non-failed components. Leave the failed jobs and system jobs for postmortem analysis.
	for _, job := range r.view.GetPendingJobs() {
		common.Delete(ctx, r, job)
	}

	for _, job := range r.view.GetRunningJobs() {
		common.Delete(ctx, r, job)
	}

	for _, job := range r.view.GetSuccessfulJobs() {
		common.Delete(ctx, r, job)
	}

	// Block from creating further jobs
	suspend := true
	cr.Spec.Suspend = &suspend

	return common.Update(ctx, r, cr)
}

/*
### Finalizers

*/

func (r *Controller) Finalizer() string {
	return "calls.frisbee.dev/finalizer"
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
	Finally, we'll update our setup.  In order to allow to quickly look up Entries by their owner,
	we'll need an index.  We declare an index key that we can later use with the client as a pseudo-field name,
	and then describe how to extract the indexed value from the Service object.
	The indexer will automatically take care of namespaces for us, so we just have to extract the
	owner name if the Service has a Cluster owner.

	Additionally, We'll inform the manager that this controller owns some resources, so that it
	will automatically call Reconcile on the underlying controller when a resource changes, is
	deleted, etc.
*/

func NewController(mgr ctrl.Manager, logger logr.Logger) error {
	r := &Controller{
		Manager:           mgr,
		Logger:            logger.WithName("call"),
		gvk:               v1alpha1.GroupVersion.WithKind("Call"),
		executor:          executor.NewExecutor(mgr.GetConfig()),
		regionAnnotations: cmap.New(),
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Call{}).
		Named("call").
		Owns(&v1alpha1.VirtualObject{}, watchers.NewWatchWithRangeAnnotations(r, r.gvk)).
		Complete(r)
}
