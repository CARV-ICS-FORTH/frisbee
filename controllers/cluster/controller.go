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

package cluster

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/carv-ics-forth/frisbee/controllers/common/scheduler"
	"github.com/carv-ics-forth/frisbee/controllers/common/watchers"
	"github.com/carv-ics-forth/frisbee/pkg/distributions"
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

// +kubebuilder:rbac:groups=frisbee.dev,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.dev,resources=clusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.dev,resources=clusters/finalizers,verbs=update

// Controller reconciles a Cluster object.
type Controller struct {
	ctrl.Manager
	logr.Logger

	view *lifecycle.Classifier
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current view of the cluster closer to the desired view.
func (r *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		1: Load CR by name and extract the Desired State
		------------------------------------------------------------------
	*/
	var cluster v1alpha1.Cluster

	var requeue bool
	result, err := common.Reconcile(ctx, r, req, &cluster, &requeue)

	if requeue {
		return result, err
	}

	r.Logger.Info("-> Reconcile",
		"obj", client.ObjectKeyFromObject(&cluster),
		"phase", cluster.Status.Phase,
		"version", cluster.GetResourceVersion(),
	)

	defer func() {
		r.Logger.Info("<- Reconciler",
			"obj", client.ObjectKeyFromObject(&cluster),
			"phase", cluster.Status.Phase,
			"version", cluster.GetResourceVersion(),
		)
	}()

	/*
		2: Load CR's children and classify their current state (view)
		------------------------------------------------------------------
	*/
	if err := r.PopulateView(ctx, req.NamespacedName); err != nil {
		return lifecycle.Failed(ctx, r, &cluster, errors.Wrapf(err, "cannot populate view for '%s'", req))
	}

	/*
		3: Use the view to update the CR's lifecycle.
		------------------------------------------------------------------
		The Update serves as "journaling" for the upcoming operations,
		and as a roadblock for stall (queued) requests.
	*/
	if r.updateLifecycle(&cluster) {
		if err := common.UpdateStatus(ctx, r, &cluster); err != nil {
			// due to the multiple updates, it is possible for this function to
			// be in conflict. We fix this issue by re-queueing the request.
			return common.RequeueAfter(r, req, time.Second)
		}
	}

	/*
		4: Make the world matching what we want in our spec.
		------------------------------------------------------------------
	*/

	if cluster.Spec.Suspend != nil && *cluster.Spec.Suspend {
		// If this object is suspended, we don't want to run any jobs, so we'll stop now.
		// This is useful if something's broken with the job we're running, and we want to
		// pause runs to investigate the cluster, without deleting the object.
		return common.Stop(r, req)
	}

	log := r.Logger.WithValues("object", client.ObjectKeyFromObject(&cluster))

	switch cluster.Status.Phase {
	case v1alpha1.PhaseUninitialized:
		if err := r.Initialize(ctx, &cluster); err != nil {
			return lifecycle.Failed(ctx, r, &cluster, errors.Wrapf(err, "initialization error"))
		}

		return lifecycle.Pending(ctx, r, &cluster, "ready to start creating jobs.")

	case v1alpha1.PhasePending:
		nextJob := cluster.Status.ScheduledJobs + 1

		//	If all jobs are scheduled but are not in the Running phase, they may be in the Pending phase.
		//	In both cases, we have nothing else to do but waiting for the next reconciliation cycle.
		if cluster.Spec.SuspendWhen == nil && (nextJob >= len(cluster.Status.QueuedJobs)) {
			return common.Stop(r, req)
		}

		// Get the next scheduled job
		hasJob, nextTick, err := scheduler.Schedule(log, &cluster, scheduler.Parameters{
			State:            *r.view,
			ScheduleSpec:     cluster.Spec.Schedule,
			LastScheduleTime: cluster.Status.LastScheduleTime,
			ExpectedTimeline: cluster.Status.ExpectedTimeline,
			JobName:          cluster.GetName(),
			ScheduledJobs:    cluster.Status.ScheduledJobs,
		})
		if err != nil {
			return lifecycle.Failed(ctx, r, &cluster, errors.Wrapf(err, "scheduling error"))
		}

		if !hasJob {
			r.Logger.Info("Nothing to do right now. Requeue request.", "nextTick", nextTick)

			if nextTick.IsZero() {
				return common.Stop(r, req)
			}

			return common.RequeueAfter(r, req, time.Until(nextTick))
		}

		// Build the job in kubernetes
		if err := r.runJob(ctx, &cluster, nextJob); err != nil {
			return lifecycle.Failed(ctx, r, &cluster, errors.Wrapf(err, "cannot create job"))
		}

		// Update the scheduling information
		cluster.Status.ScheduledJobs = nextJob
		cluster.Status.LastScheduleTime = metav1.Time{Time: time.Now()}

		return lifecycle.Pending(ctx, r, &cluster, fmt.Sprintf("Scheduled jobs: '%d/%d'",
			cluster.Status.ScheduledJobs+1, cluster.Spec.MaxInstances))

	case v1alpha1.PhaseRunning:
		// Nothing to do. Just wait for something to happen.
		return common.Stop(r, req)

	case v1alpha1.PhaseSuccess:
		if err := r.HasSucceed(ctx, &cluster); err != nil {
			return common.RequeueAfter(r, req, time.Second)
		}

		return common.Stop(r, req)

	case v1alpha1.PhaseFailed:
		if err := r.HasFailed(ctx, &cluster); err != nil {
			return common.RequeueAfter(r, req, time.Second)
		}

		return common.Stop(r, req)
	}

	panic(errors.New("This should never happen"))
}

func (r *Controller) Initialize(ctx context.Context, cluster *v1alpha1.Cluster) error {
	/*
		calculate any top-level distribution. this distribution will be respected during the construction of the jobs.
	*/
	if distName := cluster.Spec.DefaultDistributionSpec; distName != nil {
		cluster.Status.DefaultDistribution = distributions.GetPointDistribution(int64(cluster.Spec.MaxInstances), distName)
	}

	/*
		We construct a list of job specifications based on the CR's template.
		This list is used by the execution step to create the actual job.
		If the template is invalid, it should be captured at this stage.
	*/
	jobList, err := r.constructJobSpecList(ctx, cluster)
	if err != nil {
		return errors.Wrapf(err, "building joblist")
	}

	cluster.Status.QueuedJobs = jobList
	cluster.Status.ScheduledJobs = -1

	// Metrics-driven execution requires to set alerts on Grafana.
	if until := cluster.Spec.SuspendWhen; until != nil && until.HasMetricsExpr() {
		if err := expressions.SetAlert(ctx, cluster, until.Metrics); err != nil {
			return errors.Wrapf(err, "spec.suspendWhen")
		}
	}

	if schedule := cluster.Spec.Schedule; schedule != nil && schedule.Event.HasMetricsExpr() {
		if err := expressions.SetAlert(ctx, cluster, schedule.Event.Metrics); err != nil {
			return errors.Wrapf(err, "spec.schedule")
		}
	}

	if _, err := lifecycle.Pending(ctx, r, cluster, "submitting job requests"); err != nil {
		return errors.Wrapf(err, "status update")
	}

	return nil
}

func (r *Controller) PopulateView(ctx context.Context, req types.NamespacedName) error {
	r.view.Reset()

	var serviceJobs v1alpha1.ServiceList
	{
		if err := common.ListChildren(ctx, r, &serviceJobs, req); err != nil {
			return errors.Wrapf(err, "cannot list children for '%s'", req)
		}

		for i, job := range serviceJobs.Items {
			r.view.Classify(job.GetName(), &serviceJobs.Items[i])
		}
	}

	return nil
}

func (r *Controller) HasSucceed(ctx context.Context, cluster *v1alpha1.Cluster) error {
	r.Logger.Info("CleanOnSuccess",
		"obj", client.ObjectKeyFromObject(cluster).String(),
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

func (r *Controller) HasFailed(ctx context.Context, cluster *v1alpha1.Cluster) error {
	r.Logger.Info("!! JobError",
		"obj", client.ObjectKeyFromObject(cluster).String(),
		"reason ", cluster.Status.Reason,
		"message", cluster.Status.Message,
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
	cluster.Spec.Suspend = &suspend

	r.Logger.Info("Suspended",
		"obj", client.ObjectKeyFromObject(cluster),
		"reason", cluster.Status.Reason,
		"message", cluster.Status.Message,
	)

	if cluster.GetDeletionTimestamp().IsZero() {
		r.GetEventRecorderFor(cluster.GetName()).Event(cluster, corev1.EventTypeNormal,
			"Suspended", cluster.Status.Lifecycle.Message)
	}

	// Update is needed since we modify the spec.suspend
	return common.Update(ctx, r, cluster)
}

/*
### Finalizers

*/

func (r *Controller) Finalizer() string {
	return "clusters.frisbee.dev/finalizer"
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
	controller := &Controller{
		Manager: mgr,
		Logger:  logger.WithName("cluster"),
		view:    &lifecycle.Classifier{},
	}

	gvk := v1alpha1.GroupVersion.WithKind("Cluster")

	var (
		cluster v1alpha1.Cluster
		service v1alpha1.Service
	)

	return ctrl.NewControllerManagedBy(mgr).
		For(&cluster).
		Named("cluster").
		Owns(&service, watchers.Watch(controller, gvk)).
		Complete(controller)
}
