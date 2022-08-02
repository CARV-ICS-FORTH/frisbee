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

package template

import (
	"context"
	"reflect"
	"time"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/carv-ics-forth/frisbee/controllers/common/lifecycle"
	"github.com/go-logr/logr"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=frisbee.dev,resources=templates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.dev,resources=templates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.dev,resources=templates/finalizers,verbs=update

// Controller reconciles a Templates object.
type Controller struct {
	ctrl.Manager
	logr.Logger
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		1: Load CR by name.
		------------------------------------------------------------------
	*/
	var cr v1alpha1.Template

	// Use a slightly different approach than other controllers, since we do not need finalizers.
	if err := r.GetClient().Get(ctx, req.NamespacedName, &cr); err != nil {
		// Request object not found, could have been deleted after reconcile request.
		// We'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on added / deleted requests.
		if k8errors.IsNotFound(err) {
			return common.Stop()
		}

		r.Error(err, "obj retrieval")

		return common.RequeueAfter(time.Second)
	}

	/*
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
			2: Update the CR status using the data we've gathered
			------------------------------------------------------------------

		if err := common.UpdateStatus(ctx, r, &cr); err != nil {
			r.Info("Reschedule", "object", cr.GetName(), "UpdateStatusErr", err)

			return common.RequeueAfter(time.Second)
		}

	*/

	/*
		3: Clean up the controller from finished jobs
		------------------------------------------------------------------

		Not needed now.
	*/

	/*
		4: Make the world matching what we want in our spec
		------------------------------------------------------------------
	*/
	if cr.Status.Lifecycle.Phase.Is(v1alpha1.PhaseRunning) {
		return common.Stop()
	}

	if cr.Status.Lifecycle.Phase.Is(v1alpha1.PhaseUninitialized) {
		r.Logger.Info("Add Template",
			"name", req.NamespacedName,
		)

		return lifecycle.Running(ctx, r, &cr, "all templates are loaded")
	}

	return common.Stop()
}

/*
### Finalizers
*/

func (r *Controller) Finalizer() string {
	return ""
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
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Template{}).
		Named("template").
		Complete(&Controller{
			Manager: mgr,
			Logger:  logger.WithName("template"),
		})
}
