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

package scenario

import (
	"context"
	"reflect"
	"time"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/carv-ics-forth/frisbee/controllers/common/configuration"
	"github.com/carv-ics-forth/frisbee/controllers/common/expressions"
	"github.com/carv-ics-forth/frisbee/controllers/common/lifecycle"
	"github.com/carv-ics-forth/frisbee/controllers/common/watchers"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
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

type Controller struct {
	ctrl.Manager
	logr.Logger

	view lifecycle.Classifier

	alertingPort int
}

func (r *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		1: Load CR by name and extract the Desired State
		------------------------------------------------------------------
	*/
	var cr v1alpha1.Scenario

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
		2: Load CR's children and classify their current state (view)
		------------------------------------------------------------------
	*/
	if err := r.PopulateView(ctx, req.NamespacedName); err != nil {
		return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "cannot populate view for '%s'", req))
	}

	/*
		3: Use the view to update the CR's lifecycle.
		------------------------------------------------------------------
		The Update serves as "journaling" for the upcoming operations,
		and as a roadblock for stall (queued) requests.
	*/
	r.updateLifecycle(&cr)

	if err := common.UpdateStatus(ctx, r, &cr); err != nil {
		// due to the multiple updates, it is possible for this function to
		// be in conflict. We fix this issue by re-queueing the request.
		return common.RequeueAfter(time.Second)
	}

	/*
		4: Make the world matching what we want in our spec.
		------------------------------------------------------------------
	*/

	if cr.Spec.Suspend != nil && *cr.Spec.Suspend {
		// If this object is suspended, we don't want to run any jobs, so we'll stop now.
		// This is useful if something's broken with the job we're running, and we want to
		// pause runs to investigate the cluster, without deleting the object.

		r.GetEventRecorderFor(cr.GetName()).Event(&cr, corev1.EventTypeNormal,
			"Suspend", cr.Status.Lifecycle.Message)

		r.Logger.Info("Suspended",
			"scenario", cr.GetName(),
			"reason", cr.Status.Reason,
			"message", cr.Status.Message,
		)

		return common.Stop()
	}

	// Label this resource with the name of the scenario.
	// This label will be adopted by all children objects of this workflow.
	v1alpha1.SetScenarioLabel(&cr.ObjectMeta, cr.GetName())

	switch cr.Status.Phase {

	case v1alpha1.PhaseSuccess:
		if err := r.HasSucceed(ctx, &cr); err != nil {
			return common.RequeueAfter(time.Second)
		}

		return common.Stop()

	case v1alpha1.PhaseFailed:
		if err := r.HasFailed(ctx, &cr); err != nil {
			return common.RequeueAfter(time.Second)
		}

		return common.Stop()

	case v1alpha1.PhaseRunning:
		// Nothing to do. Just wait for something to happen.
		r.Logger.Info(".. Awaiting",
			"name", cr.GetName(),
			cr.Status.Reason, cr.Status.Message,
		)

		return common.Stop()

	case v1alpha1.PhaseUninitialized:
		if err := r.Initialize(ctx, &cr); err != nil {
			return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "scenario initialization error"))
		}

		return lifecycle.Pending(ctx, r, &cr, "The Scenario is ready to start submitting jobs.")

	case v1alpha1.PhasePending:
		actionList, nextRun := r.NextJobs(&cr)

		if len(actionList) == 0 {
			if nextRun.IsZero() {
				// nothing to do on this cycle. wait the next cycle trigger by watchers.
				return common.Stop()
			}

			r.Logger.Info(".. Awaiting",
				"name", cr.GetName(),
				"sleep until", nextRun)

			return common.RequeueAfter(time.Until(nextRun))
		}

		if err := r.RunActions(ctx, &cr, actionList); err != nil {
			return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "actions failed"))
		}

		return lifecycle.Pending(ctx, r, &cr, "some jobs are still pending")

	}

	panic(errors.New("This should never happen"))
}

func (r *Controller) Initialize(ctx context.Context, cr *v1alpha1.Scenario) error {
	/* Clone system configuration, needed to retrieve telemetry, chaos, etc  */
	sysconf, err := configuration.Get(ctx, r.GetClient(), r.Logger)
	if err != nil {
		return errors.Wrapf(err, "cannot get system configuration")
	}

	/* FIXME: we set the configuration be global here. is there any better way ? */
	configuration.SetGlobal(sysconf)

	{
		/*  Not the best place, but the webhook should start after we get the configuration parameters.
		Given that, we need to start it here, and only once. An alternative solution would be to get
		the webhook port and developer mode as parameters on the executable.
		*/
		startWebhookOnce.Do(func() {
			err = r.CreateWebhookServer(ctx, r.alertingPort)
		})

		if err != nil {
			return errors.Wrapf(err, "cannot create grafana webhook")
		}
	}

	// Ensure that the scenario is OK
	if errValidate := r.Validate(ctx, cr); errValidate != nil {
		return errors.Wrapf(errValidate, "malformed description")
	}

	// Start Prometheus + Grafana
	if errTelemetry := r.StartTelemetry(ctx, cr); errTelemetry != nil {
		return errors.Wrapf(errTelemetry, "cannot create the telemetry stack")
	}

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
		if err := common.ListChildren(ctx, r, &serviceJobs, req); err != nil {
			return errors.Wrapf(err, "unable to list child services for '%s'", req)
		}

		for i, job := range serviceJobs.Items {
			// Do not account telemetry jobs, unless they have failed.
			if job.GetName() == notRandomGrafanaName || job.GetName() == notRandomPrometheusName {
				r.view.Exclude(job.GetName(), &serviceJobs.Items[i])
			} else {
				r.view.Classify(job.GetName(), &serviceJobs.Items[i])
			}
		}
	}

	var clusterJobs v1alpha1.ClusterList
	{
		if err := common.ListChildren(ctx, r, &clusterJobs, req); err != nil {
			return errors.Wrapf(err, "unable to list child clusters for '%s'", req)
		}

		for i, job := range clusterJobs.Items {
			r.view.Classify(job.GetName(), &clusterJobs.Items[i])
		}
	}

	var chaosJobs v1alpha1.ChaosList
	{
		if err := common.ListChildren(ctx, r, &chaosJobs, req); err != nil {
			return errors.Wrapf(err, "unable to list child chaos for '%s'", req)
		}

		for i, job := range chaosJobs.Items {
			r.view.Classify(job.GetName(), &chaosJobs.Items[i])
		}
	}

	var cascadeJobs v1alpha1.CascadeList
	{
		if err := common.ListChildren(ctx, r, &cascadeJobs, req); err != nil {
			return errors.Wrapf(err, "unable to list child cascades for '%s'", req)
		}

		for i, job := range cascadeJobs.Items {
			r.view.Classify(job.GetName(), &cascadeJobs.Items[i])
		}
	}

	var virtualJobs v1alpha1.VirtualObjectList
	{
		if err := common.ListChildren(ctx, r, &virtualJobs, req); err != nil {
			return errors.Wrapf(err, "unable to list child virtualobjects for '%s'", req)
		}

		for i, job := range virtualJobs.Items {
			r.view.Classify(job.GetName(), &virtualJobs.Items[i])
		}
	}

	var callJobs v1alpha1.CallList
	{
		if err := common.ListChildren(ctx, r, &callJobs, req); err != nil {
			return errors.Wrapf(err, "unable to list child calls for '%s'", req)
		}

		for i, job := range callJobs.Items {
			r.view.Classify(job.GetName(), &callJobs.Items[i])
		}
	}

	return nil
}

func (r *Controller) HasSucceed(ctx context.Context, cr *v1alpha1.Scenario) error {
	r.GetEventRecorderFor(cr.GetName()).Event(cr, corev1.EventTypeNormal,
		cr.Status.Lifecycle.Reason, cr.Status.Lifecycle.Message)

	r.Logger.Info("CleanOnSuccess",
		"name", cr.GetName(),
		"successfulJobs", r.view.ListSuccessfulJobs(),
	)

	// Remove the testing components once the experiment is successfully complete.
	// We maintain testbed components (e.g, prometheus and grafana) for getting back the test results.
	// These components are removed by deleting the Scenario.
	for _, job := range r.view.GetSuccessfulJobs() {
		if v1alpha1.GetComponentLabel(job) == v1alpha1.ComponentSUT { // System services should not be removed

			expressions.UnsetAlert(job)
			common.Delete(ctx, r, job)
		}
	}

	return nil
}

func (r *Controller) HasFailed(ctx context.Context, cr *v1alpha1.Scenario) error {
	r.GetEventRecorderFor(cr.GetName()).Event(cr, corev1.EventTypeWarning,
		cr.Status.Lifecycle.Reason, cr.Status.Lifecycle.Message)

	r.Logger.Error(errors.New("Resource has failed"), "CleanOnFailure",
		"name", cr.GetName(),
		"successfulJobs", r.view.ListSuccessfulJobs(),
		"runningJobs", r.view.ListRunningJobs(),
		"pendingJobs", r.view.ListPendingJobs(),
		"reason", cr.Status.Reason,
		"message", cr.Status.Message)

	// TODO: What should we do when a call action fails ? Should we delete all services ?

	// Remove the non-failed components. Leave the failed jobs and system jobs for postmortem analysis.
	for _, job := range r.view.GetPendingJobs() {
		if v1alpha1.GetComponentLabel(job) == v1alpha1.ComponentSUT { // System jobs should not be deleted
			r.GetEventRecorderFor("").Event(job, corev1.EventTypeWarning, "Terminating", cr.Status.Message)

			expressions.UnsetAlert(job)
			common.Delete(ctx, r, job)
		}
	}

	for _, job := range r.view.GetRunningJobs() {
		if v1alpha1.GetComponentLabel(job) == v1alpha1.ComponentSUT { // System jobs should not be deleted
			r.GetEventRecorderFor("").Event(job, corev1.EventTypeWarning, "Terminating", cr.Status.Message)

			expressions.UnsetAlert(job)
			common.Delete(ctx, r, job)
		}
	}

	for _, job := range r.view.GetSuccessfulJobs() { // System jobs should not be deleted
		if v1alpha1.GetComponentLabel(job) == v1alpha1.ComponentSUT {
			r.GetEventRecorderFor("").Event(job, corev1.EventTypeWarning, "Terminating", cr.Status.Message)

			expressions.UnsetAlert(job)
			common.Delete(ctx, r, job)
		}
	}

	r.StopTelemetry(cr)

	// Suspend the workflow from creating new job.
	suspend := true
	cr.Spec.Suspend = &suspend

	if err := common.Update(ctx, r, cr); err != nil {
		return errors.Wrapf(err, "unable to suspend execution for '%s'", cr.GetName())
	}

	return nil
}

func (r *Controller) RunActions(ctx context.Context, t *v1alpha1.Scenario, actionList []v1alpha1.Action) error {
	endpoints := r.supportedActions()
	for _, action := range actionList {

		if action.Assert.HasMetricsExpr() {
			// Assert belong to the top-level workflow. Not to the job
			if err := expressions.SetAlert(ctx, t, action.Assert.Metrics); err != nil {
				return errors.Wrapf(err, "assertion error")
			}
		}

		e, ok := endpoints[action.ActionType]
		if !ok {
			return errors.Errorf("unknown type [%s] for action [%s]", action.ActionType, action.Name)
		}

		job, err := e(ctx, t, action)
		if err != nil {
			return errors.Wrapf(err, "action failed [%s]", action.Name)
		}

		// Some jobs are virtual and do not require something to be created.
		// Yet, they must be tracked in order to respect the execution dependencies of the graph.
		if action.ActionType != v1alpha1.ActionDelete {
			if err := common.Create(ctx, r, t, job); err != nil {
				return errors.Wrapf(err, "execution error for action '%s'", action.Name)
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
		r.GetEventRecorderFor(t.GetName()).Event(t, corev1.EventTypeNormal, "Scheduled", action.Name)

		t.Status.ScheduledJobs = append(t.Status.ScheduledJobs, action.Name)
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

func NewController(mgr ctrl.Manager, logger logr.Logger, alertingPort int) error {
	// instantiate the controller
	r := &Controller{
		Manager:      mgr,
		Logger:       logger.WithName("scenario"),
		alertingPort: alertingPort,
	}

	gvk := v1alpha1.GroupVersion.WithKind("Scenario")

	return ctrl.NewControllerManagedBy(mgr).
		Named("scenario").
		For(&v1alpha1.Scenario{}).
		Owns(&v1alpha1.Service{}, watchers.Watch(r, gvk)).       // Watch Services
		Owns(&v1alpha1.Cluster{}, watchers.Watch(r, gvk)).       // Watch Cluster
		Owns(&v1alpha1.Chaos{}, watchers.Watch(r, gvk)).         // Watch Chaos
		Owns(&v1alpha1.Cascade{}, watchers.Watch(r, gvk)).       // Watch Cascade
		Owns(&v1alpha1.VirtualObject{}, watchers.Watch(r, gvk)). // Watch VirtualObjects
		Owns(&v1alpha1.Call{}, watchers.Watch(r, gvk)).          // Watch Calls
		Complete(r)
}
