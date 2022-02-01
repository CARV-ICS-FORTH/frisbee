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

package stop

import (
	"context"
	"reflect"
	"time"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	serviceutils "github.com/carv-ics-forth/frisbee/controllers/service/utils"
	"github.com/carv-ics-forth/frisbee/controllers/utils"
	"github.com/carv-ics-forth/frisbee/controllers/utils/expressions"
	"github.com/carv-ics-forth/frisbee/controllers/utils/lifecycle"
	"github.com/carv-ics-forth/frisbee/controllers/utils/scheduler"
	"github.com/carv-ics-forth/frisbee/pkg/executor"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=frisbee.io,resources=stops,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.io,resources=stops/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.io,resources=stops/finalizers,verbs=update

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;list;watch

// Controller reconciles a Cluster object.
type Controller struct {
	ctrl.Manager
	logr.Logger

	gvk schema.GroupVersionKind

	state lifecycle.Classifier

	serviceControl serviceutils.ServiceControlInterface

	// executor is used to run commands directly into containers
	executor executor.Executor
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		1: Load CR by name.
		------------------------------------------------------------------
	*/
	var cr v1alpha1.Stop

	var requeue bool
	result, err := utils.Reconcile(ctx, r, req, &cr, &requeue)

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
	var childJobs v1alpha1.ServiceList

	filters := []client.ListOption{
		client.InNamespace(req.Namespace),
		client.MatchingLabels{v1alpha1.LabelManagedBy: req.Name},
	}

	if err := r.GetClient().List(ctx, &childJobs, filters...); err != nil {
		return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "unable to list child services"))
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
	newStatus := calculateLifecycle(&cr, r.state)

	cr.Status = newStatus

	if err := utils.UpdateStatus(ctx, r, &cr); err != nil {
		// due to the multiple updates, it is possible for this function to
		// be in conflict. We fix this issue by re-queueing the request.
		// We also omit verbose error re
		// porting as to avoid polluting the output.
		return utils.RequeueAfter(time.Second)
	}

	/*
		If this object is suspended, we don't want to run any jobs, so we'll stop now.
		This is useful if something's broken with the job we're running, and we want to
		pause runs to investigate the cluster, without deleting the object.
	*/
	if cr.Spec.Suspend != nil && *cr.Spec.Suspend {
		r.Logger.Info("Stopper is suspended",
			"stopper", cr.GetName(),
			"reason", cr.Status.Reason,
			"message", cr.Status.Message,
		)

		return utils.Stop()
	}

	/*
		5: Clean up the controller from finished jobs
		------------------------------------------------------------------

		First, we'll try to clean up old jobs, so that we don't leave too many lying
		around.
	*/
	if newStatus.Phase == v1alpha1.PhaseSuccess {
		r.GetEventRecorderFor("").Event(&cr, corev1.EventTypeNormal,
			newStatus.Reason, "stop succeeded")

		r.Logger.Info("Cleaning up stop jobs",
			"stop", cr.GetName(),
			"successfulJobs", r.state.SuccessfulList(),
		)

		/*
			Remove cr children once the cr is successfully complete.
			We should not remove the cr descriptor itself, as we need to maintain its
			status for higher-entities like the Workflow.
		*/
		for _, job := range r.state.SuccessfulJobs() {
			utils.Delete(ctx, r, job)
		}

		return utils.Stop()
	}

	if newStatus.Phase == v1alpha1.PhaseFailed {
		r.Logger.Error(errors.New(newStatus.Reason), newStatus.Message)

		r.Logger.Info("Cleaning up stop jobs",
			"stop", cr.GetName(),
			"successfulJobs", r.state.SuccessfulList(),
			"runningJobs", r.state.RunningList(),
			"pendingJobs", r.state.PendingList(),
		)

		// Remove the non-failed components. Leave the failed jobs and system jobs for postmortem analysis.
		for _, job := range r.state.PendingJobs() {
			utils.Delete(ctx, r, job)
		}

		for _, job := range r.state.RunningJobs() {
			utils.Delete(ctx, r, job)
		}

		for _, job := range r.state.SuccessfulJobs() {
			utils.Delete(ctx, r, job)
		}

		// Block from creating further jobs
		suspend := true
		cr.Spec.Suspend = &suspend

		if err := utils.Update(ctx, r, &cr); err != nil {
			r.Error(err, "unable to suspend execution", "instance", cr.GetName())

			return utils.RequeueAfter(time.Second)
		}

		return utils.Stop()
	}

	/*
		6: Make the world matching what we want in our spec
		------------------------------------------------------------------

		Once we've updated our status, we can move on to ensuring that the status of
		the world matches what we want in our spec.

		We may delete the service, add a pod, or wait for existing pod to change its status.
	*/
	if newStatus.Phase == v1alpha1.PhaseUninitialized {
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
			return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "unable to construct job list"))
		}

		cr.Status.QueuedJobs = jobList
		cr.Status.ScheduledJobs = -1

		// Metrics-driven execution requires to set alerts on Grafana.
		if until := cr.Spec.Until; until != nil && until.HasMetricsExpr() {
			if err := expressions.SetAlert(&cr, until.Metrics); err != nil {
				return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "metrics expression error"))
			}
		}

		if schedule := cr.Spec.Schedule; schedule != nil && schedule.Conditions.HasMetricsExpr() {
			if err := expressions.SetAlert(&cr, schedule.Conditions.Metrics); err != nil {
				return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "metrics expression error"))
			}
		}

		if _, err := lifecycle.Pending(ctx, r, &cr, "submitting job requests"); err != nil {
			return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "status update"))
		}

		return utils.Stop()
	}

	/*
		If all jobs are scheduled, we have nothing else to do.
		If all jobs are scheduled but are not in the Running phase, they may be in the Pending phase.
		In both cases, we have nothing else to do but waiting for the next reconciliation cycle.
	*/
	nextJobToStop := cr.Status.ScheduledJobs + 1

	if newStatus.Phase == v1alpha1.PhaseRunning ||
		(cr.Spec.Until == nil && (nextJobToStop >= len(cr.Status.QueuedJobs))) {
		r.Logger.Info("All jobs are scheduled. Nothing else to do. Waiting for something to happen",
			"stop", cr.GetName(),
		)

		return utils.Stop()
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
		8: Construct our desired job and create it on Kubernetes
		------------------------------------------------------------------

		Since we have prepared these jobs at initialization, all we need is to get a pointer to the next job.
		We then use the Kubernetes client to exec the stopping command directly into the target container.
	*/

	serviceName, stopCmd := getJob(&cr, nextJobToStop)

	if err := r.stopJob(&cr, serviceName, stopCmd); err != nil {
		return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "unable to stop service %s", serviceName))
	}

	r.Logger.Info("Stop service",
		"job", cr.GetName(),
		"service", serviceName,
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
	cr.Status.ScheduledJobs = nextJobToStop
	cr.Status.LastScheduleTime = &metav1.Time{Time: time.Now()}

	return lifecycle.Pending(ctx, r, &cr, "some jobs are still pending")
}

/*
### Finalizers

*/

func (r *Controller) Finalizer() string {
	return "stops.frisbee.io/finalizer"
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
		Manager:  mgr,
		Logger:   logger.WithName("stop"),
		gvk:      v1alpha1.GroupVersion.WithKind("Stop"),
		executor: executor.NewExecutor(ctrl.GetConfigOrDie()),
	}

	r.serviceControl = serviceutils.NewServiceControl(r)

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Stop{}).
		Named("stop").
		Complete(r)
}
