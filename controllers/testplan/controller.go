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

package testplan

import (
	"context"
	"reflect"
	"time"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	chaosutils "github.com/carv-ics-forth/frisbee/controllers/chaos/utils"
	serviceutils "github.com/carv-ics-forth/frisbee/controllers/service/utils"
	"github.com/carv-ics-forth/frisbee/controllers/telemetry/grafana"
	"github.com/carv-ics-forth/frisbee/controllers/utils"
	"github.com/carv-ics-forth/frisbee/controllers/utils/expressions"
	"github.com/carv-ics-forth/frisbee/controllers/utils/lifecycle"
	"github.com/carv-ics-forth/frisbee/controllers/utils/watchers"
	"github.com/go-logr/logr"
	notifier "github.com/golanghelper/grafana-webhook"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=frisbee.io,resources=testplans,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.io,resources=testplans/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.io,resources=testplans/finalizers,verbs=update

// +kubebuilder:rbac:groups=frisbee.io,resources=virtualobjects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.io,resources=virtualobjects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.io,resources=virtualobjects/finalizers,verbs=update

// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=configmaps/finalizers,verbs=update

type Controller struct {
	ctrl.Manager
	logr.Logger

	state lifecycle.Classifier

	serviceControl serviceutils.ServiceControlInterface
	chaosControl   chaosutils.ChaosControlInterface
}

func (r *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		1: Load CR by name.
		------------------------------------------------------------------
	*/
	var cr v1alpha1.TestPlan

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
		client.MatchingLabels{v1alpha1.LabelCreatedBy: req.Name},
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

	var cascadeJobs v1alpha1.CascadeList

	if err := r.GetClient().List(ctx, &cascadeJobs, filters...); err != nil {
		return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "unable to list child cascadeJobs"))
	}

	var virtualJobs v1alpha1.VirtualObjectList

	if err := r.GetClient().List(ctx, &virtualJobs, filters...); err != nil {
		return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "unable to list child virtual Jobs"))
	}

	var callJobs v1alpha1.CallList

	if err := r.GetClient().List(ctx, &callJobs, filters...); err != nil {
		return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "unable to list child call Jobs"))
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

	for i, job := range cascadeJobs.Items {
		r.state.Classify(job.GetName(), &cascadeJobs.Items[i])
	}

	for i, job := range virtualJobs.Items {
		r.state.Classify(job.GetName(), &virtualJobs.Items[i])
	}

	for i, job := range callJobs.Items {
		r.state.Classify(job.GetName(), &callJobs.Items[i])
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
	cr.SetReconcileStatus(r.updateLifecycle(&cr))

	if err := utils.UpdateStatus(ctx, r, &cr); err != nil {
		r.Info("update status error. retry", "object", cr.GetName(), "err", err)
		return utils.RequeueAfter(time.Second)
	}

	/*
		If this object is suspended, we don't want to run any jobs, so we'll call now.
		This is useful if something's broken with the job we're running, and we want to
		pause runs to investigate or putz with the cluster, without deleting the object.
	*/
	if cr.Spec.Suspend != nil && *cr.Spec.Suspend {
		r.Logger.Info("TestPlan is suspended",
			"testplan", cr.GetName(),
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
		// These components are removed by deleting the TestPlan.
		for _, job := range r.state.SuccessfulJobs() {
			expressions.UnsetAlert(job)

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

		for _, job := range r.state.PendingJobs() {
			utils.Delete(ctx, r, job)
		}

		for _, job := range r.state.RunningJobs() {
			utils.Delete(ctx, r, job)
		}

		// Suspend the workflow from creating new job.
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
	utils.AppendLabel(&cr, v1alpha1.LabelPartOfPlan, cr.GetName())

	/*
		initialize the CR.
	*/
	if cr.Status.Phase.Is(v1alpha1.PhaseUninitialized) {
		if err := r.Validate(ctx, &cr); err != nil {
			return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "invalid testplan"))
		}

		if err := UsePlatformConfiguration(ctx, r, &cr); err != nil {
			return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "cannot get platform configuration"))
		}

		telemetry, err := r.HasTelemetry(ctx, &cr)
		if err != nil {
			return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "cannot extract imports"))
		}

		if telemetry != nil {
			telemetryJob.SetName("telemetry")
			telemetryJob.Spec.ImportDashboards = telemetry

			// mark the job as system specific in order to exclude it from chaos events
			utils.AppendLabel(&cr, v1alpha1.LabelComponent, v1alpha1.ComponentSys)

			if err := utils.Create(ctx, r, &cr, &telemetryJob); err != nil {
				return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "cannot create the telemetry stack"))
			}

			cr.Status.WithTelemetry = true
		}

		meta.SetStatusCondition(&cr.Status.Conditions, metav1.Condition{
			Type:    v1alpha1.ConditionCRInitialized.String(),
			Status:  metav1.ConditionTrue,
			Reason:  "TestPlanInitialized",
			Message: "The Test Plan has been initialized. Start running actions",
		})

		if cr.Status.Executed == nil {
			cr.Status.Executed = make(map[string]v1alpha1.ConditionalExpr)
		}

		return lifecycle.Pending(ctx, r, &cr, "The TestPlan is ready to start submitting jobs.")
	}

	/*
		ensure that telemetry is running
	*/
	if cr.Status.WithTelemetry {
		// Call until the telemetry stack becomes ready.
		if telemetryJob.Status.Phase.Is(v1alpha1.PhaseUninitialized) || telemetryJob.Status.Phase.Is(v1alpha1.PhasePending) {
			return utils.Stop()
		}

		if telemetryJob.Status.Phase.Is(v1alpha1.PhaseSuccess) || telemetryJob.Status.Phase.Is(v1alpha1.PhaseFailed) {
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

	actionList, nextRun := GetNextLogicalJob(cr.GetCreationTimestamp(), cr.Spec.Actions, r.state, cr.Status.Executed)

	if len(actionList) == 0 {
		if nextRun.IsZero() {
			// nothing to do on this cycle. wait the next cycle trigger by watchers.
			return utils.Stop()
		}

		r.Logger.Info("no upcoming logical execution, sleeping until", "next", nextRun)

		return utils.RequeueAfter(time.Until(nextRun))
	}

	endpoints := r.supportedActions()
	for _, action := range actionList {
		if action.Assert.HasMetricsExpr() {
			// Assertions belong to the top-level workflow. Not to the job
			if err := expressions.SetAlert(&cr, action.Assert.Metrics); err != nil {
				return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "assertion error"))
			}
		}

		e, ok := endpoints[action.ActionType]
		if !ok {
			return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "unknown type [%s] for action [%s]",
				action.ActionType, action.Name))
		}

		job, err := e(ctx, &cr, action)
		if err != nil {
			return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "cannot run action [%s]", action.Name))
		}

		if err := utils.Create(ctx, r, &cr, job); err != nil {
			return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "execution error for action %s", action.Name))
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

		if action.Assert.IsZero() {
			cr.Status.Executed[action.Name] = v1alpha1.ConditionalExpr{}
		} else {
			cr.Status.Executed[action.Name] = *action.Assert
		}
	}

	return lifecycle.Pending(ctx, r, &cr, "some jobs are still pending")
}

func (r *Controller) ConnectToGrafana(ctx context.Context, cr *v1alpha1.TestPlan) error {
	conf := cr.Status.Configuration

	return grafana.NewGrafanaClient(ctx, r, conf.AdvertisedHost, conf.GrafanaEndpoint,
		// Set a callback that will be triggered when there is Grafana alert.
		// Through this channel we can get informed for SLA violations.
		grafana.WithNotifyOnAlert(func(b *notifier.Body) {
			r.Logger.Info("Grafana Alert", "body", b)

			// when Grafana fires an alert, this alert is captured by the Webhook.
			// The webhook must someone notify the appropriate controller.
			// To do that, it adds information of the fired alert to the object's metadata
			// and updates (patches) the object.
			if err := expressions.DispatchAlert(ctx, r, b); err != nil {
				r.Logger.Error(err, "unable to inform CR for metrics alert", "cr", cr.GetName())
			}
		}),
	)
}

/*
### Finalizers
*/

func (r *Controller) Finalizer() string {
	return "testplans.frisbee.io/finalizer"
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
		Logger:  logger.WithName("testplan"),
	}

	r.serviceControl = serviceutils.NewServiceControl(r)
	r.chaosControl = chaosutils.NewChaosControl(r)

	gvk := v1alpha1.GroupVersion.WithKind("TestPlan")

	return ctrl.NewControllerManagedBy(mgr).
		Named("testplan").
		For(&v1alpha1.TestPlan{}).
		Owns(&v1alpha1.Service{}, watchers.WatchService(r, gvk)).             // Watch Services
		Owns(&v1alpha1.Cluster{}, watchers.WatchCluster(r, gvk)).             // Watch Cluster
		Owns(&v1alpha1.Chaos{}, watchers.WatchChaos(r, gvk)).                 // Watch Chaos
		Owns(&v1alpha1.Telemetry{}, watchers.WatchTelemetry(r, gvk)).         // Watch Telemetry
		Owns(&v1alpha1.VirtualObject{}, watchers.WatchVirtualObject(r, gvk)). // Watch VirtualObjects
		Owns(&v1alpha1.Call{}, watchers.WatchCall(r, gvk)).                   // Watch Calls
		Complete(r)
}
