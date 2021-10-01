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

package workflow

import (
	"context"
	"reflect"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/utils"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=frisbee.io,resources=workflows,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.io,resources=workflows/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.io,resources=workflows/finalizers,verbs=update

type Controller struct {
	ctrl.Manager
	logr.Logger

	state utils.LifecycleClassifier

	prometheus chan *v1alpha1.Lifecycle
	grafana    chan *v1alpha1.Lifecycle
}

func (r *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		1: Load CR by name.
		------------------------------------------------------------------
	*/
	var w v1alpha1.Workflow

	var requeue bool
	result, err := utils.Reconcile(ctx, r, req, &w, &requeue)

	if requeue {
		return result, errors.Wrapf(err, "initialization error")
	}

	r.Logger.Info("-> Reconcile",
		"kind", reflect.TypeOf(w),
		"name", w.GetName(),
		"lifecycle", w.Status.Phase,
		"epoch", w.GetResourceVersion(),
	)

	defer func() {
		r.Logger.Info("<- Reconcile",
			"kind", reflect.TypeOf(w),
			"name", w.GetName(),
			"lifecycle", w.Status.Phase,
			"epoch", w.GetResourceVersion(),
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
		client.MatchingLabels{utils.Owner: req.Name},
		//	client.MatchingFields{jobOwnerKey: req.Name},
	}

	var serviceJobs v1alpha1.ServiceList

	if err := r.GetClient().List(ctx, &serviceJobs, filters...); err != nil {
		return utils.Failed(ctx, r, &w, errors.Wrapf(err, "unable to list child serviceJobs"))
	}

	var clusterJobs v1alpha1.ClusterList

	if err := r.GetClient().List(ctx, &clusterJobs, filters...); err != nil {
		return utils.Failed(ctx, r, &w, errors.Wrapf(err, "unable to list child clusterJobs"))
	}

	var chaosJobs v1alpha1.ChaosList

	if err := r.GetClient().List(ctx, &chaosJobs, filters...); err != nil {
		return utils.Failed(ctx, r, &w, errors.Wrapf(err, "unable to list child chaosJobs"))
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

		However, to relief the garbage collector, we use a root structure that we reset
		at every reconciliation cycle.
	*/
	r.state.Reset()

	for _, job := range serviceJobs.Items {
		r.state.Classify(job.GetName(), job.DeepCopy())
	}

	for _, job := range clusterJobs.Items {
		r.state.Classify(job.GetName(), job.DeepCopy())
	}

	for _, job := range chaosJobs.Items {
		r.state.Classify(job.GetName(), job.DeepCopy())
	}

	/*
		4: Update the CR status using the data we've gathered
		------------------------------------------------------------------

		Just like before, we use our client.  To specifically update the status
		subresource, we'll use the `Status` part of the client, with the `Update`
		method.
	*/

	newStatus := calculateLifecycle(&w, r.state)
	w.Status.Lifecycle = newStatus

	if err := utils.UpdateStatus(ctx, r, &w); err != nil {
		// due to the multiple updates, it is possible for this function to
		// be in conflict. We fix this issue by re-queueing the request.
		// We also omit verbose error reporting as to avoid polluting the output.
		runtime.HandleError(err)

		return utils.Requeue()
	}

	/*
		5: Clean up the controller from finished jobs
		------------------------------------------------------------------

		First, we'll try to clean up old jobs, so that we don't leave too many lying
		around.
	*/

	if newStatus.Phase == v1alpha1.PhaseSuccess {
		r.Logger.Info("Workflow succeeded", "name", w.GetName())

		// Remove the testing components once the experiment is successfully complete.
		// We maintain testbed components (e.g, prometheus and grafana) for getting back the test results.
		// These components are removed by deleting the Workflow.
		for _, job := range r.state.SuccessfulJobs() {
			utils.Delete(ctx, r, job)
		}

		return utils.Stop()
	}

	if newStatus.Phase == v1alpha1.PhaseFailed {
		r.Logger.Error(errors.New(w.Status.Reason), "Workflow failed", "name", w.GetName())

		// utils.Delete(ctx, r, &w)

		return utils.Stop()
	}

	/*
		6: Make the world matching what we want in our spec
		------------------------------------------------------------------

		Once we've updated our status, we can move on to ensuring that the status of
		the world matches what we want in our spec.

		We may delete the service, add a pod, or wait for existing pod to change its status.
	*/

	/*
		If all actions are executed, just wait for them to complete.
	*/
	if newStatus.Phase == v1alpha1.PhaseRunning {
		return utils.Stop()
	}

	/*
		If this object is suspended, we don't want to run any jobs, so we'll stop now.
		This is useful if something's broken with the job we're running and we want to
		pause runs to investigate or putz with the cluster, without deleting the object.
	*/
	if w.Spec.Suspend != nil && *w.Spec.Suspend {
		r.Logger.Info("Not starting job because the workflow is suspended",
			"workflow", w.GetName())

		return utils.Stop()
	}

	/*
		If we are not suspended, initialize the workflow. At this step, we create the observability stack
	*/
	if w.Status.Phase == v1alpha1.PhaseUninitialized {
		logrus.Warn(" -- Observability Stack --")

		if err := r.newMonitoringStack(ctx, &w); err != nil {
			return utils.Failed(ctx, r, &w, errors.Wrapf(err, "unable to create the observability stack"))
		}

		w.Status.Scheduled = make(map[string]bool)

		return utils.Pending(ctx, r, &w, "monitoring stack has been started")
	}

	/*
		7: Run jobs that meeting the logical dependencies
		------------------------------------------------------------------
	*/

	// TODO: add support for time dependencies
	actionList := GetNextLogicalJob(w.Spec.Actions, r.state, w.Status.Scheduled)

	if len(actionList) == 0 {
		return utils.Stop()
	}

	logrus.Warn("Ready to start ", actionList)

	for _, action := range actionList {
		if r.state.IsActive(action.Name) || isJobInScheduledList(action.Name, w.Status.Scheduled) {
			r.Logger.Info("Not starting action because it is already processed", "action", action.Name)
		}

		if err := r.runJob(ctx, &w, action); err != nil {
			return utils.Failed(ctx, r, &w, errors.Wrapf(err, "waiting failed"))
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

	// Add the just-started jobs to the status list.
	for _, action := range actionList {
		w.Status.Scheduled[action.Name] = true
	}

	return utils.Pending(ctx, r, &w, "some jobs are still pending")
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
		"epoch", obj.GetResourceVersion(),
	)
	return nil
}

/*
### Setup
	Finally, we'll update our setup.

	We'll inform the manager that this controller owns some Services, so that it
	will automatically call Reconcile on the underlying Service when a Pod changes, is
	deleted, etc.
*/

var controllerKind = v1alpha1.GroupVersion.WithKind("Workflow")

func NewController(mgr ctrl.Manager, logger logr.Logger) error {
	r := &Controller{
		Manager:    mgr,
		Logger:     logger.WithName("workflow"),
		prometheus: make(chan *v1alpha1.Lifecycle),
		grafana:    make(chan *v1alpha1.Lifecycle),
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named("workflow").
		For(&v1alpha1.Workflow{}).
		Owns(&v1alpha1.Service{}, builder.WithPredicates(r.WatchServices())).
		Owns(&v1alpha1.Cluster{}, builder.WithPredicates(r.WatchClusters())).
		Complete(r)
}

// isJobInScheduledList take a job and checks if activeJobs has a job with the same
// name and namespace.
func isJobInScheduledList(name string, scheduledJobs map[string]bool) bool {
	_, ok := scheduledJobs[name]

	return ok
}
