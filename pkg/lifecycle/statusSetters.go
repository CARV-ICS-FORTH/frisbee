/*
Copyright 2021-2023 ICS-FORTH.

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
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

/******************************************************
			Lifecycle Setters
/******************************************************/

// Success is a wrapper that sets Phase to Success and does not requeue the request.
func Success(ctx context.Context, reconciler common.Reconciler, obj client.Object, msg string) (reconcile.Result, error) {
	if ctx == nil || obj == nil || msg == "" {
		panic("invalid args")
	}

	status := v1alpha1.Lifecycle{
		Phase:   v1alpha1.PhaseSuccess,
		Reason:  v1alpha1.PhaseSuccess.String(),
		Message: msg,
	}

	if statusAware, updateStatus := obj.(v1alpha1.ReconcileStatusAware); updateStatus {
		statusAware.SetReconcileStatus(status)

		reconciler.Info("SetLifecycle",
			"obj", client.ObjectKeyFromObject(obj),
			"phase", status.Phase)

		if err := common.UpdateStatus(ctx, reconciler, obj); err != nil {
			return ctrl.Result{RequeueAfter: time.Second, Requeue: true}, nil
		}
	} else {
		reconciler.Info("Object does not support RecocileStatusAware interface. Not setting status",
			"obj", obj.GetName(),
			"status", status,
		)
	}

	reconciler.GetEventRecorderFor(obj.GetName()).Event(obj, corev1.EventTypeNormal, status.Reason, status.Message)

	return ctrl.Result{}, nil
}

// Pending is a wrapper that sets Phase to Pending and does not requeue the request.
func Pending(ctx context.Context, reconciler common.Reconciler, obj client.Object, msg string) (reconcile.Result, error) {
	if ctx == nil || obj == nil || msg == "" {
		panic("invalid args")
	}

	status := v1alpha1.Lifecycle{
		Phase:   v1alpha1.PhasePending,
		Reason:  v1alpha1.PhasePending.String(),
		Message: msg,
	}

	if statusAware, updateStatus := obj.(v1alpha1.ReconcileStatusAware); updateStatus {
		statusAware.SetReconcileStatus(status)

		reconciler.Info("SetLifecycle",
			"obj", client.ObjectKeyFromObject(obj),
			"phase", status.Phase)

		if err := common.UpdateStatus(ctx, reconciler, obj); err != nil {
			return ctrl.Result{RequeueAfter: time.Second, Requeue: true}, nil
		}
	} else {
		reconciler.Info("Object does not support RecocileStatusAware interface. Not setting status",
			"obj", obj.GetName(),
			"status", status,
		)
	}

	reconciler.GetEventRecorderFor(obj.GetName()).Event(obj, corev1.EventTypeNormal, status.Reason, status.Message)

	return ctrl.Result{}, nil
}

// Failed is a wrap that logs the error, updates the status, and does not requeue the request.
func Failed(ctx context.Context, reconciler common.Reconciler, obj client.Object, issue error) (reconcile.Result, error) {
	if ctx == nil || obj == nil || issue == nil {
		panic("invalid args")
	}

	status := v1alpha1.Lifecycle{
		Phase:   v1alpha1.PhaseFailed,
		Reason:  v1alpha1.PhaseFailed.String(),
		Message: issue.Error(),
	}

	if statusAware, updateStatus := obj.(v1alpha1.ReconcileStatusAware); updateStatus {
		statusAware.SetReconcileStatus(status)

		reconciler.Info("SetLifecycle",
			"obj", client.ObjectKeyFromObject(obj),
			"phase", status.Phase)

		if err := common.UpdateStatus(ctx, reconciler, obj); err != nil {
			return ctrl.Result{RequeueAfter: time.Second, Requeue: true}, nil
		}
	} else {
		reconciler.Info("Object does not support RecocileStatusAware interface. Not setting status",
			"obj", obj.GetName(), "status", status,
		)
	}

	if !obj.GetDeletionTimestamp().IsZero() {
		// If the object is deleted, then probably the namespace is deleted as well and pushing
		// a notification will raise warnings.
		reconciler.GetEventRecorderFor(obj.GetName()).Event(obj, corev1.EventTypeWarning, status.Reason, status.Message)
	}

	return ctrl.Result{}, nil
}
