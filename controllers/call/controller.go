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

package call

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/carv-ics-forth/frisbee/controllers/common/watchers"
	"github.com/carv-ics-forth/frisbee/pkg/expressions"
	"github.com/carv-ics-forth/frisbee/pkg/kubexec"
	"github.com/carv-ics-forth/frisbee/pkg/lifecycle"
	"github.com/carv-ics-forth/frisbee/pkg/scheduler"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	view *lifecycle.Classifier

	// executor is used to run commands directly into containers
	executor kubexec.Executor
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current view of the cluster closer to the desired view.
func (r *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		1: Load CR by name and extract the Desired State
		------------------------------------------------------------------
	*/
	var call v1alpha1.Call

	var requeue bool
	result, err := common.Reconcile(ctx, r, req, &call, &requeue)

	if requeue {
		return result, err
	}

	r.Logger.Info("-> Reconcile",
		"obj", client.ObjectKeyFromObject(&call),
		"phase", call.Status.Phase,
		"version", call.GetResourceVersion(),
	)

	defer func() {
		r.Logger.Info("<- Reconciler",
			"obj", client.ObjectKeyFromObject(&call),
			"phase", call.Status.Phase,
			"version", call.GetResourceVersion(),
		)
	}()

	/*
		2: Load CR's children and classify their current state (view)
		------------------------------------------------------------------
	*/
	if err := r.PopulateView(ctx, req.NamespacedName); err != nil {
		return lifecycle.Failed(ctx, r, &call, errors.Wrapf(err, "cannot get the cluster view for '%s'", req))
	}

	/*
		3: Use the view to update the CR's lifecycle.
		------------------------------------------------------------------
		The Update serves as "journaling" for the upcoming operations,
		and as a roadblock for stall (queued) requests.
	*/
	if r.updateLifecycle(&call) {
		if err := common.UpdateStatus(ctx, r, &call); err != nil {
			// due to the multiple updates, it is possible for this function to
			// be in conflict. We fix this issue by re-queueing the request.
			return common.RequeueAfter(r, req, time.Second)
		}
	}

	/*
		4: Make the world matching what we want in our spec.
		------------------------------------------------------------------
	*/

	if call.Spec.Suspend != nil && *call.Spec.Suspend {
		// If this object is suspended, we don't want to run any jobs, so we'll stop now.
		// This is useful if something's broken with the job we're running, and we want to
		// pause runs to investigate the cluster, without deleting the object.
		r.Logger.Info("Call has been suspend. Nothing else it scheduled.")

		return common.Stop(r, req)
	}

	log := r.Logger.WithValues("object", client.ObjectKeyFromObject(&call))

	switch call.Status.Phase {
	case v1alpha1.PhaseUninitialized:
		if err := r.Initialize(ctx, &call); err != nil {
			return lifecycle.Failed(ctx, r, &call, errors.Wrapf(err, "initialization error"))
		}

		return lifecycle.Pending(ctx, r, &call, "ready to start creating jobs.")

	case v1alpha1.PhasePending:
		//	If all jobs are scheduled but are not in the Running phase, they may be in the Pending phase.
		//	In both cases, we have nothing else to do but waiting for the next reconciliation cycle.
		if r.view.Count() >= len(call.Status.QueuedJobs) {
			r.Logger.Info("All jobs have been scheduled. Nothing else to do. ")
			return common.Stop(r, req)
		}

		// Check if the conditions are right to spawn a new job.
		hasJob, nextTick, err := scheduler.Schedule(log, &call, scheduler.Parameters{
			State:            *r.view,
			LastScheduleTime: call.Status.LastScheduleTime,
			ScheduleSpec:     call.Spec.Schedule,
			ExpectedTimeline: call.Status.ExpectedTimeline,
			JobName:          call.GetName(),
			ScheduledJobs:    call.Status.ScheduledJobs,
		})
		if err != nil {
			return lifecycle.Failed(ctx, r, &call, errors.Wrapf(err, "scheduling error"))
		}

		if !hasJob {
			// nothing to schedule
			if nextTick.IsZero() {
				return common.Stop(r, req)
			}

			// sleep until next tick
			return common.RequeueAfter(r, req, time.Until(nextTick))
		}

		// Fetch the next job from the queuing list, and submit it to Kubernetes.
		nextJobIndex := call.Status.ScheduledJobs + 1

		if nextJobIndex >= len(call.Status.QueuedJobs) {
			r.Logger.Error(errors.New("Ignore job as it is out of range compared to QueuedJobs"),
				"BadScheduling",
				"nextJobIndex", nextJobIndex,
				"queueJobs", len(call.Status.QueuedJobs),
				"viewedJobs", r.view.Count(),
				"jobList", r.view.ListAll(),
			)

			return common.Stop(r, req)
		}

		if err := r.runJob(ctx, &call, nextJobIndex); err != nil {
			return lifecycle.Failed(ctx, r, &call, errors.Wrapf(err, "cannot create job"))
		}

		// Update the scheduling information
		call.Status.ScheduledJobs = nextJobIndex
		call.Status.LastScheduleTime = metav1.Time{Time: time.Now()}

		return lifecycle.Pending(ctx, r, &call, fmt.Sprintf("Scheduled jobs: '%d/%d'",
			call.Status.ScheduledJobs+1, len(call.Spec.Services)))

	case v1alpha1.PhaseRunning:
		// Nothing to do. Just wait for something to happen.
		return common.Stop(r, req)

	case v1alpha1.PhaseSuccess:
		if err := r.HasSucceed(ctx, &call); err != nil {
			return common.RequeueAfter(r, req, time.Second)
		}

		return common.Stop(r, req)

	case v1alpha1.PhaseFailed:
		if err := r.HasFailed(ctx, &call); err != nil {
			return common.RequeueAfter(r, req, time.Second)
		}

		return common.Stop(r, req)
	}

	panic(errors.New("This should never happen"))
}

func (r *Controller) Initialize(ctx context.Context, call *v1alpha1.Call) error {
	/*
		We construct a list of job specifications based on the CR's template.
		This list is used by the execution step to create the actual job.
		If the template is invalid, it should be captured at this stage.
	*/
	jobList, err := r.buildJobQueue(ctx, call)
	if err != nil {
		return errors.Wrapf(err, "building joblist")
	}

	call.Status.QueuedJobs = jobList
	call.Status.ScheduledJobs = -1

	// Metrics-driven execution requires to set alerts on Grafana.
	if until := call.Spec.SuspendWhen; until != nil && until.HasMetricsExpr() {
		if err := expressions.SetAlert(ctx, call, until.Metrics); err != nil {
			return errors.Wrapf(err, "spec.suspendWhen")
		}
	}

	if schedule := call.Spec.Schedule; schedule != nil && schedule.Event.HasMetricsExpr() {
		if err := expressions.SetAlert(ctx, call, schedule.Event.Metrics); err != nil {
			return errors.Wrapf(err, "spec.schedule")
		}
	}

	if _, err := lifecycle.Pending(ctx, r, call, "submitting job requests"); err != nil {
		return errors.Wrapf(err, "status update")
	}

	return nil
}

func (r *Controller) PopulateView(ctx context.Context, req types.NamespacedName) error {
	r.view.Reset()

	var streamJobs v1alpha1.VirtualObjectList
	{
		if err := common.ListChildren(ctx, r.GetClient(), &streamJobs, req); err != nil {
			return errors.Wrapf(err, "cannot list children for '%s'", req)
		}

		for i, job := range streamJobs.Items {
			r.view.Classify(job.GetName(), &streamJobs.Items[i])
		}
	}

	return nil
}

func (r *Controller) HasSucceed(ctx context.Context, call *v1alpha1.Call) error {
	r.Logger.Info("CleanOnSuccess",
		"obj", client.ObjectKeyFromObject(call).String(),
		"sucessfulJobs", r.view.ListSuccessfulJobs(),
	)

	/*
		Remove cr children once the call is successfully complete.
		We should not remove the call descriptor itself, as we need to maintain its
		status for higher-entities like the Scenario.
	*/
	for _, job := range r.view.GetSuccessfulJobs() {
		common.Delete(ctx, r, job)
	}

	return nil
}

func (r *Controller) HasFailed(ctx context.Context, call *v1alpha1.Call) error {
	r.Logger.Info("!! JobError",
		"obj", client.ObjectKeyFromObject(call).String(),
		"reason", call.Status.Reason,
		"message", call.Status.Message,
	)

	// Remove the non-failed components. Leave the failed jobs and system jobs for postmortem analysis.
	for _, job := range r.view.GetPendingJobs() {
		common.Delete(ctx, r, job)
	}

	for _, job := range r.view.GetRunningJobs() {
		common.Delete(ctx, r, job)
	}

	// Block from creating further jobs
	suspend := true
	call.Spec.Suspend = &suspend

	r.Logger.Info("Suspended",
		"obj", client.ObjectKeyFromObject(call),
		"reason", call.Status.Reason,
		"message", call.Status.Message,
	)

	if call.GetDeletionTimestamp().IsZero() {
		r.GetEventRecorderFor(call.GetName()).Event(call, corev1.EventTypeNormal,
			"Suspended", call.Status.Lifecycle.Message)
	}

	// Update is needed since we modify the spec.suspend
	return common.Update(ctx, r, call)
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
	reconciler := &Controller{
		Manager:  mgr,
		Logger:   logger.WithName("call"),
		view:     &lifecycle.Classifier{},
		executor: kubexec.NewExecutor(mgr.GetConfig()),
	}

	gvk := v1alpha1.GroupVersion.WithKind("Call")

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Call{}).
		Named("call").
		Owns(&v1alpha1.VirtualObject{}, watchers.WatchWithRangeAnnotations(reconciler, gvk)).
		Complete(reconciler)
}
