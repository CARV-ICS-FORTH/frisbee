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
	"github.com/carv-ics-forth/frisbee/controllers/utils"
	"github.com/carv-ics-forth/frisbee/controllers/utils/configuration"
	"github.com/carv-ics-forth/frisbee/controllers/utils/expressions"
	"github.com/carv-ics-forth/frisbee/controllers/utils/lifecycle"
	"github.com/carv-ics-forth/frisbee/controllers/utils/watchers"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
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

	clusterView lifecycle.Classifier

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

	if err := r.GetClusterView(ctx, &cr); err != nil {
		return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "cannot get the cluster view"))
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
	cr.SetReconcileStatus(r.updateLifecycle(&cr, r.clusterView))

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
		for _, job := range r.clusterView.SuccessfulJobs() {
			expressions.UnsetAlert(job)

			utils.Delete(ctx, r, job)
		}

		return utils.Stop()
	}

	if cr.Status.Phase.Is(v1alpha1.PhaseFailed) {
		if err := r.HasFailed(ctx, &cr, r.clusterView); err != nil {
			return utils.RequeueAfter(time.Second)
		}

		return utils.Stop()
	}

	if cr.Status.Phase.Is(v1alpha1.PhaseRunning) {
		return utils.Stop()
	}

	if cr.Status.Phase.Is(v1alpha1.PhaseUninitialized) {
		if err := r.Initialize(ctx, &cr); err != nil {
			return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "cannot initialize"))
		}

		if err := r.ConnectToGrafana(ctx, &cr); err != nil {
			return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "cannot communicate with the telemetry stack"))
		}

		return lifecycle.Pending(ctx, r, &cr, "The TestPlan is ready to start submitting jobs.")
	}

	/*
		6: Make the world matching what we want in our spec
		------------------------------------------------------------------

		Once we've updated our status, we can move on to ensuring that the status of
		the world matches what we want in our spec.

		We may delete the service, add a pod, or wait for existing pod to change its status.
	*/

	/*
		7: Get the next logical run
		------------------------------------------------------------------
	*/
	actionList, nextRun := GetNextLogicalJob(cr.GetCreationTimestamp(), cr.Spec.Actions, r.clusterView, cr.Status.ExecutedActions)

	if len(actionList) == 0 {
		if nextRun.IsZero() {
			// nothing to do on this cycle. wait the next cycle trigger by watchers.
			return utils.Stop()
		}

		r.Logger.Info("no upcoming logical execution, sleeping until", "next", nextRun)

		return utils.RequeueAfter(time.Until(nextRun))
	}

	if err := r.RunActions(ctx, &cr, actionList); err != nil {
		return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "cannot run actions"))
	}

	return lifecycle.Pending(ctx, r, &cr, "some jobs are still pending")
}

func (r *Controller) Initialize(ctx context.Context, t *v1alpha1.TestPlan) error {
	/* Clone system configuration, needed to retrieve telemetry, chaos, etc  */
	sysconf, err := configuration.Get(ctx, r.GetClient(), r.Logger)
	if err != nil {
		return errors.Wrapf(err, "cannot get system configuration")
	}

	/* FIXME: we set the configuration be global here. is there any better way ? */
	configuration.SetGlobal(sysconf)

	/* Ensure that the plan is OK */
	if err := r.Validate(ctx, t, r.clusterView); err != nil {
		return errors.Wrapf(err, "invalid testplan")
	}

	{ // Initialize metadata
		/* Inherit the metadata of the configuration. This is used to automatically delete and remove the
		resources if the configuration is deleted */
		// utils.AppendLabels(t, configMeta.GetLabels())
		// utils.AppendLabels(t, configMeta.GetAnnotations())

		/* Inherit the metadata of the test plan. This label will be adopted by all children objects of this workflow.
		 */
		utils.AppendLabel(t, v1alpha1.LabelPartOfPlan, t.GetName())

		if err := utils.Update(ctx, r, t); err != nil {
			return errors.Wrap(err, "cannot update metadata")
		}
	}

	if err := r.StartTelemetry(ctx, t); err != nil {
		return errors.Wrapf(err, "cannot create the telemetry stack")
	}

	meta.SetStatusCondition(&t.Status.Conditions, metav1.Condition{
		Type:    v1alpha1.ConditionCRInitialized.String(),
		Status:  metav1.ConditionTrue,
		Reason:  "TestPlanInitialized",
		Message: "The Test Plan has been initialized. Start running actions",
	})

	return nil
}

/*
	GetClusterView list all child objects in this namespace that belong to this plan, and split them into
	active, successful, and failed jobs.
*/
func (r *Controller) GetClusterView(ctx context.Context, t *v1alpha1.TestPlan) error {
	/*
		to relief garbage collector, we use a common state that is reset at every reconciliation cycle.
		fixme: this approach may fail if we run multiple  reconciliation loops simultaneously.
	*/
	r.clusterView.Reset()

	/*
		As our number of services increases, looking these up can become quite slow as we have to filter through all
		of them. For a more efficient lookup, these services will be indexed locally on the controller's name.
		A jobOwnerKey field is added to the cached job objects, which references the owning controller.
		Check how we configure the manager to actually index this field.
	*/
	filters := []client.ListOption{
		client.InNamespace(t.GetNamespace()),
		client.MatchingLabels{v1alpha1.LabelCreatedBy: t.GetName()},
		//	client.MatchingFields{jobOwnerKey: req.Name},
	}

	var serviceJobs v1alpha1.ServiceList
	{
		if err := r.GetClient().List(ctx, &serviceJobs, filters...); err != nil {
			return errors.Wrapf(err, "unable to list child serviceJobs")
		}

		for i, job := range serviceJobs.Items {
			// Do not account telemetry jobs, unless they have failed.
			if job.GetName() == notRandomGrafanaName || job.GetName() == notRandomPrometheusName {
				r.clusterView.Exclude(job.GetName(), &serviceJobs.Items[i])
			} else {
				r.clusterView.Classify(job.GetName(), &serviceJobs.Items[i])
			}
		}
	}

	var clusterJobs v1alpha1.ClusterList
	{
		if err := r.GetClient().List(ctx, &clusterJobs, filters...); err != nil {
			return errors.Wrapf(err, "unable to list child clusterJobs")
		}

		for i, job := range clusterJobs.Items {
			r.clusterView.Classify(job.GetName(), &clusterJobs.Items[i])
		}
	}

	var chaosJobs v1alpha1.ChaosList
	{
		if err := r.GetClient().List(ctx, &chaosJobs, filters...); err != nil {
			return errors.Wrapf(err, "unable to list child chaosJobs")
		}

		for i, job := range chaosJobs.Items {
			r.clusterView.Classify(job.GetName(), &chaosJobs.Items[i])
		}
	}

	var cascadeJobs v1alpha1.CascadeList
	{
		if err := r.GetClient().List(ctx, &cascadeJobs, filters...); err != nil {
			return errors.Wrapf(err, "unable to list child cascadeJobs")
		}

		for i, job := range cascadeJobs.Items {
			r.clusterView.Classify(job.GetName(), &cascadeJobs.Items[i])
		}
	}

	var virtualJobs v1alpha1.VirtualObjectList
	{
		if err := r.GetClient().List(ctx, &virtualJobs, filters...); err != nil {
			return errors.Wrapf(err, "unable to list child virtual Jobs")
		}

		for i, job := range virtualJobs.Items {
			r.clusterView.Classify(job.GetName(), &virtualJobs.Items[i])
		}
	}

	var callJobs v1alpha1.CallList
	{
		if err := r.GetClient().List(ctx, &callJobs, filters...); err != nil {
			return errors.Wrapf(err, "unable to list child call Jobs")
		}

		for i, job := range callJobs.Items {
			r.clusterView.Classify(job.GetName(), &callJobs.Items[i])
		}
	}

	return nil
}

func (r *Controller) HasFailed(ctx context.Context, t *v1alpha1.TestPlan, clusterView lifecycle.ClassifierReader) error {
	r.Logger.Error(errors.New(t.Status.Reason), t.Status.Message)

	// Remove the non-failed components. Leave the failed jobs and system jobs for postmortem analysis.
	for _, job := range clusterView.PendingJobs() {
		if utils.IsSystemService(job) {
			continue // System jobs should not be deleted
		}

		r.GetEventRecorderFor("").Event(job, corev1.EventTypeWarning, "Terminating", t.Status.Message)

		utils.Delete(ctx, r, job)
	}

	for _, job := range clusterView.RunningJobs() {
		if utils.IsSystemService(job) {
			continue // System jobs should not be deleted
		}

		r.GetEventRecorderFor("").Event(job, corev1.EventTypeWarning, "Terminating", t.Status.Message)

		utils.Delete(ctx, r, job)
	}

	for _, job := range clusterView.SuccessfulJobs() {
		if utils.IsSystemService(job) {
			continue // System jobs should not be deleted
		}

		r.GetEventRecorderFor("").Event(job, corev1.EventTypeWarning, "Terminating", t.Status.Message)

		utils.Delete(ctx, r, job)
	}

	// Suspend the workflow from creating new job.
	suspend := true
	t.Spec.Suspend = &suspend

	if err := utils.Update(ctx, r, t); err != nil {
		return errors.Wrapf(err, "unable to suspend execution for '%s'", t.GetName())
	}

	return nil
}

func (r *Controller) RunActions(ctx context.Context, t *v1alpha1.TestPlan, actionList []v1alpha1.Action) error {
	endpoints := r.supportedActions()
	for _, action := range actionList {
		if action.Assert.HasMetricsExpr() {
			// Assertions belong to the top-level workflow. Not to the job
			if err := expressions.SetAlert(t, action.Assert.Metrics); err != nil {
				return errors.Wrapf(err, "assertion error")
			}
		}

		e, ok := endpoints[action.ActionType]
		if !ok {
			return errors.Errorf("unknown type [%s] for action [%s]", action.ActionType, action.Name)
		}

		job, err := e(ctx, t, action)
		if err != nil {
			return errors.Wrapf(err, "cannot run action [%s]", action.Name)
		}

		if err := utils.Create(ctx, r, t, job); err != nil {
			return errors.Wrapf(err, "execution error for action %s", action.Name)
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
		if t.Status.ExecutedActions == nil {
			t.Status.ExecutedActions = make(map[string]v1alpha1.ConditionalExpr)
		}

		if action.Assert.IsZero() {
			t.Status.ExecutedActions[action.Name] = v1alpha1.ConditionalExpr{}
		} else {
			t.Status.ExecutedActions[action.Name] = *action.Assert
		}
	}

	return nil
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

	if err := r.CreateWebhookServer(context.Background()); err != nil {
		return errors.Wrapf(err, "cannot create grafana webhook")
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named("testplan").
		For(&v1alpha1.TestPlan{}).
		Owns(&v1alpha1.Service{}, watchers.WatchService(r, gvk)).             // Watch Services
		Owns(&v1alpha1.Cluster{}, watchers.WatchCluster(r, gvk)).             // Watch Cluster
		Owns(&v1alpha1.Chaos{}, watchers.WatchChaos(r, gvk)).                 // Watch Chaos
		Owns(&v1alpha1.VirtualObject{}, watchers.WatchVirtualObject(r, gvk)). // Watch VirtualObjects
		Owns(&v1alpha1.Call{}, watchers.WatchCall(r, gvk)).                   // Watch Calls
		Complete(r)
}
