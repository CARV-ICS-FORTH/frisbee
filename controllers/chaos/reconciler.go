package chaos

import (
	"context"
	"reflect"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=frisbee.io,resources=chaoss,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.io,resources=chaoss/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.io,resources=chaoss/finalizers,verbs=update

func NewController(mgr ctrl.Manager, logger logr.Logger) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Chaos{}).
		Named("chaos").
		Complete(&Reconciler{
			Client: mgr.GetClient(),
			Logger: logger.WithName("chaos"),
		})
}

// Reconciler reconciles a Reference object
type Reconciler struct {
	client.Client
	logr.Logger
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var obj v1alpha1.Chaos

	var ret bool
	result, err := common.Reconcile(ctx, r, req, &obj, &ret)
	if ret {
		return result, err
	}

	r.Logger.Info("-> Reconcile", "kind", reflect.TypeOf(obj), "name", obj.GetName(), "lifecycle", obj.Status.Phase)
	defer func() {
		r.Logger.Info("<- Reconcile", "kind", reflect.TypeOf(obj), "name", obj.GetName(), "lifecycle", obj.Status.Phase)
	}()

	handler := r.dispatch(obj.Spec.Type)

	// The reconcile logic
	switch obj.Status.Phase {
	case v1alpha1.PhaseUninitialized:
		return handler.Inject(ctx, &obj)

	case v1alpha1.PhaseRunning:
		return handler.WaitForDuration(ctx, &obj)

	case v1alpha1.PhaseSuccess:
		// Chaos is Success, meaning that duration has elapsed.
		// Revoke the chaos
		return handler.Revoke(ctx, &obj)

	case v1alpha1.PhaseFailed:
		r.Logger.Info("Chaos failed", "name", obj.GetName())

		return common.DoNotRequeue()

	case v1alpha1.PhaseChaos, v1alpha1.PhaseDiscoverable, v1alpha1.PhasePending:
		// These phases should not happen in the workflow
		panic(errors.Errorf("invalid lifecycle phase %s", obj.Status.Phase))

	default:
		panic(errors.Errorf("unknown lifecycle phase: %s", obj.Status.Phase))
	}
}

func (r *Reconciler) Finalizer() string {
	return "chaoss.frisbee.io/finalizer"
}

func (r *Reconciler) Finalize(obj client.Object) error {
	r.Logger.Info("Finalize", "kind", reflect.TypeOf(obj), "name", obj.GetName())

	return nil
}

type chaoHandler interface {
	Inject(ctx context.Context, obj *v1alpha1.Chaos) (ctrl.Result, error)
	WaitForDuration(ctx context.Context, obj *v1alpha1.Chaos) (ctrl.Result, error)
	Revoke(ctx context.Context, obj *v1alpha1.Chaos) (ctrl.Result, error)
}

func (r *Reconciler) dispatch(faultType v1alpha1.FaultType) chaoHandler {
	switch faultType {
	case v1alpha1.FaultPartition:
		return &partition{r: r}

	default:
		panic("should never happen")
	}
}
