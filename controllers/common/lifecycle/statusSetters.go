/*
Copyright 2021 ICS-FORTH.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package lifecycle

import (
	"context"
	"time"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

/******************************************************
			Lifecycle Setters
/******************************************************/

// Pending is a wrapper that sets Phase to Pending and does not requeue the request.
func Pending(ctx context.Context, r common.Reconciler, obj client.Object, msg string) (reconcile.Result, error) {
	if ctx == nil || obj == nil || msg == "" {
		panic("invalid args")
	}

	status := v1alpha1.Lifecycle{
		Phase:   v1alpha1.PhasePending,
		Reason:  "SetToPending",
		Message: msg,
	}

	if statusAware, updateStatus := obj.(v1alpha1.ReconcileStatusAware); updateStatus {
		statusAware.SetReconcileStatus(status)

		if err := common.UpdateStatus(ctx, r, obj); err != nil {
			r.Info("Retry set lifecycle", "object", obj.GetName(), "phase", status.Phase)

			return common.RequeueAfter(time.Second)
		}
	} else {
		r.Info("Object does not support RecocileStatusAware interface. Not setting status",
			"obj", obj.GetName(), "status", status,
		)
	}

	return common.Stop()
}

// Running is a wrapper that sets Phase to Running and does not requeue the request.
func Running(ctx context.Context, r common.Reconciler, obj client.Object, msg string) (reconcile.Result, error) {
	if ctx == nil || obj == nil || msg == "" {
		panic("invalid args")
	}

	status := v1alpha1.Lifecycle{
		Phase:   v1alpha1.PhaseRunning,
		Reason:  "SetToRunning",
		Message: msg,
	}

	if statusAware, updateStatus := obj.(v1alpha1.ReconcileStatusAware); updateStatus {
		statusAware.SetReconcileStatus(status)

		if err := common.UpdateStatus(ctx, r, obj); err != nil {
			r.Info("Retry set lifecycle", "object", obj.GetName(), "phase", status.Phase)

			return common.RequeueAfter(time.Second)
		}
	} else {
		r.Info("Object does not support RecocileStatusAware interface. Not setting status",
			"obj", obj.GetName(), "status", status,
		)
	}

	return common.Stop()
}

// Success is a wrapper that sets Phase to Success and does not requeue the request.
func Success(ctx context.Context, r common.Reconciler, obj client.Object, msg string) (reconcile.Result, error) {
	if ctx == nil || obj == nil || msg == "" {
		panic("invalid args")
	}

	status := v1alpha1.Lifecycle{
		Phase:   v1alpha1.PhaseSuccess,
		Reason:  "SetToSuccess",
		Message: msg,
	}

	if statusAware, updateStatus := obj.(v1alpha1.ReconcileStatusAware); updateStatus {
		statusAware.SetReconcileStatus(status)

		if err := common.UpdateStatus(ctx, r, obj); err != nil {
			r.Info("Retry set lifecycle", "object", obj.GetName(), "phase", status.Phase)

			return common.RequeueAfter(time.Second)
		}
	} else {
		r.Info("Object does not support RecocileStatusAware interface. Not setting status",
			"obj", obj.GetName(),
			"status", status,
		)
	}

	return common.Stop()
}

// Failed is a wrap that logs the error, updates the status, and does not requeue the request.
func Failed(ctx context.Context, r common.Reconciler, obj client.Object, issue error) (reconcile.Result, error) {
	if ctx == nil || obj == nil || issue == nil {
		panic("invalid args")
	}

	r.Error(issue, "resource failed", "obj", client.ObjectKeyFromObject(obj))

	status := v1alpha1.Lifecycle{
		Phase:   v1alpha1.PhaseFailed,
		Reason:  "SetToFailed",
		Message: errors.Wrapf(issue, "execution error (rv = %s)", obj.GetResourceVersion()).Error(),
	}

	if statusAware, updateStatus := obj.(v1alpha1.ReconcileStatusAware); updateStatus {
		statusAware.SetReconcileStatus(status)

		if err := common.UpdateStatus(ctx, r, obj); err != nil {
			r.Info("Retry set lifecycle", "object", obj.GetName(), "phase", status.Phase)

			return common.RequeueAfter(time.Second)
		}
	} else {
		r.Info("Object does not support RecocileStatusAware interface. Not setting status",
			"obj", obj.GetName(), "status", status,
		)
	}

	return common.Stop()
}
