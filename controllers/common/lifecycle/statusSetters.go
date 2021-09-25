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

package lifecycle

import (
	"context"
	"reflect"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

/******************************************************
			Lifecycle Getters
/******************************************************/

// WaitRunningAndUpdate is a lifecycle that waits for WaitRunning and then replaced given object with the updated object.
func WaitRunningAndUpdate(ctx context.Context, r common.Reconciler, obj InnerObject) error {
	err := New(
		Watch(obj, obj.GetName()),
		WithFilters(FilterByNames(obj.GetName())),
		WithExpectedPhase(v1alpha1.PhaseRunning),
	).Run(ctx, r)

	if err != nil {
		return errors.Wrapf(err, "Phase failed")
	}

	err = common.Globals.Client.Get(ctx, client.ObjectKeyFromObject(obj), obj)

	return errors.Wrapf(err, "update failed")
}

/******************************************************
			Lifecycle Setters
/******************************************************/

// Pending is a wrapper that sets Phase to Pending and does not requeue the request.
func Pending(ctx context.Context, r common.Reconciler, obj InnerObject, reason string) (ctrl.Result, error) {
	if ctx == nil || obj == nil || reason == "" {
		panic("invalid args")
	}

	obj.SetLifecycle(v1alpha1.Lifecycle{
		Kind:      reflect.TypeOf(obj).String(),
		Name:      obj.GetName(),
		Phase:     v1alpha1.PhasePending,
		Reason:    reason,
		StartTime: &metav1.Time{Time: obj.GetCreationTimestamp().Time},
		EndTime:   nil,
	})

	return common.UpdateStatus(ctx, r, obj)
}

// Running is a wrapper that sets Phase to Running and does not requeue the request.
func Running(ctx context.Context, r common.Reconciler, obj InnerObject, reason string) (ctrl.Result, error) {
	if ctx == nil || obj == nil || reason == "" {
		panic("invalid args")
	}

	obj.SetLifecycle(v1alpha1.Lifecycle{
		Kind:      reflect.TypeOf(obj).String(),
		Name:      obj.GetName(),
		Phase:     v1alpha1.PhaseRunning,
		Reason:    reason,
		StartTime: &metav1.Time{Time: obj.GetCreationTimestamp().Time},
		EndTime:   nil,
	})

	return common.UpdateStatus(ctx, r, obj)
}

// Success is a wrapper that sets Phase to Success and does not requeue the request.
func Success(ctx context.Context, r common.Reconciler, obj InnerObject, reason string) (ctrl.Result, error) {
	if ctx == nil || obj == nil || reason == "" {
		panic("invalid args")
	}

	obj.SetLifecycle(v1alpha1.Lifecycle{
		Kind:      reflect.TypeOf(obj).String(),
		Name:      obj.GetName(),
		Phase:     v1alpha1.PhaseSuccess,
		Reason:    reason,
		StartTime: &metav1.Time{Time: obj.GetCreationTimestamp().Time},
		EndTime:   obj.GetDeletionTimestamp(),
	})

	return common.UpdateStatus(ctx, r, obj)
}

// Chaos is a wrapper that sets Phase to Chaos and does not requeue the request.
func Chaos(ctx context.Context, r common.Reconciler, obj InnerObject, reason string) (ctrl.Result, error) {
	if ctx == nil || obj == nil || reason == "" {
		panic("invalid args")
	}

	obj.SetLifecycle(v1alpha1.Lifecycle{
		Kind:      reflect.TypeOf(obj).String(),
		Name:      obj.GetName(),
		Phase:     v1alpha1.PhaseChaos,
		Reason:    reason,
		StartTime: &metav1.Time{Time: obj.GetCreationTimestamp().Time},
		EndTime:   nil,
	})

	return common.UpdateStatus(ctx, r, obj)
}

// Failed is a wrap that logs the error, updates the status, and does not requeue the request.
func Failed(ctx context.Context, r common.Reconciler, obj InnerObject, err error) (ctrl.Result, error) {
	if ctx == nil || obj == nil || err == nil {
		panic("invalid args")
	}

	r.Error(err, "object failed", "name", obj.GetName())

	obj.SetLifecycle(v1alpha1.Lifecycle{
		Kind:      reflect.TypeOf(obj).String(),
		Name:      obj.GetName(),
		Phase:     v1alpha1.PhaseFailed,
		Reason:    err.Error(),
		StartTime: &metav1.Time{Time: obj.GetCreationTimestamp().Time},
		EndTime:   obj.GetDeletionTimestamp(),
	})

	return common.UpdateStatus(ctx, r, obj)
}
