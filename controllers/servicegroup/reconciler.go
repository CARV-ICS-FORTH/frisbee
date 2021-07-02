package servicegroup

import (
	"context"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewController(mgr ctrl.Manager, logger logr.Logger) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ServiceGroup{}).
		Named("creategroup").
		Complete(&Reconciler{
			Client: mgr.GetClient(),
			Logger: logger.WithName("group"),
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
		r.Logger.Info("ServiceGroup group", "name", obj.GetName())

		return r.create(ctx, &obj)

	case v1alpha1.Running: // if we're here, then we're either still running or haven't started yet
		r.Logger.Info("ServiceGroup is already running",
			"name", obj.GetName(),
			"CreationTimestamp", obj.CreationTimestamp.String(),
		)

		return common.DoNotRequeue()

	case v1alpha1.Complete: // If we're Complete but not deleted yet, nothing to do but return
		r.Logger.Info("Group completed", "name", obj.GetName())

		/*
			if err := r.Client.Delete(ctx, &obj); err != nil {
				runtimeutil.HandleError(err)
			}
		*/

		return common.DoNotRequeue()

	case v1alpha1.Failed: // if we're here, then something went completely wrong
		r.Logger.Info("Group failed", "name", obj.GetName())

		return common.DoNotRequeue()

	case v1alpha1.Chaos: // if we're here, a controlled failure has occurred.
		r.Logger.Info("ServiceGroup failed gracefully", "name", obj.GetName())

		return common.DoNotRequeue()

	default:
		return common.Failed(ctx, &obj, errors.Errorf("unknown phase: %s", obj.Status.Phase))
	}
}

func (r *Reconciler) Finalizer() string {
	return "servicegroups.frisbee.io/finalizer"
}

func (r *Reconciler) Finalize(obj client.Object) error {
	return nil
}
