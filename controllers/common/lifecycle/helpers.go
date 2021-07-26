package lifecycle

import (
	"context"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/pkg/errors"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

/******************************************************
			Lifecycle Getters
/******************************************************/

// WaitReady is a lifecycle wrapper that waits until an object has reached the Running phase.
// If another phase is reached (e.g, failed), it returns error.
func WaitReady(ctx context.Context, obj InnerObject) error {
	return New(ctx,
		NewWatchdog(obj, obj.GetName()),
	).Expect(v1alpha1.PhaseRunning)
}

// WaitSuccess is a lifecycle wrapper that waits until an object has reached the Running phase.
// If another phase is reached (e.g, failed), it returns error.
func WaitSuccess(ctx context.Context, obj InnerObject) error {
	return New(ctx,
		NewWatchdog(obj, obj.GetName()),
	).Expect(v1alpha1.PhaseSuccess)
}

/******************************************************
			Lifecycle Setters
/******************************************************/

// Discoverable is a wrapper that sets phase to Discoverable and does not requeue the request.
func Discoverable(ctx context.Context, obj InnerObject, reason string) (ctrl.Result, error) {
	if ctx == nil || obj == nil || reason == "" {
		panic("invalid args")
	}

	status := obj.GetLifecycle()

	status.Phase = v1alpha1.PhaseDiscoverable
	status.Reason = reason

	obj.SetLifecycle(status)

	return common.UpdateStatus(ctx, obj)
}

// Pending is a wrapper that sets phase to Pending and does not requeue the request.
func Pending(ctx context.Context, obj InnerObject, reason string) (ctrl.Result, error) {
	if ctx == nil || obj == nil || reason == "" {
		panic("invalid args")
	}

	status := obj.GetLifecycle()

	status.Phase = v1alpha1.PhasePending
	status.Reason = reason

	obj.SetLifecycle(status)

	return common.UpdateStatus(ctx, obj)
}

// Running is a wrapper that sets phase to Running and does not requeue the request.
func Running(ctx context.Context, obj InnerObject, reason string) (ctrl.Result, error) {
	if ctx == nil || obj == nil || reason == "" {
		panic("invalid args")
	}

	status := obj.GetLifecycle()

	status.Phase = v1alpha1.PhaseRunning
	status.Reason = reason

	obj.SetLifecycle(status)

	return common.UpdateStatus(ctx, obj)
}

// Success is a wrapper that sets phase to Success and does not requeue the request.
func Success(ctx context.Context, obj InnerObject, reason string) (ctrl.Result, error) {
	if ctx == nil || obj == nil || reason == "" {
		panic("invalid args")
	}

	status := obj.GetLifecycle()

	status.Phase = v1alpha1.PhaseSuccess
	status.Reason = reason

	obj.SetLifecycle(status)

	return common.UpdateStatus(ctx, obj)
}

// Chaos is a wrapper that sets phase to Chaos and does not requeue the request.
func Chaos(ctx context.Context, obj InnerObject, reason string) (ctrl.Result, error) {
	if ctx == nil || obj == nil || reason == "" {
		panic("invalid args")
	}

	status := obj.GetLifecycle()

	status.Phase = v1alpha1.PhaseChaos
	status.Reason = reason

	obj.SetLifecycle(status)

	return common.UpdateStatus(ctx, obj)
}

// Failed is a wrap that logs the error, updates the status, and does not requeue the request.
func Failed(ctx context.Context, obj InnerObject, err error) (ctrl.Result, error) {
	if ctx == nil || obj == nil || err == nil {
		panic("invalid args")
	}

	runtimeutil.HandleError(errors.Wrapf(err, "object %s has failed", obj.GetName()))

	status := obj.GetLifecycle()
	status.Phase = v1alpha1.PhaseFailed
	status.Reason = err.Error()

	obj.SetLifecycle(status)

	return common.UpdateStatus(ctx, obj)
}

/******************************************************
	Wrappers and Unwrappers for InnerObjects
/******************************************************/

// externalToInnerObject is a wrapper for converting external objects (e.g, Pods) to InnerObjects managed
// by the Frisbee controller
type externalToInnerObject struct {
	client.Object

	LifecycleFunc func(obj interface{}) v1alpha1.Lifecycle
}

func (d *externalToInnerObject) GetLifecycle() v1alpha1.Lifecycle {
	return d.LifecycleFunc(d.Object)
}

func (d *externalToInnerObject) SetLifecycle(v1alpha1.Lifecycle) {
	panic(errors.Errorf("cannot set status on external object"))
}

func unwrap(obj client.Object) client.Object {
	wrapped, ok := obj.(*externalToInnerObject)
	if ok {
		return wrapped.Object
	}

	return obj
}

func accessStatus(obj interface{}) func(interface{}) v1alpha1.Lifecycle {
	external, ok := obj.(*externalToInnerObject)
	if ok {
		return external.LifecycleFunc
	}

	return func(inner interface{}) v1alpha1.Lifecycle {
		return inner.(InnerObject).GetLifecycle()
	}
}
