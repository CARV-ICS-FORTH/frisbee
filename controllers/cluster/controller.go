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
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/controllers/common/lifecycle"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
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

var (
	jobOwnerKey = ".metadata.controller"
)

// Controller reconciles a Cluster object.
type Controller struct {
	ctrl.Manager
	logr.Logger
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		### 1: Load the cluster by name.
	*/
	var cluster v1alpha1.Cluster

	var ret bool
	result, err := common.Reconcile(ctx, r, req, &cluster, &ret)
	if ret {
		return result, err
	}

	r.Logger.Info("-> Reconcile",
		"kind", reflect.TypeOf(cluster),
		"name", cluster.GetName(),
		"lifecycle", cluster.Status.Phase,
		"deleted", !cluster.GetDeletionTimestamp().IsZero(),
		"epoch", cluster.GetResourceVersion(),
	)

	defer func() {
		r.Logger.Info("<- Reconcile",
			"kind", reflect.TypeOf(cluster),
			"name", cluster.GetName(),
			"lifecycle", cluster.Status.Phase,
			"deleted", !cluster.GetDeletionTimestamp().IsZero(),
			"epoch", cluster.GetResourceVersion(),
		)
	}()

	/*
		### 2:  Construct jobs based on our Cluster's template.

		Since all the jobs of a cluster are known in advance (their number, their parameters, and so on),
		we pre-create a list of the specification of each service.

		To specifically update the status subresource, we'll use the `Status` part of the client, with the `Update`
		method. The status subresource ignores changes to spec, so it's less likely to conflict
		with any other updates, and can have separate permissions.
	*/
	if cluster.Status.Expected == nil {
		jobList, err := constructJobSpecList(ctx, r, &cluster)
		if err != nil {
			return lifecycle.Failed(ctx, r, &cluster, errors.Wrapf(err, "unable to construct job list"))
		}

		cluster.Status.Expected = jobList

		if _, err := common.UpdateStatus(ctx, r, &cluster); err != nil {
			r.Logger.Error(err, "update status error")
			return lifecycle.Failed(ctx, r, &cluster, errors.Wrapf(err, "status update"))
		}
	}

	/*
		### 3: Load the cluster's components.

		To fully update our status, we'll need to list all child services in this namespace that belong to this Cluster.
		Similarly to Get, we can use the List method to list the child services.

		As our number of services increases, looking these up can become quite slow as we have to filter through all
		of them. For a more efficient lookup, these services will be indexed locally on the controller's name.
		A jobOwnerKey field is added to the cached job objects, which references the owning controller.
		Check how we configure the manager to actually index this field.
	*/

	var childJobs v1alpha1.ServiceList

	filters := []client.ListOption{
		client.InNamespace(req.Namespace),
		client.MatchingFields{jobOwnerKey: req.Name},
	}

	if err := r.GetClient().List(ctx, &childJobs, filters...); err != nil {
		return lifecycle.Failed(ctx, r, &cluster, errors.Wrapf(err, "unable to list child services"))
	}

	/*
		### 4: Classify the cluster's components.

		Once we have all the jobs we own, we'll split them into active, successful, and failed services, keeping track
		of the most recent run so that we can record it in status.
		Remember, status should be able to be reconstituted from the state 	of the world, so it's generally not a good
		idea to read from the status of the root object.
		Instead, you should reconstruct it every run.  That's what we'll do here.


		We can check if a service is "finished" and whether it succeeded or failed using Frisbee Phases.
		We'll put that logic in a helper to make our code cleaner.
	*/

	var activeJobs v1alpha1.SList
	var successfulJobs v1alpha1.SList
	var failedJobs v1alpha1.SList
	var mostRecentTime *time.Time // find the last run so we can update the status

	for i, job := range childJobs.Items {
		switch job.Status.Lifecycle.Phase {
		case v1alpha1.PhaseSuccess:
			successfulJobs = append(successfulJobs, childJobs.Items[i])

		case v1alpha1.PhaseFailed:
			failedJobs = append(failedJobs, childJobs.Items[i])

		default:
			activeJobs = append(activeJobs, childJobs.Items[i])
		}

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
		### 5: Calculate and update the cluster status

		Using the date we've gathered, we'll update the status of our CRD.
		Depending on the outcome, the execution may proceed or terminate.
		For example, if all services are successfully complete the cluster will terminate successfully.
		If there is a failed job, the cluster will fail itself.
	*/
	if mostRecentTime != nil {
		cluster.Status.LastScheduleTime = &metav1.Time{Time: *mostRecentTime}
	} else {
		cluster.Status.LastScheduleTime = nil
	}

	cluster.Status.Active = activeJobs

	newStatus := calculateLifecycle(&cluster, activeJobs, successfulJobs, failedJobs)
	cluster.Status.Lifecycle = newStatus

	if _, err := common.UpdateStatus(ctx, r, &cluster); err != nil {
		r.Logger.Error(err, "update status error")

		return lifecycle.Failed(ctx, r, &cluster, errors.Wrapf(err, "status update"))
	}

	/*
		### 6: Decide the next steps based on the update status.

		Once we've updated our status, we can move on to ensuring that the status of
		the world matches what we want in our spec.
		We may delete the cluster, add new components, or wait for existing components to change their status.
	*/

	switch newStatus.Phase {
	case v1alpha1.PhaseRunning:
		return common.Stop()

	case v1alpha1.PhaseSuccess:
		// Delete the cluster when all its jobs are doned
		if err := r.GetClient().Delete(ctx, &cluster); client.IgnoreNotFound(err) != nil {
			return lifecycle.Failed(ctx, r, &cluster, errors.Wrapf(err, "cluster deletion"))
		}

		return common.Stop()

	case v1alpha1.PhaseFailed:
		r.Logger.Info("Oracle has failed for ",
			"cluster", cluster.GetName(),
			"reason", newStatus.Reason,
		)

		return common.Stop()
	}

	/*
		### 7: Check if we're suspended

		If this object is suspended, we don't want to run any jobs, so we'll stop the reconciliation cycle now.
		This is useful if something's broken with the job we're running, and we want to
		pause runs to investigate or putz with the cluster, without deleting the object.
	*/
	if cluster.Spec.Suspend != nil && *cluster.Spec.Suspend {
		r.Logger.Info("cluster suspended, skipping")

		return common.Stop()
	}

	/*
		### 8:  Get the next scheduled run

		If we're not paused, we'll need to calculate the next scheduled run, and whether
		we've got a run that we haven't processed yet (or anything we missed).

		We'll calculate the next scheduled time using the helpful cron library.
		We'll start calculating appropriate times from our last run, or the creation
		of the Service if we can't find a last run.

		If there are too many missed runs, and we don't have any deadlines set, we'll
		bail so that we don't cause issues on controller restarts or wedges.
		Otherwise, we'll just return the missed runs (of which we'll just use the latest),
		and the next run, so that we can know when it's time to reconcile again.
	*/

	// used as index in Expected list for the next job. because of the len() semantics,
	// the index already shows the next position.
	nextExpectedJob := len(childJobs.Items)

	missedRun, nextRun, err := getNextSchedule(&cluster, time.Now())
	if err != nil {
		r.Logger.Error(err, "unable to figure out execution schedule")

		// we don't really care about requeuing until we get an update that
		// fixes the schedule, so don't return an error
		return common.Stop()
	}

	r.Logger.Info("next run", "missed ", missedRun, "next", nextRun, " job ", nextExpectedJob)

	if missedRun.IsZero() {
		if nextRun.IsZero() {
			r.Logger.Info("scheduling is complete.")

			return common.Stop()
		}

		r.Logger.Info("no upcoming scheduled times, sleeping until next")

		return common.RequeueAfter(nextRun.Sub(time.Now()))
	}

	if schedule := cluster.Spec.Schedule; schedule != nil {
		// if there is a schedule defined, make sure we're not too late to start the run
		tooLate := false

		if deadline := schedule.StartingDeadlineSeconds; deadline != nil {
			tooLate = missedRun.Add(time.Duration(*deadline) * time.Second).Before(time.Now())
		}

		if tooLate {
			return lifecycle.Failed(ctx, r, &cluster, errors.New("scheduling violation"))
		}
	}

	/*
		### 9  Run a new job if it's on schedule, not past the deadline, and not blocked by our concurrency policy
	*/
	nextJob := constructJob(&cluster, nextExpectedJob)

	// if the next reconciliation cycle happens faster than the API update, it is possible to
	// reschedule the creation of a Job. To avoid that, get if the Job is already submitted.
	if _, err := ctrl.CreateOrUpdate(ctx, r.GetClient(), nextJob, func() error { return nil }); err != nil {
		return lifecycle.Failed(ctx, r, &cluster, errors.Wrapf(err, "cannot create job"))
	}

	r.Logger.Info("Create clustered job",
		"cluster", cluster.GetName(),
		"service", nextJob.GetName(),
	)

	// exit and wait for watchers to trigger the next reconcile cycle
	return common.Stop()
}

/*
### Finalizers

*/

func (r *Controller) Finalizer() string {
	return "clusters.frisbee.io/finalizer"
}

func (r *Controller) Finalize(obj client.Object) error {
	r.Logger.Info("Finalize", "kind", reflect.TypeOf(obj), "name", obj.GetName())

	return nil
}

/*
### Setup
	Finally, we'll update our setup.  In order to allow our reconciler to quickly
	look up Services by their owner, we'll need an index.  We declare an index key that
	we can later use with the client as a pseudo-field name, and then describe how to
	extract the indexed value from the Service object.  The indexer will automatically take
	care of namespaces for us, so we just have to extract the owner name if the Service has
	a Cluster owner.

	Additionally, we'll inform the manager that this controller owns some Services, so that it
	will automatically call Reconcile on the underlying Cluster when a Service changes, is
	deleted, etc.
*/

func NewController(mgr ctrl.Manager, logger logr.Logger) error {

	// FieldIndexer knows how to index over a particular "field" such that it
	// can later be used by a field selector.
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1alpha1.Service{}, jobOwnerKey,
		func(rawObj client.Object) []string {
			// grab the job object, extract the owner...
			job := rawObj.(*v1alpha1.Service)

			owner := metav1.GetControllerOf(job)
			if owner == nil {
				return nil
			}

			// ...make sure it's managed by a Cluster controller...
			if owner.APIVersion != v1alpha1.GroupVersion.String() || owner.Kind != "Cluster" {
				return nil
			}

			// fixme: make sure it's managed by THIS cluster controller
			/*
				if owner.UID == thisControllerRef.UID {
					// The controller we found with this Name is not the same one that the
					// ControllerRef points to.
					return nil
				}
			*/

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
		Owns(&v1alpha1.Service{}, builder.WithPredicates(r.Watchers())).
		Complete(r)
}
