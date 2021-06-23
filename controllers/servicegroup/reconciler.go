package servicegroup

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
		For(&v1alpha1.ServiceGroup{}).
		Named("servicegroup").
		Complete(&Reconciler{
			Client: mgr.GetClient(),
			Logger: logger.WithName("servicegroup"),
		})
}

// +kubebuilder:rbac:groups=frisbee.io,resources=servicegroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.io,resources=servicegroups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.io,resources=servicegroups/finalizers,verbs=update

// Reconciler reconciles a Templates object
type Reconciler struct {
	client.Client
	logr.Logger
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var obj v1alpha1.ServiceGroup

	var ret bool
	result, err := common.Reconcile(ctx, r, req, &obj, &ret)
	if ret {
		return result, err
	}

	// The reconcile logic
	switch obj.Status.Phase {
	case v1alpha1.Uninitialized:
		r.Logger.Info("Create group", "name", obj.GetName())

		return r.create(ctx, &obj)

	case v1alpha1.Running: // if we're here, then we're either still running or haven't started yet

		return common.DoNotRequeue()

	case v1alpha1.Succeed: // If we're Complete but not deleted yet, nothing to do but return
		r.Logger.Info("Group completed", "name", obj.GetName())

		return common.DoNotRequeue()

	case v1alpha1.Failed: // if we're here, then something went completely wrong
		r.Logger.Info("Group failed", "name", obj.GetName())

		return common.DoNotRequeue()

	default:
		r.Logger.Info("unknown status", "phase", obj.Status.Phase)
		return common.DoNotRequeue()
	}
}

func (r *Reconciler) Finalizer() string {
	return "servicegroups.frisbee.io/finalizer"
}

func (r *Reconciler) Finalize(obj client.Object) error {
	return nil
}
