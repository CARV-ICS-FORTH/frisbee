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
	"github.com/carv-ics-forth/frisbee/controllers/utils"
	"github.com/carv-ics-forth/frisbee/controllers/utils/lifecycle"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=frisbee.io,resources=templates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.io,resources=templates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.io,resources=templates/finalizers,verbs=update

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

	var requeue bool
	result, err := utils.Reconcile(ctx, r, req, &cr, &requeue)

	if requeue {
		return result, errors.Wrapf(err, "initialization error")
	}

	/*
		2: Update the CR status using the data we've gathered
		------------------------------------------------------------------
	*/
	if err := utils.UpdateStatus(ctx, r, &cr); err != nil {
		runtime.HandleError(err)

		return utils.RequeueAfter(time.Second)
	}

	/*
		3: Clean up the controller from finished jobs
		------------------------------------------------------------------

		Not needed now.
	*/

	/*
		4: Make the world matching what we want in our spec
		------------------------------------------------------------------
	*/
	if cr.Status.Lifecycle.Phase == v1alpha1.PhaseRunning {
		return utils.Stop()
	}

	if cr.Status.Lifecycle.Phase == v1alpha1.PhaseUninitialized {
		r.Logger.Info("Import Group",
			"name", req.NamespacedName,
		)

		return lifecycle.Running(ctx, r, &cr, "all templates are loaded")
	}

	return utils.Stop()
}

/*
### Finalizers
*/

func (r *Controller) Finalizer() string {
	return "templates.frisbee.io/finalizer"
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
