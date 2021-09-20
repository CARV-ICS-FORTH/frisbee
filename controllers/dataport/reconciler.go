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

package dataport

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

// +kubebuilder:rbac:groups=frisbee.io,resources=dataports,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.io,resources=dataports/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.io,resources=dataports/finalizers,verbs=update

func NewController(mgr ctrl.Manager, logger logr.Logger) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.DataPort{}).
		Named("dataport").
		Complete(&Reconciler{
			Client: mgr.GetClient(),
			Logger: logger.WithName("dataport"),
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
	var obj v1alpha1.DataPort

	var ret bool
	result, err := common.Reconcile(ctx, r, req, &obj, &ret)
	if ret {
		return result, err
	}

	r.Logger.Info("-> Reconcile", "kind", reflect.TypeOf(obj), "name", obj.GetName(), "lifecycle", obj.Status.Phase)
	defer func() {
		r.Logger.Info("<- Reconcile", "kind", reflect.TypeOf(obj), "name", obj.GetName(), "lifecycle", obj.Status.Phase)
	}()

	handler := r.dispatch(obj.Spec.Protocol)

	// The reconcile logic
	switch obj.Status.Phase {
	case v1alpha1.PhaseUninitialized:
		return handler.Create(ctx, &obj)

	case v1alpha1.PhasePending:
		return handler.Pending(ctx, &obj)

	case v1alpha1.PhaseRunning:
		return handler.Running(ctx, &obj)

	case v1alpha1.PhaseFailed:
		r.Logger.Info("Dataport failed", "name", obj.GetName())

		return common.Stop()

	case v1alpha1.PhaseChaos, v1alpha1.PhaseSuccess:
		// These phases should not happen in the workflow
		panic(errors.Errorf("invalid lifecycle phase %s", obj.Status.Phase))

	default:
		panic(errors.Errorf("unknown lifecycle phase: %s", obj.Status.Phase))
	}
}

func (r *Reconciler) Finalizer() string {
	return "dataports.frisbee.io/finalizer"
}

func (r *Reconciler) Finalize(obj client.Object) error {
	r.Logger.Info("Finalize", "kind", reflect.TypeOf(obj), "name", obj.GetName())

	return nil
}

type protocolHandler interface {
	// Create starts the object and procures external dependencies (e.g, create a queue)
	Create(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error)

	// Pending phase accepts connection offers from remote objects.
	Pending(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error)

	// Running means that the object is occupied and
	Running(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error)
}

func (r *Reconciler) dispatch(proto v1alpha1.PortProtocol) protocolHandler {
	switch proto {
	case v1alpha1.Direct:
		return &direct{r: r}

	case v1alpha1.Kafka:
		return &kafka{r: r}

	default:
		panic("should never happen")
	}
}
