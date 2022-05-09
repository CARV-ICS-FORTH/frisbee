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

package telemetry

import (
	"context"
	"reflect"
	"time"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	serviceutils "github.com/carv-ics-forth/frisbee/controllers/service/utils"
	"github.com/carv-ics-forth/frisbee/controllers/utils"
	"github.com/carv-ics-forth/frisbee/controllers/utils/lifecycle"
	"github.com/carv-ics-forth/frisbee/controllers/utils/watchers"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=frisbee.io,resources=telemetries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.io,resources=telemetries/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.io,resources=telemetries/finalizers,verbs=update

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services/status,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=endpoints,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=endpoints/status,verbs=get;list;watch

type Controller struct {
	ctrl.Manager
	logr.Logger

	state lifecycle.Classifier

	serviceControl serviceutils.ServiceControlInterface
}

func (r *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		1: Load CR by name.
		------------------------------------------------------------------
	*/
	var cr v1alpha1.Telemetry

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

	// validate
	if len(cr.Spec.ImportDashboards) == 0 {
		return utils.Stop()
	}

	var prometheus v1alpha1.Service
	{
		key := client.ObjectKey{
			Name:      "prometheus",
			Namespace: req.Namespace,
		}

		if err := r.GetClient().Get(ctx, key, &prometheus); client.IgnoreNotFound(err) != nil {
			return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "unable to get prometheus"))
		}
	}

	var grafana v1alpha1.Service
	{
		key := client.ObjectKey{
			Name:      "grafana",
			Namespace: req.Namespace,
		}

		if err := r.GetClient().Get(ctx, key, &grafana); client.IgnoreNotFound(err) != nil {
			return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "unable to get grafana"))
		}
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

	r.state.Classify(prometheus.GetName(), &prometheus)

	r.state.Classify(grafana.GetName(), &grafana)

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
	cr.SetReconcileStatus(calculateLifecycle(&cr, r.state))

	if err := utils.UpdateStatus(ctx, r, &cr); err != nil {
		r.Info("update status error. retry", "object", cr.GetName(), "err", err)
		return utils.RequeueAfter(time.Second)
	}

	/*
		6: Make the world matching what we want in our spec
		------------------------------------------------------------------

		Once we've updated our status, we can move on to ensuring that the status of
		the world matches what we want in our spec.

		We may delete the service, add a pod, or wait for existing pod to change its status.
	*/
	if cr.Status.Phase.Is(v1alpha1.PhaseUninitialized) {
		if err := r.installPrometheus(ctx, &cr); err != nil {
			return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "prometheus error"))
		}

		if err := r.installGrafana(ctx, &cr); err != nil {
			return lifecycle.Failed(ctx, r, &cr, errors.Wrapf(err, "grafana error"))
		}

		return lifecycle.Pending(ctx, r, &cr, "some jobs are still pending")
	}

	return utils.Stop()
}

/*
### Finalizers
*/

func (r *Controller) Finalizer() string {
	return "telemetries.frisbee.io/finalizer"
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

	// register the reconcile controller
	r := &Controller{
		Manager: mgr,
		Logger:  logger.WithName("Telemetry"),
	}

	r.serviceControl = serviceutils.NewServiceControl(r)
	gvk := v1alpha1.GroupVersion.WithKind("Telemetry")

	return ctrl.NewControllerManagedBy(mgr).
		Named("Telemetry").
		For(&v1alpha1.Telemetry{}).
		Owns(&v1alpha1.Service{}, watchers.WatchService(r, gvk)). // Watch Services
		Complete(r)
}
