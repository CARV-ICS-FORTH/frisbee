package servicegroup

import (
	"context"
	"reflect"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/controllers/common/lifecycle"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=frisbee.io,resources=servicegroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.io,resources=servicegroups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.io,resources=servicegroups/finalizers,verbs=update

func NewController(mgr ctrl.Manager, logger logr.Logger) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ServiceGroup{}).
		Named("creategroup").
		Complete(&Reconciler{
			Client: mgr.GetClient(),
			Logger: logger.WithName("group"),
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
	var obj v1alpha1.ServiceGroup

	var ret bool
	result, err := common.Reconcile(ctx, r, req, &obj, &ret)
	if ret {
		return result, err
	}

	r.Logger.Info("-> Reconcile", "kind", reflect.TypeOf(obj), "name", obj.GetName(), "lifecycle", obj.Status.Phase)
	defer func() {
		r.Logger.Info("<- Reconcile", "kind", reflect.TypeOf(obj), "name", obj.GetName(), "lifecycle", obj.Status.Phase)
	}()

	// The reconcile logic
	switch obj.Status.Phase {
	case v1alpha1.PhaseUninitialized:
		if err := r.create(ctx, &obj); err != nil {
			return lifecycle.Failed(ctx, &obj, err)
		}

		return lifecycle.Pending(ctx, &obj, "waiting for services to become ready")

	case v1alpha1.PhasePending: // Managed by Lifecycle()
		return common.DoNotRequeue()

	case v1alpha1.PhaseRunning: // Passthroughs
		return common.DoNotRequeue()

	case v1alpha1.PhaseSuccess: // Passthroughs
		return common.DoNotRequeue()

	case v1alpha1.PhaseFailed:
		r.Logger.Info("ServiceGroup has failed", "name", obj.GetName())

		return common.DoNotRequeue()

	case v1alpha1.PhaseDiscoverable, v1alpha1.PhaseChaos: // Invalid
		panic(errors.Errorf("invalid lifecycle phase %s", obj.Status.Phase))

	default:
		panic(errors.Errorf("unknown lifecycle phase: %s", obj.Status.Phase))
	}
}

func (r *Reconciler) Finalizer() string {
	return "servicegroups.frisbee.io/finalizer"
}

func (r *Reconciler) Finalize(obj client.Object) error {
	r.Logger.Info("Finalize", "kind", reflect.TypeOf(obj), "name", obj.GetName())

	return nil
}
