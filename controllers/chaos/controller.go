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

package chaos

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

// +kubebuilder:rbac:groups=frisbee.io,resources=chaoss,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.io,resources=chaoss/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.io,resources=chaoss/finalizers,verbs=update

// Controller reconciles a Reference object
type Controller struct {
	ctrl.Manager
	logr.Logger
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		### 1: Load the chaos by name.
	*/

	var chaos v1alpha1.Chaos

	var ret bool
	result, err := common.Reconcile(ctx, r, req, &chaos, &ret)
	if ret {
		return result, err
	}

	r.Logger.Info("-> Reconcile",
		"kind", reflect.TypeOf(chaos),
		"name", chaos.GetName(),
		"lifecycle", chaos.Status.Phase,
		"deleted", !chaos.GetDeletionTimestamp().IsZero(),
	)

	defer func() {
		r.Logger.Info("<- Reconcile",
			"kind", reflect.TypeOf(chaos),
			"name", chaos.GetName(),
			"lifecycle", chaos.Status.Phase,
			"deleted", !chaos.GetDeletionTimestamp().IsZero(),
		)
	}()

	handler := r.dispatch(chaos.Spec.Type)

	// Here goes the actual reconcile logic
	switch chaos.Status.Phase {
	case v1alpha1.PhaseUninitialized:
		return lifecycle.Pending(ctx, r, &chaos, "received chaos request")

	case v1alpha1.PhasePending:
		if err := handler.Inject(ctx, &chaos); err != nil {
			return lifecycle.Failed(ctx, r, &chaos, errors.Wrapf(err, "injection failed"))
		}

		return common.Stop()

	case v1alpha1.PhaseRunning:
		if err := handler.WaitForDuration(ctx, &chaos); err != nil {
			return lifecycle.Failed(ctx, r, &chaos, errors.Wrapf(err, "chaos failed"))
		}

		return lifecycle.Success(ctx, r, &chaos, "chaos revoked")

	case v1alpha1.PhaseSuccess:
		r.Logger.Info("Chaos completed", "name", chaos.GetName())

		if err := handler.Revoke(ctx, &chaos); err != nil {
			return lifecycle.Failed(ctx, r, &chaos, errors.Wrapf(err, "unable to revoke chaos"))
		}

		return common.Stop()

	case v1alpha1.PhaseFailed:
		r.Logger.Info("Chaos failed", "name", chaos.GetName())

		return common.Stop()

	case v1alpha1.PhaseChaos:
		// These phases should not happen in the workflow
		panic(errors.Errorf("invalid lifecycle phase %s", chaos.Status.Phase))

	default:
		panic(errors.Errorf("unknown lifecycle phase: %s", chaos.Status.Phase))
	}
}

func (r *Controller) Finalizer() string {
	return "chaoss.frisbee.io/finalizer"
}

func (r *Controller) Finalize(obj client.Object) error {
	r.Logger.Info("Finalize", "kind", reflect.TypeOf(obj), "name", obj.GetName())

	return nil
}

type chaoHandler interface {
	Inject(ctx context.Context, obj *v1alpha1.Chaos) error
	WaitForDuration(ctx context.Context, obj *v1alpha1.Chaos) error
	Revoke(ctx context.Context, obj *v1alpha1.Chaos) error
}

func (r *Controller) dispatch(faultType v1alpha1.FaultType) chaoHandler {
	switch faultType {
	case v1alpha1.FaultPartition:
		return &partition{r: r}

	default:
		panic("should never happen")
	}
}

func NewController(mgr ctrl.Manager, logger logr.Logger) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Chaos{}).
		Named("chaos").
		Complete(&Controller{
			Manager: mgr,
			Logger:  logger.WithName("chaos"),
		})
}
