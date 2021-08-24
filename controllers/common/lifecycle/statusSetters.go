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

// WaitRunning is a lifecycle wrapper that waits until an object has reached the Running Phase.
// If another Phase is reached (e.g, failed), it returns error.
func WaitRunning(ctx context.Context, obj InnerObject) error {
	return New(
		Watch(obj, obj.GetName()),
		WithExpectedPhase(v1alpha1.PhaseRunning),
	).Run(ctx)
}

// WaitSuccess is a lifecycle wrapper that waits until an object has reached the Running Phase.
// If another Phase is reached (e.g, failed), it returns error.
func WaitSuccess(ctx context.Context, obj InnerObject) error {
	return New(
		Watch(obj, obj.GetName()),
		WithExpectedPhase(v1alpha1.PhaseSuccess),
	).Run(ctx)
}

// WaitRunningAndUpdate is a lifecycle that waits for WaitRunning and then replaced given object with the updated object.
func WaitRunningAndUpdate(ctx context.Context, obj InnerObject) error {
	err := New(
		Watch(obj, obj.GetName()),
		WithFilters(FilterByNames(obj.GetName())),
		WithExpectedPhase(v1alpha1.PhaseRunning),
	).Run(ctx)

	if err != nil {
		return errors.Wrapf(err, "Phase failed")
	}

	err = common.Common.Client.Get(ctx, client.ObjectKeyFromObject(obj), obj)

	return errors.Wrapf(err, "update failed")
}

/******************************************************
			Lifecycle Setters
/******************************************************/

// Pending is a wrapper that sets Phase to Pending and does not requeue the request.
func Pending(ctx context.Context, obj InnerObject, reason string) (ctrl.Result, error) {
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

	return common.UpdateStatus(ctx, obj)
}

// Running is a wrapper that sets Phase to Running and does not requeue the request.
func Running(ctx context.Context, obj InnerObject, reason string) (ctrl.Result, error) {
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

	return common.UpdateStatus(ctx, obj)
}

// Success is a wrapper that sets Phase to Success and does not requeue the request.
func Success(ctx context.Context, obj InnerObject, reason string) (ctrl.Result, error) {
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

	return common.UpdateStatus(ctx, obj)
}

// Chaos is a wrapper that sets Phase to Chaos and does not requeue the request.
func Chaos(ctx context.Context, obj InnerObject, reason string) (ctrl.Result, error) {
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

	return common.UpdateStatus(ctx, obj)
}

// Failed is a wrap that logs the error, updates the status, and does not requeue the request.
func Failed(ctx context.Context, obj InnerObject, err error) (ctrl.Result, error) {
	if ctx == nil || obj == nil || err == nil {
		panic("invalid args")
	}

	common.Common.Logger.Error(err, "object failed", "name", obj.GetName())

	obj.SetLifecycle(v1alpha1.Lifecycle{
		Kind:      reflect.TypeOf(obj).String(),
		Name:      obj.GetName(),
		Phase:     v1alpha1.PhaseFailed,
		Reason:    err.Error(),
		StartTime: &metav1.Time{Time: obj.GetCreationTimestamp().Time},
		EndTime:   obj.GetDeletionTimestamp(),
	})

	return common.UpdateStatus(ctx, obj)
}
