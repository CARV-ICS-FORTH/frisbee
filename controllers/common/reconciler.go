package common

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func Stop() (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func Requeue() (ctrl.Result, error) {
	return ctrl.Result{Requeue: true}, nil
}

func StopWithError(err error) (ctrl.Result, error) {
	return ctrl.Result{}, errors.Wrapf(err, "requeue request")
}



// Reconciler implements basic functionality that is common to every solid reconciler (e.g, finalizers)
type Reconciler interface {
	client.Client
	logr.Logger
	Finalizer() string

	// Finalize deletes any external resources associated with the service
	// Examples finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.
	//
	// Ensure that delete implementation is idempotent and safe to invoke
	// multiple times for same object
	Finalize(object client.Object) error
}

// Reconcile provides the most Common functions for all the Reconcilers. That includes acquisition of the CR object
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
			// r.Info("object not found", "name", req.NamespacedName)
			return Stop()
		}

		r.Error(err, "error reading the object", "name", req.NamespacedName)
		// Error reading the object - requeue the request.
		return StopWithError(err)
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
		// then lets add the finalizer and Update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(obj, r.Finalizer()) {
			// cause the delete operation to block until all dependents objects are removed
			// see Foreground cascading deletion.
			// Note that in the "foregroundDeletion", only dependents with ownerReference.blockOwnerDeletion
			// block the deletion of the owner object.
			controllerutil.AddFinalizer(obj, metav1.FinalizerDeleteDependents)
			controllerutil.AddFinalizer(obj, r.Finalizer())

			if ret, err := Update(ctx, obj); err != nil {
				runtimeutil.HandleError(errors.Wrapf(err, "unable to add finalizer for %s", req.NamespacedName))

				return ret, err
			}

			// This code changes the spec and metadata, whereas the solid Reconciler changes the status.
			// If we do not return at this point, there will be a conflict because the solid Reconciler will try
			// to Update the status of a modified object.
			// To cause the solid Reconciler to return immediate, we use *ret=true
			return Stop()
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
				return StopWithError(err)
			}

			// Remove CR (Custom Resource) finalizer.
			// Once all finalizers have been removed, the object will be deleted.
			controllerutil.RemoveFinalizer(obj, r.Finalizer())
			controllerutil.RemoveFinalizer(obj, metav1.FinalizerDeleteDependents)

			if ret, err := Update(ctx, obj); err != nil {
				runtimeutil.HandleError(errors.Wrapf(err, "unable to remove finalizer for %s", req.NamespacedName))

				return ret, err
			}

			// r.Info("remove finalizers", "name", req.NamespacedName)
			*ret = true
		}
		// Stop reconciliation as the item is being deleted
		return Stop()
	}

	*ret = false

	return Stop()
}

func Update(ctx context.Context, obj client.Object) (ctrl.Result, error) {
	// we need to Update a delete object in order to remove the finalizers.
	updateError := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return Common.Client.Update(ctx, obj)
	})

	switch {
	case updateError == nil:
		return Stop()

	case k8errors.IsInvalid(updateError):
		Common.Logger.Error(updateError, "Update error")

		return Stop()

	case k8errors.IsNotFound(updateError):
		// The object has been deleted since we read it.
		// Requeue the object to try to reconciliate again.
		return Requeue()

	case k8errors.IsConflict(updateError):
		// The object has been updated since we read it.
		// Requeue the object to try to reconciliate again.
		runtimeutil.HandleError(errors.Wrapf(updateError, "update error"))

		return Requeue()

	default:
		runtimeutil.HandleError(errors.Wrapf(updateError, "Update failed for %s [%s]",
			obj.GetName(), obj.GetObjectKind().GroupVersionKind()))

		return Stop()
	}
}

func UpdateStatus(ctx context.Context, obj client.Object) (ctrl.Result, error) {
	// The status subresource ignores changes to spec, so it’s less likely to conflict with any other updates,
	// and can have separate permissions.
	updateError := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		Common.Logger.V(4).Info("(re) to update status of ", obj.GetName())

		return Common.Client.Status().Update(ctx, obj)
	})

	switch {
	case updateError == nil:
		return Stop()

	case k8errors.IsInvalid(updateError):
		Common.Logger.Error(updateError, "Update status error")

		return Stop()

	case k8errors.IsNotFound(updateError):
		// The object has been deleted since we read it.
		// Requeue the object to try to reconciliate again.
		return Requeue()

	case k8errors.IsConflict(updateError):
		// The object has been updated since we read it.
		// Requeue the object to try to reconciliate again.

		runtimeutil.HandleError(errors.Wrapf(updateError, "update status error"))

		return Requeue()

	default:
		runtimeutil.HandleError(errors.Wrapf(updateError, "status Update failed for %s [%s]",
			obj.GetName(), obj.GetObjectKind().GroupVersionKind()))

		return Stop()
	}
}
