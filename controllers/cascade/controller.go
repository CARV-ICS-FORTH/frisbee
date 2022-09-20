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
	"fmt"
	"reflect"
	"time"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/carv-ics-forth/frisbee/controllers/common/scheduler"
	"github.com/carv-ics-forth/frisbee/controllers/common/watchers"
	"github.com/carv-ics-forth/frisbee/pkg/expressions"
	"github.com/carv-ics-forth/frisbee/pkg/lifecycle"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

	view *lifecycle.Classifier
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current view of the cascade closer to the desired view.
func (r *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		1: Load CR by name and extract the Desired State
		------------------------------------------------------------------
	*/
	var cascade v1alpha1.Cascade

	var requeue bool
	result, err := common.Reconcile(ctx, r, req, &cascade, &requeue)

	if requeue {
		return result, err
	}

	r.Logger.Info("-> Reconcile",
		"obj", client.ObjectKeyFromObject(&cascade),
		"phase", cascade.Status.Phase,
		"version", cascade.GetResourceVersion(),
	)

	defer func() {
		r.Logger.Info("<- Reconcile",
			"obj", client.ObjectKeyFromObject(&cascade),
			"phase", cascade.Status.Phase,
			"version", cascade.GetResourceVersion(),
		)
	}()

	/*
		2: Load CR's children and classify their current state (view)
		------------------------------------------------------------------
	*/
	if err := r.PopulateView(ctx, req.NamespacedName); err != nil {
		return lifecycle.Failed(ctx, r, &cascade, errors.Wrapf(err, "cannot populate view for '%s'", req))
	}

	/*
		3: Use the view to update the CR's lifecycle.
		------------------------------------------------------------------
		The Update serves as "journaling" for the upcoming operations,
		and as a roadblock for stall (queued) requests.
	*/
	r.calculateLifecycle(&cascade)

	if err := common.UpdateStatus(ctx, r, &cascade); err != nil {
		// due to the multiple updates, it is possible for this function to
		// be in conflict. We fix this issue by re-queueing the request.
		return common.RequeueAfter(time.Second)
	}

	/*
		4: Make the world matching what we want in our spec.
		------------------------------------------------------------------
	*/

	if cascade.Spec.Suspend != nil && *cascade.Spec.Suspend {
		// If this object is suspended, we don't want to run any jobs, so we'll stop now.
		// This is useful if something's broken with the job we're running, and we want to
		// pause runs to investigate the cluster, without deleting the object.
		return common.Stop()
	}

	log := r.Logger.WithValues("object", client.ObjectKeyFromObject(&cascade))

	switch cascade.Status.Phase {
	case v1alpha1.PhaseSuccess:
		if err := r.HasSucceed(ctx, &cascade); err != nil {
			return common.RequeueAfter(time.Second)
		}

		return common.Stop()

	case v1alpha1.PhaseFailed:
		if err := r.HasFailed(ctx, &cascade); err != nil {
			return common.RequeueAfter(time.Second)
		}

		return common.Stop()

	case v1alpha1.PhaseRunning:
		// Nothing to do. Just wait for something to happen.
		log.Info(".. Awaiting", cascade.Status.Reason, cascade.Status.Message)

		return common.Stop()

	case v1alpha1.PhaseUninitialized:
		if err := r.Initialize(ctx, &cascade); err != nil {
			return lifecycle.Failed(ctx, r, &cascade, errors.Wrapf(err, "initialization error"))
		}

		return lifecycle.Pending(ctx, r, &cascade, "ready to start creating jobs.")

	case v1alpha1.PhasePending:
		nextJob := cascade.Status.ScheduledJobs + 1

		//	If all jobs are scheduled but are not in the Running phase, they may be in the Pending phase.
		//	In both cases, we have nothing else to do but waiting for the next reconciliation cycle.
		if cascade.Spec.Until == nil && (nextJob >= len(cascade.Status.QueuedJobs)) {
			log.Info(".. Awaiting", cascade.Status.Reason, cascade.Status.Message)

			return common.Stop()
		}

		// Get the next scheduled job
		{
			hasJob, nextTick, err := scheduler.Schedule(log, &cascade, scheduler.Parameters{
				ScheduleSpec:     cascade.Spec.Schedule,
				LastScheduled:    cascade.Status.LastScheduleTime,
				ExpectedTimeline: cascade.Status.Timeline,
				State:            r.view,
			})
			if err != nil {
				return lifecycle.Failed(ctx, r, &cascade, errors.Wrapf(err, "scheduling error"))
			}
			if !hasJob {
				return common.RequeueAfter(time.Until(nextTick))
			}
		}

		// Build the job in kubernetes
		if err := r.runJob(ctx, &cascade, nextJob); err != nil {
			return lifecycle.Failed(ctx, r, &cascade, errors.Wrapf(err, "cannot create job"))
		}

		// Update the scheduling information
		cascade.Status.ScheduledJobs = nextJob
		cascade.Status.LastScheduleTime = &metav1.Time{Time: time.Now()}

		return lifecycle.Pending(ctx, r, &cascade, fmt.Sprintf("Scheduled jobs: '%d/%d'",
			cascade.Status.ScheduledJobs+1, cascade.Spec.MaxInstances))
	}

	panic(errors.New("This should never happen"))
}

func (r *Controller) Initialize(ctx context.Context, cascade *v1alpha1.Cascade) error {
	/*
		We construct a list of job specifications based on the CR's template.
		This list is used by the execution step to create the actual job.
		If the template is invalid, it should be captured at this stage.
	*/
	jobList, err := r.constructJobSpecList(ctx, cascade)
	if err != nil {
		return errors.Wrapf(err, "building joblist")
	}

	cascade.Status.QueuedJobs = jobList
	cascade.Status.ScheduledJobs = -1

	// Metrics-driven execution requires to set alerts on Grafana.
	if until := cascade.Spec.Until; until != nil && until.HasMetricsExpr() {
		if err := expressions.SetAlert(ctx, cascade, until.Metrics); err != nil {
			return errors.Wrapf(err, "spec.until")
		}
	}

	if schedule := cascade.Spec.Schedule; schedule != nil && schedule.Event.HasMetricsExpr() {
		if err := expressions.SetAlert(ctx, cascade, schedule.Event.Metrics); err != nil {
			return errors.Wrapf(err, "spec.schedule")
		}
	}

	if _, err := lifecycle.Pending(ctx, r, cascade, "submitting job requests"); err != nil {
		return errors.Wrapf(err, "status update")
	}

	return nil
}

func (r *Controller) PopulateView(ctx context.Context, req types.NamespacedName) error {
	r.view.Reset()

	var chaosJobs v1alpha1.ChaosList
	{
		if err := common.ListChildren(ctx, r, &chaosJobs, req); err != nil {
			return errors.Wrapf(err, "cannot list children for '%s'", req)
		}

		for i, job := range chaosJobs.Items {
			r.view.Classify(job.GetName(), &chaosJobs.Items[i])
		}
	}

	return nil
}

func (r *Controller) HasSucceed(ctx context.Context, cascade *v1alpha1.Cascade) error {
	r.Logger.Info("CleanOnSuccess",
		"obj", client.ObjectKeyFromObject(cascade).String(),
		"successfulJobs", r.view.ListSuccessfulJobs(),
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

func (r *Controller) HasFailed(ctx context.Context, cascade *v1alpha1.Cascade) error {
	r.Logger.Error(errors.Errorf(cascade.Status.Message), "!! "+cascade.Status.Reason,
		"obj", client.ObjectKeyFromObject(cascade).String())

	// Remove the non-failed components. Leave the failed jobs and system jobs for postmortem analysis.
	for _, job := range r.view.GetPendingJobs() {
		common.Delete(ctx, r, job)
	}

	for _, job := range r.view.GetRunningJobs() {
		common.Delete(ctx, r, job)
	}

	// Block from creating further jobs
	suspend := true
	cascade.Spec.Suspend = &suspend

	r.Logger.Info("Suspended",
		"obj", client.ObjectKeyFromObject(cascade),
		"reason", cascade.Status.Reason,
		"message", cascade.Status.Message,
	)

	if cascade.GetDeletionTimestamp().IsZero() {
		r.GetEventRecorderFor(cascade.GetName()).Event(cascade, corev1.EventTypeNormal,
			"Suspended", cascade.Status.Lifecycle.Message)
	}

	// Update is needed since we modify the spec.suspend
	return common.Update(ctx, r, cascade)
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
	controller := &Controller{
		Manager: mgr,
		Logger:  logger.WithName("cascade"),
		view:    &lifecycle.Classifier{},
	}

	gvk := v1alpha1.GroupVersion.WithKind("Cascade")

	var (
		cascade v1alpha1.Cascade
		chaos   v1alpha1.Chaos
	)

	return ctrl.NewControllerManagedBy(mgr).
		For(&cascade).
		Named("cascade").
		Owns(&chaos, watchers.Watch(controller, gvk)).
		Complete(controller)
}
