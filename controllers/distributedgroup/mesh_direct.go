// Licensed to FORTH/ICS under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. FORTH/ICS licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package distributedgroup

import (
	"context"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *Reconciler) direct(ctx context.Context, obj *v1alpha1.ServiceSpec, port *v1alpha1.DataPort) (ctrl.Result, error) {
	/*
		logrus.Warnf("-> Bind service %s with port %s", obj.GetName(), port.GetName())
		defer logrus.Warnf("<- Bind service %s with port %s", obj.GetName(), port.GetName())

		// wait for port to become ready
		err := lifecycle.New(ctx,
			lifecycle.Watch(port, port.GetName()),
			lifecycle.WithLogger(r.Logger),
		).Until(v1alpha1.PhaseRunning, port)
		if err != nil {
			return lifecycle.Failed(ctx, obj, errors.Wrapf(err, "waiting for port %s failed", port.GetName()))
		}

		// convert status of the remote port to local annotations that will be used by the ENV.
		annotations := service.portStatusToAnnotations(port.GetName(), port.Spec.Protocol, port.GetProtocolStatus())
		if len(annotations) == 0 {
			panic("empty annotations")
		}

		if structure.Contains(obj.GetAnnotations(), annotations) {
			return lifecycle.Pending(ctx, obj, "wait for dataport to become ready")
		}

		obj.SetAnnotations(labels.Merge(obj.GetAnnotations(), annotations))

		// update is needed because the mesh operations may change labels and annotations
		return common.Update(ctx, obj)

	*/

	return common.Stop()
}

/*


// +kubebuilder:rbac:groups=frisbee.io,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.io,resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.io,resources=services/finalizers,verbs=update

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;

func NewController(mgr ctrl.Manager, logger logr.Logger) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Service{}).
		Named("service").
		Complete(&Reconciler{
			Client: mgr.GetClient(),
			Logger: logger.WithName("service"),
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
			return r.discoverDataMesh(ctx, &obj)
		}

		return lifecycle.Pending(ctx, &obj, "dependencies resolved. create pods...")

	case v1alpha1.PhasePending:
		if err := r.createKubePod(ctx, &obj); err != nil {
			return lifecycle.Failed(ctx, &obj, err)
		}

		// if we're here, the lifecycle of service is driven by the pod
		return common.Stop()

	case v1alpha1.PhaseRunning:
		/ *
			r.Logger.Info("Service is already running",
				"name", obj.GetName(),
				"CreationTimestamp", obj.CreationTimestamp.String(),
			)

		* /

		return common.Stop()

	case v1alpha1.PhaseSuccess: // If we're PhaseSuccess but not deleted yet, nothing to do but return
		r.Logger.Info("Service completed", "name", obj.GetName())

		if err := lifecycle.Delete(ctx, r.Client, &obj); err != nil {
			r.Logger.Error(err, "garbage collection error", "object", obj.GetName())
		}

		return common.Stop()

	case v1alpha1.PhaseFailed: // if we're here, then something went completely wrong
		r.Logger.Info("Service failed. Omit garbage collection for debugging purposes.", "name", obj.GetName())

		return common.Stop()

	case v1alpha1.PhaseChaos: // if we're here, a controlled failure has occurred.
		r.Logger.Info("Service consumed by PhaseChaos", "service", obj.GetName())

		return common.Stop()

	default:
		return lifecycle.Failed(ctx, &obj, errors.Errorf("unknown phase: %s", obj.Status.Phase))
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
			return lifecycle.Failed(ctx, obj, errors.Wrapf(err, "port error"))
		}
	}

	// TODO: fix this crappy thing
	return r.direct(ctx, obj, &ports[0])

	/ *
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

	* /
}

// portStatusAnnotations translates a Status struct to annotations that will be used for rewiring the service's dataports.
func portStatusToAnnotations(portName string, proto v1alpha1.PortProtocol, status interface{}) map[string]string {
	if status == nil {
		panic("empty status")
	}

	val := reflect.ValueOf(status)

	switch {
	case val.IsNil(), val.IsZero():
		return nil
	case val.CanInterface():
		status = val.Interface()
	default:
		panic("invalid type")
	}

	ret := make(map[string]string)

	s := structs.New(status)

	for _, f := range s.Fields() {
		ret[fmt.Sprintf("ports.%s.%s.%s", portName, proto, f.Name())] = fmt.Sprint(f.Value())
	}

	return ret
}

*/
