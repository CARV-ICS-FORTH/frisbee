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

package cluster

import (
	"context"
	"reflect"
	"time"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/utils"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=frisbee.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.io,resources=clusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.io,resources=clusters/finalizers,verbs=update

// +kubebuilder:rbac:groups=frisbee.io,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.io,resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.io,resources=services/finalizers,verbs=update

const (
	jobOwnerKey = ".metadata.controller"
)

// Controller reconciles a Cluster object.
type Controller struct {
	ctrl.Manager
	logr.Logger

	state utils.LifecycleClassifier
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		1: Load CR by name.
		------------------------------------------------------------------
	*/
	var cr v1alpha1.Cluster

	var requeue bool
	result, err := utils.Reconcile(ctx, r, req, &cr, &requeue)

	if requeue {
		return result, errors.Wrapf(err, "initialization error")
	}

	r.Logger.Info("-> Reconcile",
		"kind", reflect.TypeOf(cr),
		"name", cr.GetName(),
		"lifecycle", cr.Status.Phase,
		"deleted", !cr.GetDeletionTimestamp().IsZero(),
		"version", cr.GetResourceVersion(),
	)

	defer func() {
		r.Logger.Info("<- Reconcile",
			"kind", reflect.TypeOf(cr),
			"name", cr.GetName(),
			"lifecycle", cr.Status.Phase,
			"deleted", !cr.GetDeletionTimestamp().IsZero(),
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
		client.MatchingFields{jobOwnerKey: req.Name},
	}

	if err := r.GetClient().List(ctx, &childJobs, filters...); err != nil {
		return utils.Failed(ctx, r, &cr, errors.Wrapf(err, "unable to list child services"))
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

	var mostRecentTime *time.Time // find the last run so we can update the status

	for _, job := range childJobs.Items {
		r.state.Classify(job.GetName(), job.DeepCopy())

		scheduledTimeForJob := &job.CreationTimestamp.Time

		if !scheduledTimeForJob.IsZero() {
			if mostRecentTime == nil {
				mostRecentTime = scheduledTimeForJob
			} else if mostRecentTime.Before(*scheduledTimeForJob) {
				mostRecentTime = scheduledTimeForJob
			}
		}
	}

	/*
		4: Update the CR status using the data we've gathered
		------------------------------------------------------------------

		Just like before, we use our client.  To specifically update the status
		subresource, we'll use the `Status` part of the client, with the `Update`
		method.
	*/
	if mostRecentTime != nil {
		cr.Status.LastScheduleTime = &metav1.Time{Time: *mostRecentTime}
	} else {
		cr.Status.LastScheduleTime = nil
	}

	newStatus := calculateLifecycle(&cr, r.state)

	cr.Status.Lifecycle = newStatus

	if err := utils.UpdateStatus(ctx, r, &cr); err != nil {
		// due to the multiple updates, it is possible for this function to
		// be in conflict. We fix this issue by re-queueing the request.
		// We also omit verbose error re
		// porting as to avoid polluting the output.
		return utils.Requeue()
	}

	/*
		5: Clean up the controller from finished jobs
		------------------------------------------------------------------

		First, we'll try to clean up old jobs, so that we don't leave too many lying
		around.
	*/
	if newStatus.Phase == v1alpha1.PhaseSuccess {
		r.GetEventRecorderFor("").Event(&cr, corev1.EventTypeNormal,
			newStatus.Reason, "cluster succeeded")

		// Remove cr children once the cr is successfully complete.
		// We should not remove the cr descriptor itself, as we need to maintain its
		// status for higher-entities like the Workflow.
		for _, job := range r.state.SuccessfulJobs() {
			utils.Delete(ctx, r, job)
		}

		return utils.Stop()
	}

	if newStatus.Phase == v1alpha1.PhaseFailed {
		r.GetEventRecorderFor("").Event(&cr, corev1.EventTypeWarning, newStatus.Reason, newStatus.Message)

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
		jobList, err := constructJobSpecList(ctx, r, &cr)
		if err != nil {
			return utils.Failed(ctx, r, &cr, errors.Wrapf(err, "unable to construct job list"))
		}

		cr.Status.Expected = jobList
		cr.Status.LastScheduleJob = -1

		if _, err := utils.Pending(ctx, r, &cr, "submitting job requests"); err != nil {
			return utils.Failed(ctx, r, &cr, errors.Wrapf(err, "status update"))
		}

		return utils.Stop()
	}

	/*
		If all actions are executed, just wait for them to complete.
	*/
	if newStatus.Phase == v1alpha1.PhaseRunning {
		return utils.Stop()
	}

	/*
		If this object is suspended, we don't want to run any jobs, so we'll stop now.
		This is useful if something's broken with the job we're running, and we want to
		pause runs to investigate or putz with the cluster, without deleting the object.
	*/
	if cr.Spec.Suspend != nil && *cr.Spec.Suspend {
		r.Logger.Info("Not starting job because the cluster is suspended",
			"cluster", cr.GetName())

		return utils.Stop()
	}

	/*
		7: Get the next scheduled run
		------------------------------------------------------------------

		If we're not paused, we'll need to calculate the next scheduled run, and whether
		we've got a run that we haven't processed yet  (or anything we missed).

		If we've missed a run, and we're still within the deadline to start it, we'll need to run a job.
	*/
	missedRun, nextRun, err := utils.GetNextScheduleTime(&cr, cr.Spec.Schedule, cr.Status.LastScheduleTime)

	if err != nil {
		r.GetEventRecorderFor("").Event(&cr, corev1.EventTypeWarning,
			err.Error(), "unable to figure execution schedule")

		// we don't really care about re-queuing until we get an update that
		// fixes the schedule, so don't return an error.
		return utils.Stop()
	}

	r.Logger.Info("next run", "missed ", missedRun, "next", nextRun)

	if missedRun.IsZero() {
		if nextRun.IsZero() {
			r.Logger.Info("scheduling is complete.")

			return utils.Stop()
		}

		r.Logger.Info("no upcoming scheduled times, sleeping until", "next", nextRun)

		return utils.RequeueAfter(time.Until(nextRun))
	}

	if schedule := cr.Spec.Schedule; schedule != nil {
		// if there is a schedule defined, make sure we're not too late to start the run
		tooLate := false

		if deadline := schedule.StartingDeadlineSeconds; deadline != nil {
			tooLate = missedRun.Add(time.Duration(*deadline) * time.Second).Before(time.Now())
		}

		if tooLate {
			return utils.Failed(ctx, r, &cr, errors.New("scheduling violation"))
		}
	}

	/*
		8: Construct our desired job  and create it on the cluster
		------------------------------------------------------------------

		We need to construct a job based on our Cluster's template. Since we have prepared these jobs at
		initialization, all we need is to get a pointer to the next job.
	*/

	nextExpectedJob := cr.Status.LastScheduleJob + 1
	nextJob := getJob(r, &cr, nextExpectedJob)

	if err := utils.CreateUnlessExists(ctx, r, nextJob); err != nil {
		return utils.Failed(ctx, r, &cr, errors.Wrapf(err, "cannot create job"))
	}

	r.Logger.Info("Create clustered job",
		"cluster", cr.GetName(),
		"service", nextJob.GetName(),
	)

	/*
		8: Avoid double actions
		------------------------------------------------------------------

		If this process restarts at this point (after posting a job, but
		before updating the status), then we might try to start the job on
		the next time.  Actually, if we re-list the Jobs on the next cycle
		we might not see our own status update, and then post one again.
		So, we need to use the job name as a lock to prevent us from making the job twice.
	*/
	cr.Status.LastScheduleJob = nextExpectedJob

	return utils.Pending(ctx, r, &cr, "some jobs are still pending")
}

/*
### Finalizers

*/

func (r *Controller) Finalizer() string {
	return "clusters.frisbee.io/finalizer"
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
	Finally, we'll update our setup.  In order to allow to quickly look up Services by their owner,
	we'll need an index.  We declare an index key that we can later use with the client as a pseudo-field name,
	and then describe how to extract the indexed value from the Service object.
	The indexer will automatically take care of namespaces for us, so we just have to extract the
	owner name if the Service has a Cluster owner.

	Additionally, we'll inform the manager that this controller owns some Services, so that it
	will automatically call Reconcile on the underlying Cluster when a Service changes, is
	deleted, etc.
*/

var controllerKind = v1alpha1.GroupVersion.WithKind("Cluster")

func NewController(mgr ctrl.Manager, logger logr.Logger) error {
	// FieldIndexer knows how to index over a particular "field" such that it
	// can later be used by a field selector.
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1alpha1.Service{}, jobOwnerKey,
		func(rawObj client.Object) []string {
			// grab the job object, extract the owner...
			job := rawObj.(*v1alpha1.Service)

			if !utils.IsManagedByThisController(job, controllerKind) {
				return nil
			}

			owner := metav1.GetControllerOf(job)

			// ...and if so, return it
			return []string{owner.Name}
		}); err != nil {
		return err
	}

	r := &Controller{
		Manager: mgr,
		Logger:  logger.WithName("cluster"),
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Cluster{}).
		Named("cluster").
		// WithEventFilter(r.Filters()).
		Owns(&v1alpha1.Service{}, builder.WithPredicates(r.WatchServices())).
		Complete(r)
}
