package template

import (
	"context"

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

	serviceNames := make([]string, 0, len(obj.Spec.Services))
	for name := range obj.Spec.Services {
		serviceNames = append(serviceNames, name)
	}

	r.Logger.Info("Register templates",
		"name", req.NamespacedName,
		"services", serviceNames,
	)

	return common.DoNotRequeue()
}

func (r *Reconciler) Finalizer() string {
	return "templates.frisbee.io/finalizer"
}

func (r *Reconciler) Finalize(obj client.Object) error {
	// delete any external resources associated with the service
	// Examples finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.
	//
	// Ensure that delete implementation is idempotent and safe to invoke
	// multiple times for same object
	return nil
}
