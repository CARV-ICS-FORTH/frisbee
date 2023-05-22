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

package common

import (
	"context"
	"reflect"
	"time"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func Stop(r Reconciler, req ctrl.Request) (ctrl.Result, error) {
	r.Info("** Dequeue", "obj", req)

	return ctrl.Result{Requeue: false, RequeueAfter: 0}, nil
}

// RequeueAfter will place the request in a queue, but it will be dequeue after the specified period.
func RequeueAfter(r Reconciler, req ctrl.Request, delay time.Duration) (ctrl.Result, error) {
	r.Info("** Requeue", "request", req, "delay", delay)

	return ctrl.Result{Requeue: true, RequeueAfter: delay}, nil
}

// RequeueWithError will place the request in a queue, and will be immediately dequeued.
// State dequeuing the request, the controller will report the error.
func RequeueWithError(r Reconciler, req ctrl.Request, err error) (ctrl.Result, error) {
	r.Info("** Requeue", "request", req, "error", err)

	return ctrl.Result{Requeue: true, RequeueAfter: 0}, err
}

type Logger interface {
	// Info logs a non-error message with the given key/value pairs as context.
	// The level argument is provided for optional logging.  This method will
	// only be called when Enabled(level) is true. See Logger.Info for more
	// details.
	Info(msg string, keysAndValues ...interface{})

	// Error logs an error, with the given message and key/value pairs as
	// context.  See Logger.Error for more details.
	Error(err error, msg string, keysAndValues ...interface{})

	// WithValues returns a new LogSink with additional key/value pairs.  See
	// Logger.WithValues for more details.
	WithValues(keysAndValues ...interface{}) logr.Logger

	// WithName returns a new LogSink with the specified name appended.  See
	// Logger.WithName for more details.
	WithName(name string) logr.Logger
}

// Reconciler implements basic functionality that is common to every solid reconciler (e.g, finalizers).
type Reconciler interface {
	GetClient() client.Client
	GetCache() cache.Cache

	GetEventRecorderFor(name string) record.EventRecorder

	Logger

	// Finalizer returns a list of finalizers associated with the controller.
	Finalizer() string

	// Finalize handles termination logic of the object, such as performing backups and deleting resources.
	Finalize(object client.Object) error
}

// Reconcile provides the most common functions for all the Reconcilers. That includes acquisition of the CR object
//
//	and management of the CR (Custom Resource) finalizers.
//
// Bool indicate whether the caller should return immediately (true) or continue (false).
// The reconciliation cycle is where the framework gives us back control after a watch has passed up an event.
func Reconcile(ctx context.Context, r Reconciler, req ctrl.Request, obj client.Object, requeue *bool) (ctrl.Result, error) {
	/*-- make the calling controller to return --*/
	*requeue = true

	/*---------------------------------------------------
	 * Retrieve CR by name
	 *---------------------------------------------------*/
	if err := r.GetClient().Get(ctx, req.NamespacedName, obj); err != nil {
		// 	Request object not found, could have been deleted after reconcile request.
		// 	We'll ignore not-found errors, since they can't be fixed by an immediate
		// 	requeue (we'll need to wait for a new notification), and we can get them
		// 	on added / deleted requests.
		if k8errors.IsNotFound(err) {
			return Stop(r, req)
		}

		return RequeueWithError(r, req, err)
	}

	logger := r.WithValues("obj", client.ObjectKeyFromObject(obj))

	/*---------------------------------------------------
	 * Set Finalizers for CR
	 *---------------------------------------------------*/
	if obj.GetDeletionTimestamp().IsZero() {
		/*---------------------------------------------------
		 * Add the Finalizer
		 *---------------------------------------------------*/
		if controllerutil.AddFinalizer(obj, r.Finalizer()) {
			logger.Info("AddFinalizer",
				"finalizer", r.Finalizer(),
				"current", obj.GetFinalizers(),
			)

			/*---------------------------------------------------
			 * Update the Object with Added Finalizers
			 *---------------------------------------------------*/
			retryCond := func() (done bool, err error) {
				if err := Update(context.Background(), r, obj); err != nil {
					// Retry
					logger.Info("Retry to add finalizer", "obj", obj, "Err", err)

					return false, nil
				}
				// OK
				return true, nil
			}

			ctxTimeout, cancel := context.WithTimeout(ctx, DefaultTimeoutFork8sEndpoint)
			defer cancel()

			if err := wait.ExponentialBackoffWithContext(ctxTimeout, DefaultBackoffForK8sEndpoint, retryCond); err != nil {
				logger.Error(err, "Abort retrying to add finalizer. Requeue the request")

				return RequeueWithError(r, req, err)
			}

			return Stop(r, req)
		}
	} else {
		if controllerutil.ContainsFinalizer(obj, r.Finalizer()) {
			/*---------------------------------------------------
			 * Run the Finalizer
			 *---------------------------------------------------*/
			if err := r.Finalize(obj); err != nil {
				logger.Error(err, "Finalize error", "finalizer", r.Finalizer())

				// retry to run the finalizer during the next reconciliation.
				return RequeueWithError(r, req, err)
			}

			/*---------------------------------------------------
			 * Remove the Finalizer
			 *---------------------------------------------------*/
			if controllerutil.RemoveFinalizer(obj, r.Finalizer()) {
				logger.Info("RemoveFinalizer",
					"finalizer", r.Finalizer(),
					"current", obj.GetFinalizers(),
				)

				/*---------------------------------------------------
				 * Update the Object with Removed Finalizers
				 *---------------------------------------------------*/
				retryCond := func() (done bool, err error) {
					if err := Update(context.Background(), r, obj); err != nil {
						logger.Info("Retry to remove finalizer", "obj", obj, "Err", err)

						// Retry
						return false, nil
					}

					// OK
					return true, nil
				}

				ctxTimeout, cancel := context.WithTimeout(ctx, DefaultTimeoutFork8sEndpoint)
				defer cancel()

				if err := wait.ExponentialBackoffWithContext(ctxTimeout, DefaultBackoffForK8sEndpoint, retryCond); err != nil {
					logger.Error(err, "Abort retrying to remove finalizer. Requeue the request")

					return RequeueWithError(r, req, err)
				}

				return Stop(r, req)
			}
		}
	}

	*requeue = false

	return Stop(r, req)
}

// Update will update the metadata and the spec of the Object. If there is a conflict, it will retry again.
func Update(ctx context.Context, reconciler Reconciler, obj client.Object) error {
	reconciler.Info("OO UpdtMeta",
		"obj", client.ObjectKeyFromObject(obj),
		"version", obj.GetResourceVersion(),
	)

	return reconciler.GetClient().Update(ctx, obj)
}

// UpdateStatus will update the status of the Object. If there is a conflict, it will retry again.
func UpdateStatus(ctx context.Context, reconciler Reconciler, obj client.Object) error {
	statusAwre, ok := obj.(v1alpha1.ReconcileStatusAware)
	if ok {
		reconciler.Info("OO UpdtStatus",
			"obj", client.ObjectKeyFromObject(obj),
			"phase", statusAwre.GetReconcileStatus().Phase,
			"version", obj.GetResourceVersion(),
		)

		return reconciler.GetClient().Status().Update(ctx, obj)
	}

	return errors.Errorf("object '%s' of GKV '%s' is not status aware",
		client.ObjectKeyFromObject(obj), obj.GetObjectKind().GroupVersionKind())
}

// Create ignores existing objects.
// if the next reconciliation cycle happens faster than the API update, it is possible to
// reschedule the creation of a Job. To avoid that, get if the Job is already submitted.
func Create(ctx context.Context, reconciler Reconciler, parent, child client.Object) error {
	if reconciler == nil || parent == nil || child == nil {
		panic(errors.Errorf("empty parameters.  Reconciler:%t Parent:%t Child:%t",
			reconciler == nil, parent == nil, child == nil))
	}

	// Create a searchable link between the parent and the children.
	v1alpha1.SetCreatedByLabel(child, parent)

	child.SetNamespace(parent.GetNamespace())

	// SetControllerReference sets owner as a Controller OwnerReference on controlled.
	// This is used for garbage collection of the controlled object and for
	// reconciling the owner object on changes to controlled (with a Logs + EnqueueRequestForOwner).
	// Since only one OwnerReference can be a controller, it returns an error if
	// there is another OwnerReference with Controller flag set.
	if err := controllerutil.SetControllerReference(parent, child, reconciler.GetClient().Scheme()); err != nil {
		return errors.Wrapf(err, "set controller reference")
	}

	reconciler.Info("++ Create",
		"kind", reflect.TypeOf(child),
		"obj", client.ObjectKeyFromObject(child),
	)

	if err := reconciler.GetClient().Create(ctx, child); err != nil {
		if k8errors.IsAlreadyExists(err) {
			// already exists. nothing to do.
			return nil
		}

		return errors.Wrapf(err, "creation error")
	}

	return nil
}

func ListChildren(ctx context.Context, cli client.Client, childJobs client.ObjectList, req types.NamespacedName) error {
	filters := []client.ListOption{
		client.InNamespace(req.Namespace),
		client.MatchingLabels{v1alpha1.LabelCreatedBy: req.Name},
	}

	if err := cli.List(ctx, childJobs, filters...); err != nil {
		return errors.Wrapf(err, "cannot list children")
	}

	return nil
}

// Delete removes a Kubernetes object, ignoring the NotFound error. If any error exists,
// it is recorded in the reconciler's logger.
func Delete(ctx context.Context, reconciler Reconciler, obj client.Object) {
	reconciler.Info("-- Delete",
		"kind", reflect.TypeOf(obj),
		"obj", client.ObjectKeyFromObject(obj),
		"version", obj.GetResourceVersion(),
	)

	// propagation := metav1.DeletePropagationForeground
	propagation := metav1.DeletePropagationBackground
	options := client.DeleteOptions{PropagationPolicy: &propagation}

	err := reconciler.GetClient().Delete(ctx, obj, &options)

	switch {
	case k8errors.IsNotFound(err):
	// Ignore
	case err != nil:
		reconciler.Error(err, "deletion error", "obj", client.ObjectKeyFromObject(obj))
	default:
		return
	}
}

// IsManagedByThisController returns true if the object is managed by the specified controller.
// If it is managed by another controller, or no controller is being resolved, it returns false.
func IsManagedByThisController(obj metav1.Object, controller schema.GroupVersionKind) bool {
	owner := metav1.GetControllerOf(obj)
	if owner == nil {
		return false
	}

	if owner.APIVersion != controller.GroupVersion().String() ||
		owner.Kind != controller.Kind {
		return false
	}

	return true
}
