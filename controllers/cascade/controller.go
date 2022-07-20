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

package cascade

import (
	"context"
	"reflect"
	"time"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/carv-ics-forth/frisbee/controllers/common/expressions"
	"github.com/carv-ics-forth/frisbee/controllers/common/lifecycle"
	"github.com/carv-ics-forth/frisbee/controllers/common/scheduler"
	"github.com/carv-ics-forth/frisbee/controllers/common/watchers"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=frisbee.dev,resources=cascades,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.dev,resources=cascades/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.dev,resources=cascades/finalizers,verbs=update

// Controller reconciles a Cascade object.
type Controller struct {
	ctrl.Manager
	logr.Logger

	state lifecycle.Classifier
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cascade closer to the desired state.
func (r *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		1: Load CR by name.
		------------------------------------------------------------------
	*/
	var cr v1alpha1.Cascade

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

		To fully update our status, we'll need to list all child objects in this namespace that belong to this CR.

		As our number of services increases, looking these up can become quite slow as we have to filter through all
		of them. For a more efficient lookup, these services will be indexed locally on the controller's name.
		A jobOwnerKey field is added to the cached job objects, which references the owning controller.
		Check how we configure the manager to actually index this field.
	*/
	var childJobs v1alpha1.ChaosList

	if err := common.ListChildren(ctx, r, &childJobs, req.NamespacedName); err != nil {
		return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "unable to list children for '%s'", req.NamespacedName))
	}

	/*
		3: Classify CR's components.
		------------------------------------------------------------------

		Once we have all the jobs we own, we'll split them into active, successful,
		and failed jobs, keeping track of the most recent run so that we can record it
		in status.  Remember, status should be able to be reconstituted from the state
		of the world, so it's generally not a good idea to read from the status of the
		root object.  Instead, you should reconstruct it every run.  That's what we'll
		do here.

		To relief the garbage collector, we use a root structure that we reset at every reconciliation cycle.
	*/
	r.state.Reset()

	for i, job := range childJobs.Items {
		r.state.Classify(job.GetName(), &childJobs.Items[i])
	}

	/*
		4: Update the CR status using the data we've gathered
		------------------------------------------------------------------

		Just like before, we use our client.  To specifically update the status
		subresource, we'll use the `Status` part of the client, with the `Update`
		method.
	*/
	cr.SetReconcileStatus(r.calculateLifecycle(&cr))

	if err := common.UpdateStatus(ctx, r, &cr); err != nil {
		r.Info("Reschedule.", "object", cr.GetName(), "UpdateStatusErr", err)
		return common.RequeueAfter(time.Second)
	}

	/*
		If this object is suspended, we don't want to run any jobs, so we'll stop now.
		This is useful if something's broken with the job we're running, and we want to
		pause runs to investigate the cascade, without deleting the object.
	*/
	if cr.Spec.Suspend != nil && *cr.Spec.Suspend {
		r.Logger.Info("Suspended",
			"cascade", cr.GetName(),
			"reason", cr.Status.Reason,
			"message", cr.Status.Message,
		)

		return common.Stop()
	}

	/*
		5: Clean up the controller from finished jobs
		------------------------------------------------------------------

		First, we'll try to clean up old jobs, so that we don't leave too many lying
		around.
	*/
	if cr.Status.Phase.Is(v1alpha1.PhaseSuccess) {
		r.GetEventRecorderFor("").Event(&cr, corev1.EventTypeNormal,
			cr.Status.Lifecycle.Reason, cr.Status.Lifecycle.Message)

		r.Logger.Info("Cleaning up chaos jobs",
			"cascade", cr.GetName(),
			"activeJobs", r.state.ListPendingJobs(),
			"successfulJobs", r.state.ListSuccessfulJobs(),
		)

		/*
			Remove cr children once the cr is successfully complete.
			We should not remove the cr descriptor itself, as we need to maintain its
			status for higher-entities like the Scenario.
		*/
		for _, job := range r.state.GetSuccessfulJobs() {
			common.Delete(ctx, r, job)
		}

		return common.Stop()
	}

	if cr.Status.Phase.Is(v1alpha1.PhaseFailed) {
		r.Logger.Error(errors.New(cr.Status.Lifecycle.Reason), cr.Status.Lifecycle.Message)

		r.Logger.Info("Cleaning up chaos jobs",
			"cascade", cr.GetName(),
			"successfulJobs", r.state.ListSuccessfulJobs(),
			"activeJobs", r.state.ListPendingJobs(),
		)

		// Remove the non-failed components. Leave the failed jobs and system jobs for postmortem analysis.
		for _, job := range r.state.GetPendingJobs() {
			common.Delete(ctx, r, job)
		}

		for _, job := range r.state.GetRunningJobs() {
			common.Delete(ctx, r, job)
		}

		for _, job := range r.state.GetSuccessfulJobs() {
			common.Delete(ctx, r, job)
		}

		// Block from creating further jobs
		suspend := true
		cr.Spec.Suspend = &suspend

		if err := common.Update(ctx, r, &cr); err != nil {
			r.Error(err, "unable to suspend execution", "instance", cr.GetName())

			return common.RequeueAfter(time.Second)
		}

		return common.Stop()
	}

	/*
		6: Make the world matching what we want in our spec
		------------------------------------------------------------------

		Once we've updated our status, we can move on to ensuring that the status of
		the world matches what we want in our spec.

		We may delete the service, add a pod, or wait for existing pod to change its status.
	*/
	if cr.Status.Phase.Is(v1alpha1.PhaseUninitialized) {
		/*
			We construct a list of job specifications based on the CR's template.
			This list is used by the execution step to create the actual job.
			If the template is invalid, it should be captured at this stage.

			To specifically update the status subresource, we'll use the `Status` part of the client, with the `ServiceUpdate`
			method. The status subresource ignores changes to spec, so it's less likely to conflict
			with any other updates, and can have separate permissions.
		*/
		jobList, err := r.constructJobSpecList(ctx, &cr)
		if err != nil {
			return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "cannot build joblist"))
		}

		cr.Status.QueuedJobs = jobList
		cr.Status.ScheduledJobs = -1

		// Metrics-driven execution requires to set alerts on Grafana.
		if until := cr.Spec.Until; until != nil && until.HasMetricsExpr() {
			if err := expressions.SetAlert(ctx, &cr, until.Metrics); err != nil {
				return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "spec.until"))
			}
		}

		if schedule := cr.Spec.Schedule; schedule != nil && schedule.Event.HasMetricsExpr() {
			if err := expressions.SetAlert(ctx, &cr, schedule.Event.Metrics); err != nil {
				return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "spec.schedule"))
			}
		}

		if _, err := lifecycle.Pending(ctx, r, &cr, "submitting job requests"); err != nil {
			return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "status update"))
		}

		return common.Stop()
	}

	/*
		If all jobs are scheduled, we have nothing else to do.
		If all jobs are scheduled but are not in the Running phase, they may be in the Pending phase.
		In both cases, we have nothing else to do but waiting for the next reconciliation cycle.
	*/
	nextExpectedJob := cr.Status.ScheduledJobs + 1

	if cr.Status.Phase.Is(v1alpha1.PhaseRunning) ||
		(cr.Spec.Until == nil && (nextExpectedJob >= len(cr.Status.QueuedJobs))) {
		r.Logger.Info("All jobs are scheduled. Nothing else to do. Waiting for something to happen",
			"cascade", cr.GetName(),
		)

		return common.Stop()
	}

	/*
		7: Get the next scheduled run
		------------------------------------------------------------------

		If we're not paused, we'll need to calculate the next scheduled run, and whether
		we've got a run that we haven't processed yet  (or anything we missed).

		If we've missed a run, and we're still within the deadline to start it, we'll need to run a job.
	*/
	if hasJob, requeue, err := scheduler.Schedule(ctx, r, &cr, cr.Spec.Schedule, cr.Status.LastScheduleTime, r.state); !hasJob {
		return requeue, err
	}

	/*
		8: Construct our desired job  and create it on the cascade
		------------------------------------------------------------------

		We need to construct a job based on our cascade's template. Since we have prepared these jobs at
		initialization, all we need is to get a pointer to the next job.
	*/
	nextJob := getJob(&cr, nextExpectedJob)

	if err := common.Create(ctx, r, &cr, nextJob); err != nil {
		return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "cannot create job"))
	}

	r.Logger.Info("Create cascaded chaos job",
		"cascade", cr.GetName(),
		"chaos", nextJob.GetName(),
	)

	/*
		9: Avoid double actions
		------------------------------------------------------------------

		If this process restarts at this point (after posting a job, but
		before updating the status), then we might try to start the job on
		the next time.  Actually, if we re-list the Jobs on the next cycle
		we might not see our own status update, and then post one again.
		So, we need to use the job name as a lock to prevent us from making the job twice.
	*/
	cr.Status.ScheduledJobs = nextExpectedJob
	cr.Status.LastScheduleTime = &metav1.Time{Time: time.Now()}

	return lifecycle.Pending(ctx, r, &cr, "some jobs are still pending")
}

/*
### Finalizers

*/

func (r *Controller) Finalizer() string {
	return "cascades.frisbee.dev/finalizer"
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

	Additionally, We'll inform the manager that this controller owns some resources, so that it
	will automatically call Reconcile on the underlying controller when a resource changes, is
	deleted, etc.
*/

func NewController(mgr ctrl.Manager, logger logr.Logger) error {
	r := &Controller{
		Manager: mgr,
		Logger:  logger.WithName("cascade"),
	}

	gvk := v1alpha1.GroupVersion.WithKind("Cascade")

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Cascade{}).
		Named("cascade").
		Owns(&v1alpha1.Chaos{}, watchers.Watch(r, gvk)).
		Complete(r)
}
