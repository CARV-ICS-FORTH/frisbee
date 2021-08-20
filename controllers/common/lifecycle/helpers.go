package lifecycle

import (
	"context"
	"reflect"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
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

	runtimeutil.HandleError(errors.Wrapf(err, "object %s has failed", obj.GetName()))

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

/******************************************************
			Delete Managed objects
/******************************************************/

// Delete is a wrapper that addresses a circular dependency issue with the lifecycle monitoring.
// By default, Kubernetes deletes Children before the parent. When a Child is removed,
// the lifecycle watchdog detects that a child is deleted (failed) and updates the parent. However,
// the parent used in the lifecycle is a stalled copy of the actual parent object. Hence, the update
// causes a conflict between the stalled and the actual object.
//
// This deletion method addresses this issue by first deleting the parent, and then the children.
func Delete(ctx context.Context, c client.Client, obj client.Object) error {
	// There are three different options for the deletion propagation policy:
	//
	//    Foreground: Children are deleted before the parent (post-order)
	//    Background: Parent is deleted before the children (pre-order)
	//    Orphan: Owner references are ignored
	deletePolicy := metav1.DeletePropagationBackground

	if err := c.Delete(ctx, obj, &client.DeleteOptions{PropagationPolicy: &deletePolicy}); err != nil {
		return errors.Wrapf(err, "unable to delete object %s", obj.GetName())
	}

	return nil
}

/******************************************************
	Wrappers and Unwrappers for InnerObjects
/******************************************************/

// externalToInnerObject is a wrapper for converting external objects (e.g, Pods) to InnerObjects managed
// by the Frisbee controller
type externalToInnerObject struct {
	client.Object

	LifecycleFunc func(obj interface{}) []*v1alpha1.Lifecycle
}

func (d *externalToInnerObject) GetLifecycle() []*v1alpha1.Lifecycle {
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

func accessStatus(obj interface{}) GetLifecycleFunc {
	external, ok := obj.(*externalToInnerObject)
	if ok {
		return external.LifecycleFunc
	}

	return func(inner interface{}) []*v1alpha1.Lifecycle {
		return inner.(InnerObject).GetLifecycle()
	}
}
