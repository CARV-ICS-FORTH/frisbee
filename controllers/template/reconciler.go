package template

import (
	"context"
	"reflect"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewController(mgr ctrl.Manager, logger logr.Logger) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Template{}).
		Named("template").
		Complete(&Reconciler{
			Client: mgr.GetClient(),
			Logger: logger.WithName("template"),
		})
}

// Reconciler reconciles a Templates object
type Reconciler struct {
	client.Client
	logr.Logger
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var obj v1alpha1.Template

	var ret bool
	result, err := common.Reconcile(ctx, r, req, &obj, &ret)
	if ret {
		return result, err
	}

	// if the template is already registered, there is nothing else to do.
	if obj.Status.IsRegistered {
		return common.DoNotRequeue()
	}

	serviceNames := make([]string, 0, len(obj.Spec.Services))
	for name := range obj.Spec.Services {
		serviceNames = append(serviceNames, name)
	}

	monitorNames := make([]string, 0, len(obj.Spec.Monitors))
	for name := range obj.Spec.Monitors {
		monitorNames = append(monitorNames, name)
	}

	r.Logger.Info("Import Template",
		"name", req.NamespacedName,
		"services", serviceNames,
		"monitor", monitorNames,
	)

	return common.DoNotRequeue()
}

func (r *Reconciler) Finalizer() string {
	return "templates.frisbee.io/finalizer"
}

func (r *Reconciler) Finalize(obj client.Object) error {
	r.Logger.Info("Finalize", "kind", reflect.TypeOf(obj), "name", obj.GetName())

	return nil
}
