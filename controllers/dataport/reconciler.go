package dataport

import (
	"context"
	"strings"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewController(mgr ctrl.Manager, logger logr.Logger) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.DataPort{}).
		Named("dataport").
		Complete(&Reconciler{
			Client: mgr.GetClient(),
			Logger: logger.WithName("dataport"),
		})
}

// +kubebuilder:rbac:groups=frisbee.io,resources=dataports,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.io,resources=dataports/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.io,resources=dataports/finalizers,verbs=update

// Reconciler reconciles a Reference object
type Reconciler struct {
	client.Client
	logr.Logger
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var obj v1alpha1.DataPort

	var ret bool
	result, err := common.Reconcile(ctx, r, req, &obj, &ret)
	if ret {
		return result, err
	}

	// The reconcile logic
	switch obj.Status.Phase {
	case v1alpha1.PhaseUninitialized:
		return r.dispatch(ctx, &obj)

	case v1alpha1.PhasePending: // if we're here, we haven't started yet
		return common.DoNotRequeue()

	case v1alpha1.PhaseRunning:
		return common.DoNotRequeue()

	case v1alpha1.PhaseSuccess: // If we're PhaseSuccess but not deleted yet, nothing to do but return
		return common.DoNotRequeue()

	case v1alpha1.PhaseFailed: // if we're here, then something went completely wrong
		r.Logger.Info("Service failed", "name", obj.GetName())

		return common.DoNotRequeue()

	case v1alpha1.PhaseChaos: // if we're here, a controlled failure has occurred.
		r.Logger.Info("Service consumed by PhaseChaos", "service", obj.GetName())

		return common.DoNotRequeue()

	default:
		return common.Failed(ctx, &obj, errors.Errorf("unknown phase: %s", obj.Status.Phase))
	}
}

func (r *Reconciler) Finalizer() string {
	return "services.frisbee.io/finalizer"
}

func (r *Reconciler) Finalize(obj client.Object) error {
	r.Logger.Info("Finalize", "service", obj.GetName())

	return nil
}

func (r *Reconciler) dispatch(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	switch strings.ToLower(obj.Spec.Protocol) {
	case "direct":
		return r.direct(ctx, obj)

	default:
		return common.Failed(ctx, obj, errors.New("invalid protocol"))
	}
}
