package service

import (
	"context"
	"time"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/go-logr/logr"
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

// Reconciler reconciles a Service object
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
		r.Logger.Info("Create service", "name", obj.GetName())

		return r.create(ctx, &obj)

	case v1alpha1.Running: // if we're here, then we're either still running or haven't started yet
		return common.DoNotRequeue()

	case v1alpha1.Succeed: // If we're Complete but not deleted yet, nothing to do but return
		r.Logger.Info("Service completed", "name", obj.GetName())

		return common.DoNotRequeue()

	case v1alpha1.Failed: // if we're here, then something went completely wrong
		r.Logger.Info("Service failed", "name", obj.GetName())

		return common.DoNotRequeue()

	default:
		r.Logger.Info("unknown status", "phase", obj.Status.Phase)
		return common.DoNotRequeue()
	}
}

func (r *Reconciler) Finalizer() string {
	return "services.frisbee.io/finalizer"
}

func (r *Reconciler) Finalize(obj client.Object) error {

	// delete any external resources associated with the service
	// Examples finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.
	//
	// Ensure that delete implementation is idempotent and safe to invoke
	// multiple times for same object

	r.Logger.Info("Finalize", "service", obj.GetName())

	return nil
}

func (r *Reconciler) create(ctx context.Context, obj *v1alpha1.Service) (ctrl.Result, error) {

	/*
		// discovery service
		if err := createKubeService(ctx, instance, r); err != nil {
			return err
		}

		// ingress

		// configmap
		volumes, mounts, err := createKubeConfigMap(ctx, instance, r)
		if err != nil {
			return err
		}

		// kubepod
		if err := createKubePod(ctx, instance, r, volumes, mounts); err != nil {
			return err
		}

		// nic

	*/

	go func() {
		time.Sleep(30 * time.Second)
		r.changePhase(ctx, obj, v1alpha1.Running, "Started")

		select {
		case <-ctx.Done():
			return
		case <-time.After(30 * time.Second):
			r.changePhase(ctx, obj, v1alpha1.Succeed, "mock time elapsed")
			return
		}
	}()

	return common.DoNotRequeue()
}

func (r *Reconciler) changePhase(ctx context.Context, obj *v1alpha1.Service, newPhase v1alpha1.Phase, reason string) {
	obj.Status.Phase = newPhase
	obj.Status.Reason = reason

	common.UpdateStatus(ctx, obj)
}
