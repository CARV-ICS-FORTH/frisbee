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

package workflow

import (
	"context"
	"reflect"
	"time"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	serviceutils "github.com/carv-ics-forth/frisbee/controllers/service/utils"
	"github.com/carv-ics-forth/frisbee/controllers/telemetry/grafana"
	"github.com/carv-ics-forth/frisbee/controllers/utils"
	"github.com/carv-ics-forth/frisbee/controllers/utils/assertions"
	"github.com/carv-ics-forth/frisbee/controllers/utils/lifecycle"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=frisbee.io,resources=workflows,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.io,resources=workflows/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.io,resources=workflows/finalizers,verbs=update

// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=configmaps/finalizers,verbs=update

type Controller struct {
	ctrl.Manager
	logr.Logger

	gvk schema.GroupVersionKind

	state lifecycle.Classifier

	serviceControl serviceutils.ServiceControlInterface
}

func (r *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		1: Load CR by name.
		------------------------------------------------------------------
	*/
	var cr v1alpha1.Workflow

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
	filters := []client.ListOption{
		client.InNamespace(req.Namespace),
		client.MatchingLabels{v1alpha1.LabelManagedBy: req.Name},
		//	client.MatchingFields{jobOwnerKey: req.Name},
	}

	var telemetryJob v1alpha1.Telemetry
	{
		key := client.ObjectKey{
			Namespace: req.Namespace,
			Name:      "telemetry",
		}

		if err := r.GetClient().Get(ctx, key, &telemetryJob); client.IgnoreNotFound(err) != nil {
			return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "cannot get telemetryJob"))
		}
	}

	var serviceJobs v1alpha1.ServiceList

	if err := r.GetClient().List(ctx, &serviceJobs, filters...); err != nil {
		return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "unable to list child serviceJobs"))
	}

	var clusterJobs v1alpha1.ClusterList

	if err := r.GetClient().List(ctx, &clusterJobs, filters...); err != nil {
		return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "unable to list child clusterJobs"))
	}

	var chaosJobs v1alpha1.ChaosList

	if err := r.GetClient().List(ctx, &chaosJobs, filters...); err != nil {
		return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "unable to list child chaosJobs"))
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

	// Do not account telemetry jobs, unless they have failed.
	r.state.Exclude(telemetryJob.GetName(), telemetryJob.DeepCopy())

	for i, job := range serviceJobs.Items {
		r.state.Classify(job.GetName(), &serviceJobs.Items[i])
	}

	for i, job := range clusterJobs.Items {
		r.state.Classify(job.GetName(), &clusterJobs.Items[i])
	}

	for i, job := range chaosJobs.Items {
		r.state.Classify(job.GetName(), &chaosJobs.Items[i])
	}

	/*
		4: Update the CR status using the data we've gathered
		------------------------------------------------------------------

		The Update at this step serves two functions.
		First, it is like "journaling" for the upcoming operations.
		Second, it is a roadblock for stall (queued) requests.

		However, due to the multiple updates, it is possible for this function to
		be in conflict. We fix this issue by re-queueing the request.
		We also suppress verbose error reporting as to avoid polluting the output.
	*/
	r.updateLifecycle(&cr)

	if err := utils.UpdateStatus(ctx, r, &cr); err != nil {
		runtime.HandleError(err)

		return utils.RequeueAfter(time.Second)
	}

	/*
		If this object is suspended, we don't want to run any jobs, so we'll stop now.
		This is useful if something's broken with the job we're running, and we want to
		pause runs to investigate or putz with the cluster, without deleting the object.
	*/
	if cr.Spec.Suspend != nil && *cr.Spec.Suspend {
		r.Logger.Info("Workflow is suspended",
			"workflow", cr.GetName(),
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
	if cr.Status.Phase.Is(v1alpha1.PhaseSuccess) {
		// Remove the testing components once the experiment is successfully complete.
		// We maintain testbed components (e.g, prometheus and grafana) for getting back the test results.
		// These components are removed by deleting the Workflow.
		for _, job := range r.state.SuccessfulJobs() {
			assertions.UnsetAlert(job)

			utils.Delete(ctx, r, job)
		}

		return utils.Stop()
	}

	if cr.Status.Phase.Is(v1alpha1.PhaseFailed) {
		r.Logger.Error(errors.New(cr.Status.Reason), cr.Status.Message)

		// Remove the non-failed components. Leave the failed jobs and system jobs for postmortem analysis.
		for _, job := range r.state.SuccessfulJobs() {
			utils.Delete(ctx, r, job)
		}

		for _, job := range r.state.ActiveJobs() {
			utils.Delete(ctx, r, job)
		}

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

	// This label will be adopted by all children objects of this workflow.
	// It is not persisted in order to avoid additional updates.
	cr.SetLabels(labels.Merge(cr.GetLabels(), map[string]string{
		v1alpha1.BelongsToWorkflow: cr.GetName(),
	}))

	/*
		initialize the CR.
	*/
	if cr.Status.Phase.Is(v1alpha1.PhaseUninitialized) {
		if err := ValidateDAG(cr.Spec.Actions); err != nil {
			return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "invalid dependency DAG"))
		}

		if err := utils.UseDefaultPlatformConfiguration(ctx, r, cr.GetNamespace()); err != nil {
			return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "cannot get platform configuration"))
		}

		if cr.Spec.WithTelemetry != nil {
			telemetryJob.SetName("telemetry")

			cr.Spec.WithTelemetry.DeepCopyInto(&telemetryJob.Spec)

			if err := utils.Create(ctx, r, &cr, &telemetryJob); err != nil {
				return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "cannot create the telemetry stack"))
			}
		}

		meta.SetStatusCondition(&cr.Status.Conditions, metav1.Condition{
			Type:    v1alpha1.ConditionCRInitialized.String(),
			Status:  metav1.ConditionTrue,
			Reason:  "WorkflowInitialized",
			Message: "The workflow has been initialized. Start running actions",
		})

		return lifecycle.Pending(ctx, r, &cr, "The Workflow is ready to start submitting jobs.")
	}

	/*
		ensure that telemetry is running
	*/
	if cr.Spec.WithTelemetry != nil {
		// Stop until the telemetry stack becomes ready.
		if telemetryJob.Status.Phase.Is(v1alpha1.PhaseUninitialized) || telemetryJob.Status.Phase.Is(v1alpha1.PhasePending) {
			return utils.Stop()
		}

		if telemetryJob.Status.Phase.Is(v1alpha1.PhaseSuccess) {
			return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "the telemetry stack has terminated"))
		}

		// this should normally happen when the telemetry is running
		if grafana.DefaultClient == nil {
			if err := r.ConnectToGrafana(ctx, &cr); err != nil {
				return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "cannot communicate with the telemetry stack"))
			}
		}
	}

	/*
		7: Get the next logical run
		------------------------------------------------------------------
	*/
	if cr.Status.Phase.Is(v1alpha1.PhaseRunning) {
		return utils.Stop()
	}

	actionList, nextRun := GetNextLogicalJob(&cr, cr.Spec.Actions, r.state, cr.Status.Executed)

	if len(actionList) == 0 {
		if nextRun.IsZero() {
			// nothing to do on this cycle. wait the next cycle trigger by watchers.
			return utils.Stop()
		}

		r.Logger.Info("no upcoming logical execution, sleeping until", "next", nextRun)

		return utils.RequeueAfter(time.Until(nextRun))
	}

	for _, action := range actionList {
		job, err := r.getJob(ctx, &cr, action)
		if err != nil {
			return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "erroneous action [%s]", action.Name))
		}

		if action.Assert != nil && action.Assert.SLA != "" {
			if err := assertions.SetAlert(job, action.Assert.SLA, action.Name); err != nil {
				return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "assertion error"))
			}
		}

		if err := utils.Create(ctx, r, &cr, job); err != nil {
			return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "action %s execution failed", action.Name))
		}
	}

	/*
		8: Avoid double actions
		------------------------------------------------------------------

		If this process restarts at this point (after posting a job, but
		before updating the status), then we might try to start the job on
		the next time.  Actually, if we re-list the Jobs on the next cycle
		we might not see our own status update, and then post one again.
		So, we need to use the job name as a lock to prevent us from making the job twice.
	*/
	if cr.Status.Executed == nil {
		cr.Status.Executed = make(map[string]metav1.Time)
	}

	for _, action := range actionList {
		cr.Status.Executed[action.Name] = metav1.Now()
	}

	return lifecycle.Pending(ctx, r, &cr, "some jobs are still pending")
}

/*
### Finalizers
*/

func (r *Controller) Finalizer() string {
	return "workflows.frisbee.io/finalizer"
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
	// instantiate the controller
	r := &Controller{
		Manager: mgr,
		Logger:  logger.WithName("workflow"),
		gvk:     v1alpha1.GroupVersion.WithKind("Workflow"),
	}

	r.serviceControl = serviceutils.NewServiceControl(r)

	return ctrl.NewControllerManagedBy(mgr).
		Named("workflow").
		For(&v1alpha1.Workflow{}).
		Owns(&v1alpha1.Service{}, builder.WithPredicates(r.WatchServices())). // Watch Services
		Owns(&v1alpha1.Cluster{}, builder.WithPredicates(r.WatchClusters())). // Watch Cluster
		Owns(&v1alpha1.Chaos{}, builder.WithPredicates(r.WatchChaos())). // Watch Chaos
		Owns(&v1alpha1.Telemetry{}, builder.WithPredicates(r.WatchTelemetry())). // Watch Telemetry
		Complete(r)
}
