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

package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ReconcileStatusAware interface {
	GetReconcileStatus() v1alpha1.Lifecycle
	SetReconcileStatus(v1alpha1.Lifecycle)
}

/******************************************************
			Lifecycle Setters
/******************************************************/

// Pending is a wrapper that sets Phase to Pending and does not requeue the request.
func Pending(ctx context.Context, r Reconciler, obj client.Object, reason string) (reconcile.Result, error) {
	if ctx == nil || obj == nil || reason == "" {
		panic("invalid args")
	}

	status := v1alpha1.Lifecycle{
		Phase:   v1alpha1.PhasePending,
		Reason:  fmt.Sprintf("%s%s", obj.GetName(), "Running"),
		Message: reason,
	}

	if statusAware, updateStatus := obj.(ReconcileStatusAware); updateStatus {
		statusAware.SetReconcileStatus(status)

		if err := UpdateStatus(ctx, r, obj); err != nil {
			r.Error(err, "unable to update status")

			return RequeueAfter(time.Second)
		}
	} else {
		r.Info("Object does not support RecocileStatusAware interface. Not setting status",
			"obj", obj.GetName(), "status", status,
		)
	}

	return Stop()
}

// Running is a wrapper that sets Phase to Running and does not requeue the request.
func Running(ctx context.Context, r Reconciler, obj client.Object, reason string) (reconcile.Result, error) {
	if ctx == nil || obj == nil || reason == "" {
		panic("invalid args")
	}

	status := v1alpha1.Lifecycle{
		Phase:   v1alpha1.PhaseRunning,
		Reason:  fmt.Sprintf("%s%s", obj.GetName(), "Running"),
		Message: reason,
	}

	if statusAware, updateStatus := obj.(ReconcileStatusAware); updateStatus {
		statusAware.SetReconcileStatus(status)

		if err := UpdateStatus(ctx, r, obj); err != nil {
			r.Error(err, "unable to update status")

			return RequeueAfter(time.Second)
		}
	} else {
		r.Info("Object does not support RecocileStatusAware interface. Not setting status",
			"obj", obj.GetName(), "status", status,
		)
	}

	return Stop()
}

// Success is a wrapper that sets Phase to Success and does not requeue the request.
func Success(ctx context.Context, r Reconciler, obj client.Object, reason string) (reconcile.Result, error) {
	if ctx == nil || obj == nil || reason == "" {
		panic("invalid args")
	}

	status := v1alpha1.Lifecycle{
		Phase:   v1alpha1.PhaseSuccess,
		Reason:  fmt.Sprintf("%s%s", obj.GetName(), "Success"),
		Message: reason,
	}

	if statusAware, updateStatus := obj.(ReconcileStatusAware); updateStatus {
		statusAware.SetReconcileStatus(status)

		if err := UpdateStatus(ctx, r, obj); err != nil {
			r.Error(err, "unable to update status")

			return RequeueAfter(time.Second)
		}
	} else {
		r.Info("Object does not support RecocileStatusAware interface. Not setting status",
			"obj", obj.GetName(),
			"status", status,
		)
	}

	return Stop()
}

// Failed is a wrap that logs the error, updates the status, and does not requeue the request.
func Failed(ctx context.Context, r Reconciler, obj client.Object, issue error) (reconcile.Result, error) {
	if ctx == nil || obj == nil || issue == nil {
		panic("invalid args")
	}

	r.Error(issue, "object failed", "name", obj.GetName())

	// r.GetRecorder().Event(obj, "Warning", "ProcessingError", issue.Error())

	status := v1alpha1.Lifecycle{
		Phase:  v1alpha1.PhaseFailed,
		Reason: fmt.Sprintf("%s%s", obj.GetName(), "Failed"),
		Message: errors.Wrapf(issue, "unable to update status for %s (rv = %s): %v",
			obj.GetNamespace(), obj.GetName(), obj.GetResourceVersion()).Error(),
	}

	if statusAware, updateStatus := obj.(ReconcileStatusAware); updateStatus {
		statusAware.SetReconcileStatus(status)

		if err := UpdateStatus(ctx, r, obj); err != nil {
			r.Error(err, "unable to update status")

			return RequeueAfter(time.Second)
		}
	} else {
		r.Info("Object does not support RecocileStatusAware interface. Not setting status",
			"obj", obj.GetName(), "status", status,
		)
	}

	return Stop()
}
