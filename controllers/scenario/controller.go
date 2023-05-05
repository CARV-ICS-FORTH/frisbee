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

package scenario

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/carv-ics-forth/frisbee/controllers/common/watchers"
	scenarioutils "github.com/carv-ics-forth/frisbee/controllers/scenario/utils"
	"github.com/carv-ics-forth/frisbee/pkg/configuration"
	"github.com/carv-ics-forth/frisbee/pkg/expressions"
	"github.com/carv-ics-forth/frisbee/pkg/lifecycle"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=frisbee.dev,resources=scenarios,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.dev,resources=scenarios/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.dev,resources=scenarios/finalizers,verbs=update

// +kubebuilder:rbac:groups=frisbee.dev,resources=virtualobjects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.dev,resources=virtualobjects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.dev,resources=virtualobjects/finalizers,verbs=update

// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=configmaps/finalizers,verbs=update

// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=nodes/status,verbs=get

type Controller struct {
	ctrl.Manager
	logr.Logger

	view *lifecycle.Classifier

	notificationEndpoint string
}

func (r *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		1: Load CR by name and extract the Desired State
		------------------------------------------------------------------
	*/
	var scenario v1alpha1.Scenario

	var requeue bool
	result, err := common.Reconcile(ctx, r, req, &scenario, &requeue)

	if requeue {
		return result, err
	}

	r.Logger.Info("-> Reconcile",
		"obj", client.ObjectKeyFromObject(&scenario),
		"phase", scenario.Status.Phase,
		"version", scenario.GetResourceVersion(),
	)

	defer func() {
		r.Logger.Info("<- Reconciler",
			"obj", client.ObjectKeyFromObject(&scenario),
			"phase", scenario.Status.Phase,
			"version", scenario.GetResourceVersion(),
		)
	}()

	/*
		2: Load CR's children and classify their current state (view)
		------------------------------------------------------------------
	*/
	if err := r.PopulateView(ctx, req.NamespacedName); err != nil {
		return lifecycle.Failed(ctx, r, &scenario, errors.Wrapf(err, "cannot populate view for '%s'", req))
	}

	/* Check if all the SYS services are running. If they are terminated (Failed/Success), we have nothing else to do,
	and we abort the experiment. If they are still being created (Uninitialized, Pending), we sleep and retry */
	if abort, sysErr := r.view.SystemState(); sysErr != nil {
		if abort {
			return lifecycle.Failed(ctx, r, &scenario, errors.Wrapf(sysErr, "errorneous system state"))
		}

		// Just stop, waiting for system services to come up.
		return common.Stop(r, req)
	}

	/*
		3: Use the view to update the CR's lifecycle.
		------------------------------------------------------------------
		The Update serves as "journaling" for the upcoming operations,
		and as a roadblock for stall (queued) requests.
	*/
	if r.updateLifecycle(&scenario) {
		if err := common.UpdateStatus(ctx, r, &scenario); err != nil {
			// due to the multiple updates, it is possible for this function to
			// be in conflict. We fix this issue by re-queueing the request.
			return common.RequeueAfter(r, req, time.Second)
		}
	}

	/*
		4: Make the world matching what we want in our spec.
		------------------------------------------------------------------
	*/

	// If this object is suspended, we don't want to run any jobs, so we'll stop now.
	// This is useful if something's broken with the job we're running, and we want to
	// pause runs to investigate the cluster, without deleting the object.
	if scenario.Spec.Suspend != nil && *scenario.Spec.Suspend {
		return common.Stop(r, req)
	}

	// Label this resource with the name of the scenario.
	// This label will be adopted by all children objects of this workflow.
	v1alpha1.SetScenarioLabel(&scenario.ObjectMeta, scenario.GetName())

	switch scenario.Status.Phase {
	case v1alpha1.PhaseUninitialized:
		if err := r.Initialize(ctx, &scenario); err != nil {
			return lifecycle.Failed(ctx, r, &scenario, errors.Wrapf(err, "initialization error"))
		}

		// We could use common.Stop() to simply wait, but we need update status because Initialize()
		// sets the endpoints, and we want to maintain this information for connectToGrafana().
		return lifecycle.Pending(ctx, r, &scenario, "Initializing the testing environment")

	case v1alpha1.PhasePending:
		actionList, nextRun, err := r.NextJobs(&scenario)
		if err != nil {
			return lifecycle.Failed(ctx, r, &scenario, errors.Wrapf(err, "scheduling error"))
		}

		if len(actionList) == 0 {
			if nextRun.IsZero() {
				// nothing to do on this cycle. wait the next cycle trigger by watchers.
				return common.Stop(r, req)
			}

			r.Logger.Info(".. RequeueEvent",
				"obj", client.ObjectKeyFromObject(&scenario),
				"sleep until", nextRun)

			return common.RequeueAfter(r, req, time.Until(nextRun))
		}

		if err := r.RunActions(ctx, &scenario, actionList); err != nil {
			return lifecycle.Failed(ctx, r, &scenario, errors.Wrapf(err, "actions failed"))
		}

		return lifecycle.Pending(ctx, r, &scenario, fmt.Sprintf("Scheduled jobs: '%d/%d'",
			len(scenario.Status.ScheduledJobs), len(scenario.Spec.Actions)))

	case v1alpha1.PhaseRunning:
		// Nothing to do. Just wait for something to happen.
		return common.Stop(r, req)

	case v1alpha1.PhaseSuccess:
		if err := r.HasSucceed(ctx, &scenario); err != nil {
			return common.RequeueAfter(r, req, time.Second)
		}

		return common.Stop(r, req)

	case v1alpha1.PhaseFailed:
		if err := r.HasFailed(ctx, &scenario); err != nil {
			return common.RequeueAfter(r, req, time.Second)
		}

		return common.Stop(r, req)
	}

	panic(errors.New("This should never happen"))
}

func (r *Controller) Initialize(ctx context.Context, scenario *v1alpha1.Scenario) error {
	/* Clone system configuration, needed to retrieve telemetry, chaos, etc  */
	sysconf, err := configuration.Get(ctx, r.GetClient(), r.Logger)
	if err != nil {
		return errors.Wrapf(err, "cannot get system configuration")
	}

	/* FIXME: we set the configuration be global here. is there any better way ? */
	configuration.SetGlobal(sysconf)

	/*  Not the best place, but the webhook should start after we get the configuration parameters.
	Given that, we need to start it here, and only once. An alternative solution would be to get
	the webhook port and developer mode as parameters on the executable.
	*/
	startWebhookOnce.Do(func() {
		if err := r.CreateWebhookServer(ctx); err != nil {
			panic(errors.Wrapf(err, "cannot create grafana webhook"))
		}
	})

	// load the templates required by the scenario.
	if errValidate := scenarioutils.LoadTemplates(ctx, r.GetClient(), scenario); errValidate != nil {
		return errors.Wrapf(errValidate, "template error")
	}

	// Start Prometheus + Grafana
	if errTelemetry := r.StartTelemetry(ctx, scenario); errTelemetry != nil {
		return errors.Wrapf(errTelemetry, "telemetry error")
	}

	r.GetEventRecorderFor(scenario.GetName()).Event(scenario, corev1.EventTypeNormal, "Initialized", "Start scheduling jobs")

	meta.SetStatusCondition(&scenario.Status.Conditions, metav1.Condition{
		Type:    v1alpha1.ConditionCRInitialized.String(),
		Status:  metav1.ConditionTrue,
		Reason:  "Initialized2",
		Message: "Start Scheduling Jobs",
	})

	return nil
}

/*
PopulateView list all child objects in this namespace that belong to this scenario, and split them into
active, successful, and failed jobs.
*/
func (r *Controller) PopulateView(ctx context.Context, req types.NamespacedName) error {
	r.view.Reset()

	var serviceJobs v1alpha1.ServiceList
	{
		if err := common.ListChildren(ctx, r.GetClient(), &serviceJobs, req); err != nil {
			return errors.Wrapf(err, "cannot list child services for '%s'", req)
		}

		for i, job := range serviceJobs.Items {
			r.view.Classify(job.GetName(), &serviceJobs.Items[i])
		}
	}

	var clusterJobs v1alpha1.ClusterList
	{
		if err := common.ListChildren(ctx, r.GetClient(), &clusterJobs, req); err != nil {
			return errors.Wrapf(err, "cannot list child clusters for '%s'", req)
		}

		for i, job := range clusterJobs.Items {
			r.view.Classify(job.GetName(), &clusterJobs.Items[i])
		}
	}

	var chaosJobs v1alpha1.ChaosList
	{
		if err := common.ListChildren(ctx, r.GetClient(), &chaosJobs, req); err != nil {
			return errors.Wrapf(err, "cannot list child chaos for '%s'", req)
		}

		for i, job := range chaosJobs.Items {
			r.view.Classify(job.GetName(), &chaosJobs.Items[i])
		}
	}

	var cascadeJobs v1alpha1.CascadeList
	{
		if err := common.ListChildren(ctx, r.GetClient(), &cascadeJobs, req); err != nil {
			return errors.Wrapf(err, "cannot list child cascades for '%s'", req)
		}

		for i, job := range cascadeJobs.Items {
			r.view.Classify(job.GetName(), &cascadeJobs.Items[i])
		}
	}

	var virtualJobs v1alpha1.VirtualObjectList
	{
		if err := common.ListChildren(ctx, r.GetClient(), &virtualJobs, req); err != nil {
			return errors.Wrapf(err, "cannot list child virtualobjects for '%s'", req)
		}

		for i, job := range virtualJobs.Items {
			r.view.Classify(job.GetName(), &virtualJobs.Items[i])
		}
	}

	var callJobs v1alpha1.CallList
	{
		if err := common.ListChildren(ctx, r.GetClient(), &callJobs, req); err != nil {
			return errors.Wrapf(err, "cannot list child calls for '%s'", req)
		}

		for i, job := range callJobs.Items {
			r.view.Classify(job.GetName(), &callJobs.Items[i])
		}
	}

	return nil
}

func (r *Controller) HasSucceed(ctx context.Context, scenario *v1alpha1.Scenario) error {
	r.GetEventRecorderFor(scenario.GetName()).Event(scenario, corev1.EventTypeNormal,
		scenario.Status.Lifecycle.Reason, scenario.Status.Lifecycle.Message)

	r.Logger.Info("CleanOnSuccess",
		"obj", client.ObjectKeyFromObject(scenario).String(),
		"successfulJobs", r.view.ListSuccessfulJobs(),
	)

	// Remove the testing components once the experiment is successfully complete.
	// We maintain testbed components (e.g, prometheus and grafana) for getting back the test results.
	// These components are removed by deleting the Scenario.
	for _, job := range r.view.GetSuccessfulJobs() {
		expressions.UnsetAlert(ctx, job)
		// common.Delete(ctx, r, job)
	}

	if scenario.GetDeletionTimestamp().IsZero() {
		r.GetEventRecorderFor(scenario.GetName()).Event(scenario, corev1.EventTypeNormal, "Completed", scenario.Status.Lifecycle.Message)
	}

	return nil
}

func (r *Controller) HasFailed(ctx context.Context, scenario *v1alpha1.Scenario) error {
	r.Logger.Info("!! JobError",
		"obj", client.ObjectKeyFromObject(scenario).String(),
		"reason ", scenario.Status.Reason,
		"message", scenario.Status.Message,
	)

	// TODO: What should we do when a call action fails ? Should we delete all services ?

	// Remove the non-failed components. Leave the failed jobs and system jobs for postmortem analysis.
	for _, job := range r.view.GetPendingJobs() {
		expressions.UnsetAlert(ctx, job)
		common.Delete(ctx, r, job)
	}

	for _, job := range r.view.GetRunningJobs() {
		expressions.UnsetAlert(ctx, job)
		common.Delete(ctx, r, job)
	}

	for _, job := range r.view.GetSuccessfulJobs() {
		expressions.UnsetAlert(ctx, job)
		// common.Delete(ctx, r, job) Keep it commented. It is useful to see which jobs are complete.
	}

	r.StopTelemetry(scenario)

	// Suspend the workflow from creating new job.
	suspend := true
	scenario.Spec.Suspend = &suspend

	r.Logger.Info("Suspended",
		"obj", client.ObjectKeyFromObject(scenario),
		"reason", scenario.Status.Reason,
		"message", scenario.Status.Message,
	)

	if scenario.GetDeletionTimestamp().IsZero() {
		r.GetEventRecorderFor(scenario.GetName()).Event(scenario, corev1.EventTypeWarning,
			"Suspended", scenario.Status.Lifecycle.Message)
	}

	// Update is needed since we modify the spec.suspend
	return common.Update(ctx, r, scenario)
}

func (r *Controller) RunActions(ctx context.Context, scenario *v1alpha1.Scenario, actionList []v1alpha1.Action) error {
	if scenario.Status.GrafanaEndpoint == "" {
		r.Logger.Info("The Grafana endpoint is empty. Skip telemetry.", "scenario", scenario.GetName())
	} else {
		if err := r.connectToGrafana(ctx, scenario, r.notificationEndpoint); err != nil {
			return errors.Wrapf(err, "connect to grafana")
		}
	}

	for _, action := range actionList {
		if action.Assert.HasMetricsExpr() {
			// Assert belong to the top-level workflow. Not to the job
			if err := expressions.SetAlert(ctx, scenario, action.Assert.Metrics); err != nil {
				return errors.Wrapf(err, "cannot set assertions for action '%s'", action.Name)
			}
		}

		if err := r.RunAction(ctx, scenario, action); err != nil {
			return errors.Wrapf(err, "cannot run action '%s'", action.Name)
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
		scenario.Status.ScheduledJobs = append(scenario.Status.ScheduledJobs, action.Name)
	}

	return nil
}

/*
### Finalizers
*/

func (r *Controller) Finalizer() string {
	return "scenarios.frisbee.dev/finalizer"
}

func (r *Controller) Finalize(obj client.Object) error {
	r.Logger.Info("XX Finalize",
		"kind", reflect.TypeOf(obj),
		"name", obj.GetName(),
		"version", obj.GetResourceVersion(),
	)

	r.StopTelemetry(obj.(*v1alpha1.Scenario))

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
	controller := &Controller{
		Manager: mgr,
		Logger:  logger.WithName("scenario"),
		view:    &lifecycle.Classifier{},
	}

	gvk := v1alpha1.GroupVersion.WithKind("Scenario")

	// known types
	var (
		scenario v1alpha1.Scenario
		service  v1alpha1.Service
		cluster  v1alpha1.Cluster
		chaos    v1alpha1.Chaos
		cascade  v1alpha1.Cascade
		vobject  v1alpha1.VirtualObject
		call     v1alpha1.Call
	)

	// register types to the controller
	return ctrl.NewControllerManagedBy(mgr).
		Named("scenario").
		For(&scenario).
		Owns(&service, watchers.Watch(controller, gvk)). // Logs Services
		Owns(&cluster, watchers.Watch(controller, gvk)). // Logs Cluster
		Owns(&chaos, watchers.Watch(controller, gvk)).   // Logs Chaos
		Owns(&cascade, watchers.Watch(controller, gvk)). // Logs Cascade
		Owns(&vobject, watchers.Watch(controller, gvk)). // Logs VirtualObjects
		Owns(&call, watchers.Watch(controller, gvk)).    // Logs Calls
		Complete(controller)
}
