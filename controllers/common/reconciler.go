package common

import (
	"context"

	"github.com/fnikolai/frisbee/pkg/structure"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

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

// Reconcile provides the most common functions for all the Reconcilers. That includes acquisition of the CR object
//  and management of the CR (Custom Resource) finalizers.
//
// Bool indicate whether the caller should return immediately (true) or continue (false)
func Reconcile(ctx context.Context, r Reconciler, req ctrl.Request, obj client.Object, ret *bool) (ctrl.Result, error) {
	*ret = true

	// Fetch the frisbee instance
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

	// Check if the object is marked to be deleted, which is indicated by the deletion timestamp being set.
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
				r.Error(err, "unable to remove finalizer", "name", req.NamespacedName)

				return RequeueWithError(err)
			}

			r.Info("remove finalizers", "name", req.NamespacedName)
		}
		// Stop reconciliation as the item is being deleted
		return DoNotRequeue()
	}

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
			r.Error(err, "unable to add finalizer", "name", req.NamespacedName)

			return RequeueWithError(err)
		}

		// This code changes the spec and metadata, whereas the solid Reconciler changes the status.
		// If we do not return at this point, there will be a conflict because the solid Reconciler will try
		// to update the status of a modified object.
		// To cause the solid Reconciler to return immediate, we use *ret=true
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
	object.SetLabels(structure.MergeMap(object.GetLabels(), map[string]string{
		"owner": owner.GetName(),
	}))

	return nil
}
