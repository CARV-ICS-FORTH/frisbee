package service

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
		For(&v1alpha1.Service{}).
		Named("service").
		Complete(&Reconciler{
			Client: mgr.GetClient(),
			Logger: logger.WithName("service"),
		})
}

// +kubebuilder:rbac:groups=frisbee.io,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.io,resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.io,resources=services/finalizers,verbs=update

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;

// Reconciler reconciles a Reference object
type Reconciler struct {
	client.Client
	logr.Logger
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var obj v1alpha1.Service

	var ret bool
	result, err := common.Reconcile(ctx, r, req, &obj, &ret)
	if ret {
		return result, err
	}

	// The reconcile logic
	switch obj.Status.Phase {
	case v1alpha1.Uninitialized:
		return r.create(ctx, &obj)

	case v1alpha1.Running: // if we're here, then we're either still running or haven't started yet
		/*
			r.Logger.Info("Service is already running",
				"name", obj.GetName(),
				"CreationTimestamp", obj.CreationTimestamp.String(),
			)

		*/

		return common.DoNotRequeue()

	case v1alpha1.Complete: // If we're Complete but not deleted yet, nothing to do but return
		r.Logger.Info("Service completed", "name", obj.GetName())

		if err := r.Client.Delete(ctx, &obj); err != nil {
			r.Logger.Error(err, "unable to delete object", "object", obj.GetName())
		}

		return common.DoNotRequeue()

	case v1alpha1.Failed: // if we're here, then something went completely wrong
		r.Logger.Info("Service failed", "name", obj.GetName())

		return common.DoNotRequeue()

	case v1alpha1.Chaos: // if we're here, a controlled failure has occurred.
		r.Logger.Info("Service consumed by Chaos", "service", obj.GetName())

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

func (r *Reconciler) create(ctx context.Context, obj *v1alpha1.Service) (ctrl.Result, error) {
	// ingress

	// kubepod (the lifecycle of service is driven by the pod)
	if err := r.createKubePod(ctx, obj); err != nil {
		return common.Failed(ctx, obj, err)
	}

	// nic

	/*
		go func() {
			time.Sleep(30 * time.Second)
			r.changePhase(ctx, obj, v1alpha1.Running, "Started")

			wait := 30 * time.Second
			if strings.HasPrefix(obj.GetName(), "masters") {
				wait = 30 * time.Minute
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(wait):
				r.changePhase(ctx, obj, v1alpha1.Complete, "mock time elapsed")
				return
			}
		}()

	*/

	return common.DoNotRequeue()
}
