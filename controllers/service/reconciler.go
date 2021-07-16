package service

import (
	"context"
	"fmt"
	"reflect"

	"github.com/fatih/structs"
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

	r.Logger.Info("-> Reconcile", "kind", reflect.TypeOf(obj), "name", obj.GetName(), "lifecycle", obj.Status.Phase)
	defer func() {
		r.Logger.Info("<- Reconcile", "kind", reflect.TypeOf(obj), "name", obj.GetName(), "lifecycle", obj.Status.Phase)
	}()

	// The reconcile logic
	switch obj.Status.Phase {
	case v1alpha1.PhaseUninitialized:

		if len(obj.Spec.PortRefs) > 0 {
			return common.Discoverable(ctx, &obj)
		}

		return common.Pending(ctx, &obj)

	case v1alpha1.PhaseDiscoverable:
		return r.discoverDataMesh(ctx, &obj)

	case v1alpha1.PhasePending:
		if err := r.createKubePod(ctx, &obj); err != nil {
			return common.Failed(ctx, &obj, err)
		}

		// if we're here, the lifecycle of service is driven by the pod
		return common.DoNotRequeue()

	case v1alpha1.PhaseRunning:
		/*
			r.Logger.Info("Service is already running",
				"name", obj.GetName(),
				"CreationTimestamp", obj.CreationTimestamp.String(),
			)

		*/

		return common.DoNotRequeue()

	case v1alpha1.PhaseSuccess: // If we're PhaseSuccess but not deleted yet, nothing to do but return
		r.Logger.Info("Service completed", "name", obj.GetName())

		if err := r.Client.Delete(ctx, &obj); err != nil {
			r.Logger.Error(err, "unable to delete object", "object", obj.GetName())
		}

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
	r.Logger.Info("Finalize", "kind", reflect.TypeOf(obj), "name", obj.GetName())

	return nil
}

func (r *Reconciler) discoverDataMesh(ctx context.Context, obj *v1alpha1.Service) (ctrl.Result, error) {

	ports := make([]v1alpha1.DataPort, len(obj.Spec.PortRefs))

	// add ports
	for i, portRef := range obj.Spec.PortRefs {
		key := client.ObjectKey{
			Name:      portRef,
			Namespace: obj.GetNamespace(),
		}

		if err := r.Client.Get(ctx, key, &ports[i]); err != nil {
			return common.Failed(ctx, obj, errors.Wrapf(err, "port error"))
		}
	}

	// TODO: fix this crappy thing
	return r.direct(ctx, obj, &ports[0])

	/*
		var
		var err error

		// connect remote ports to local handlers
		for i, port := range ports {
			switch v := port.Spec.Protocol; v {
			case v1alpha1.Direct:
				err = r.direct(ctx, obj, &ports[i])

			default:
				return common.Failed(ctx, obj, errors.Errorf("invalid mesh protocol %s", v))
			}

			if err != nil {
				return errors.Wrapf(err, "data mesh failed")
			}
		}

	*/
}



func portStatusToAnnotations(portName string, proto v1alpha1.PortProtocol, status interface{}) map[string]string {
	if status == nil {
		return nil
	}

	ret := make(map[string]string)

	for key, value := range structs.Map(status) {
		ret[fmt.Sprintf("ports.%s.%s.%s", portName, proto, key)] = fmt.Sprint(value)
	}

	return ret
}