package common

import (
	"context"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func DoNotRequeue() (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func RequeueWithError(err error) (ctrl.Result, error) {
	return ctrl.Result{}, err
}

// Reconcile provides the most common functions for all the Reconcilers. That includes acquisition of the CR object
//  and management of the CR (Custom Resource) finalizers.
//
// Bool indicate whether the caller should return immediately (true) or continue (false).
func Reconcile(ctx context.Context, r Reconciler, req ctrl.Request, obj client.Object, ret *bool) (ctrl.Result, error) {
	*ret = true

	//
	// 1. Retrieve the interested CR instance.
	//
	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		if k8errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			r.Info("object not found", "name", req.NamespacedName)

			// Return and don't requeue
			return DoNotRequeue()
		}

		r.Error(err, "error reading the object", "name", req.NamespacedName)
		// Error reading the object - requeue the request.
		return RequeueWithError(err)
	}

	//
	// 2. Manage the instance validity. We don’t want to try to do anything on an instance that does not
	// carry valid values.
	//

	//
	// 3. Manage instance initialization. If some values of the instance are not initialized,
	// this section will take care of it.
	//

	// Check if the object is marked to be deleted, which is indicated by the deletion timestamp being set.
	if obj.GetDeletionTimestamp().IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(obj, r.Finalizer()) {
			// cause the delete operation to block until all dependents objects are removed
			// see Foreground cascading deletion.
			// Note that in the "foregroundDeletion", only dependents with ownerReference.blockOwnerDeletion
			// block the deletion of the owner object.
			controllerutil.AddFinalizer(obj, metav1.FinalizerDeleteDependents)
			controllerutil.AddFinalizer(obj, r.Finalizer())

			if err := r.Update(ctx, obj); err != nil {
				runtimeutil.HandleError(errors.Wrapf(err, "unable to add finalizer for %s", req.NamespacedName))

				return RequeueWithError(err)
			}

			// This code changes the spec and metadata, whereas the solid Reconciler changes the status.
			// If we do not return at this point, there will be a conflict because the solid Reconciler will try
			// to update the status of a modified object.
			// To cause the solid Reconciler to return immediate, we use *ret=true
			return DoNotRequeue()
		}
	}

	//
	// 4. Manage instance deletion. If the instance is being deleted and we need to do some specific clean up,
	// this is where we manage it.
	//

	if !obj.GetDeletionTimestamp().IsZero() {
		if controllerutil.ContainsFinalizer(obj, r.Finalizer()) {
			// Run finalization logic to remove external dependencies. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.Finalize(obj); err != nil {
				return RequeueWithError(err)
			}

			// Remove CR (Custom Resource) finalizer.
			// Once all finalizers have been removed, the object will be deleted.
			controllerutil.RemoveFinalizer(obj, r.Finalizer())
			controllerutil.RemoveFinalizer(obj, metav1.FinalizerDeleteDependents)

			if err := r.Update(ctx, obj); err != nil {
				runtimeutil.HandleError(errors.Wrapf(err, "unable to remove finalizer for %s", req.NamespacedName))

				return DoNotRequeue()
			}

			r.Info("remove finalizers", "name", req.NamespacedName)
		}
		// Stop reconciliation as the item is being deleted
		return DoNotRequeue()
	}

	*ret = false

	return DoNotRequeue()
}

// SetOwner is a helper method to make sure the given object contains an object reference to the object provided.
func SetOwner(owner, object metav1.Object) error {
	if err := controllerutil.SetOwnerReference(owner, object, common.client.Scheme()); err != nil {
		return errors.Wrapf(err, "unable to set owner")
	}

	// add owner labels
	object.SetLabels(labels.Merge(object.GetLabels(), map[string]string{
		"owner": owner.GetName(),
	}))

	return nil
}

// Chaos is a wrapper that sets phase to Chaos and does not requeue the request.
func Chaos(ctx context.Context, obj InnerObject) (ctrl.Result, error) {
	status := obj.GetStatus()

	status.Phase = v1alpha1.Chaos
	status.Reason = "Expect controlled failures"

	obj.SetStatus(status)

	updateStatus(ctx, obj)

	return DoNotRequeue()
}

// Running is a wrapper that sets phase to Running and does not requeue the request.
func Running(ctx context.Context, obj InnerObject) (ctrl.Result, error) {
	status := obj.GetStatus()

	status.Phase = v1alpha1.Running
	status.Reason = "OK"

	obj.SetStatus(status)

	updateStatus(ctx, obj)

	return DoNotRequeue()
}

// Success is a wrapper that sets phase to Success and does not requeue the request.
func Success(ctx context.Context, obj InnerObject) (ctrl.Result, error) {
	status := obj.GetStatus()

	status.Phase = v1alpha1.Complete
	status.Reason = "All children are complete"

	obj.SetStatus(status)

	updateStatus(ctx, obj)

	return DoNotRequeue()
}

// Failed is a wrap that logs the error, updates the status, and does not requeue the request.
func Failed(ctx context.Context, obj InnerObject, err error) (ctrl.Result, error) {
	runtimeutil.HandleError(err)

	status := obj.GetStatus()

	status.Phase = v1alpha1.Failed
	status.Reason = err.Error()

	obj.SetStatus(status)

	updateStatus(ctx, obj)

	return DoNotRequeue()
}

func updateStatus(ctx context.Context, obj client.Object) {
	// if the object is scheduled for deletion, do not update its status
	if !obj.GetDeletionTimestamp().IsZero() {
		return
	}

	// The status subresource ignores changes to spec, so it’s less likely to conflict with any other updates,
	// and can have separate permissions.
	if err := common.client.Status().Update(ctx, obj); err != nil {
		runtimeutil.HandleError(errors.Wrapf(err, "unable to update status for %s [%s]",
			obj.GetName(), obj.GetObjectKind().GroupVersionKind()))
	}
}
