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

package watchers

import (
	"fmt"
	"reflect"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func Watch(r common.Reconciler, gvk schema.GroupVersionKind) builder.Predicates {
	w := simpleWatch{}

	return w.Watch(r, gvk)
}

type simpleWatch struct{}

func (w *simpleWatch) Watch(reconciler common.Reconciler, gvk schema.GroupVersionKind) builder.Predicates {
	return builder.WithPredicates(predicate.Funcs{
		CreateFunc:  w.watchCreate(reconciler, gvk),
		DeleteFunc:  w.watchDelete(reconciler, gvk),
		UpdateFunc:  w.watchUpdate(reconciler, gvk),
		GenericFunc: w.watchGeneric(),
	})
}

func (w *simpleWatch) watchCreate(reconciler common.Reconciler, gvk schema.GroupVersionKind) CreateFunc {
	return func(event event.CreateEvent) bool {
		/*---------------------------------------------------*
		 * Filter out irrelevant of obsolete events
		 *---------------------------------------------------*/
		if !common.IsManagedByThisController(event.Object, gvk) {
			return false
		}

		if !event.Object.GetDeletionTimestamp().IsZero() {
			// on a restart of the controller manager, it's possible a new pod shows up in a state that
			// is already pending deletion. Prevent the pod from being a creation observation.
			return false
		}

		/*---------------------------------------------------*
		 * Print information of enqueued requests
		 *---------------------------------------------------*/
		reconciler.Info("** Enqueue",
			"Request", "Create",
			"kind", reflect.TypeOf(event.Object),
			"obj", client.ObjectKeyFromObject(event.Object),
			"version", event.Object.GetResourceVersion(),
		)

		// we know the creation order, so we do not need to reconcile created objects.
		return false
	}
}

func (w *simpleWatch) watchUpdate(reconciler common.Reconciler, gvk schema.GroupVersionKind) UpdateFunc {
	return func(event event.UpdateEvent) bool {
		/*---------------------------------------------------*
		 * Filter out irrelevant of obsolete events
		 *---------------------------------------------------*/
		if !common.IsManagedByThisController(event.ObjectNew, gvk) {
			return false
		}

		if event.ObjectOld.GetResourceVersion() >= event.ObjectNew.GetResourceVersion() {
			// Periodic resync will send update events for all known pods.
			// Two different versions of the same pod will always have different RVs.
			return false
		}

		if !event.ObjectNew.GetDeletionTimestamp().IsZero() {
			// when an object is deleted gracefully it's deletion timestamp is first modified to reflect a grace period,
			// and after such time has passed, the kubelet actually deletes it from the store. We receive an update
			// for modification of the deletion timestamp and expect the reconciler to act asap, not to wait until the
			// kubelet actually deletes the object.
			return false
		}

		/*---------------------------------------------------*
		 * Try to extract information about Phase changes
		 *---------------------------------------------------*/
		prev, prevOK := event.ObjectOld.(v1alpha1.ReconcileStatusAware)
		latest, latestOK := event.ObjectNew.(v1alpha1.ReconcileStatusAware)

		if !prevOK || !latestOK {
			// this may happen for external objects like Pods, Faults, etc.
			reconciler.Info("** Enqueue (External)",
				"Request", "Update",
				"kind", reflect.TypeOf(event.ObjectNew),
				"obj", client.ObjectKeyFromObject(event.ObjectNew),
				"version", fmt.Sprintf("%s -> %s", event.ObjectOld.GetResourceVersion(), event.ObjectNew.GetResourceVersion()),
			)

			return true
		}

		prevPhase := prev.GetReconcileStatus().Phase
		latestPhase := latest.GetReconcileStatus().Phase

		// a controller never initiates a phase change, and so is never asleep waiting for the same.
		if prevPhase == latestPhase {
			reconciler.Info("Ignore Update", "obj", client.ObjectKeyFromObject(event.ObjectNew))

			return false
		}

		/*---------------------------------------------------*
		 * Print information of enqueued requests
		 *---------------------------------------------------*/
		reconciler.Info("** Enqueue",
			"Request", "Update",
			"kind", reflect.TypeOf(event.ObjectNew),
			"obj", client.ObjectKeyFromObject(event.ObjectNew),
			"phase", fmt.Sprintf("%s -> %s", prevPhase, latestPhase),
			"version", fmt.Sprintf("%s -> %s", event.ObjectOld.GetResourceVersion(), event.ObjectNew.GetResourceVersion()),
		)

		return true
	}
}

func (w *simpleWatch) watchDelete(reconciler common.Reconciler, gvk schema.GroupVersionKind) DeleteFunc {
	return func(event event.DeleteEvent) bool {
		/*---------------------------------------------------*
		 * Filter out irrelevant of obsolete events
		 *---------------------------------------------------*/
		if !common.IsManagedByThisController(event.Object, gvk) {
			return false
		}

		// an object was deleted but the watch deletion event was missed while disconnected from apiserver.
		// In this case we don't know the final "resting" state of the object,
		// so there's a chance the included `Obj` is stale.
		if event.DeleteStateUnknown {
			runtimeutil.HandleError(errors.Errorf("couldn'w get object from tombstone %+v", event.Object))

			return false
		}

		/*---------------------------------------------------*
		 * Print information of enqueued requests
		 *---------------------------------------------------*/
		reconciler.Info("** Enqueue",
			"Request", "Delete",
			"kind", reflect.TypeOf(event.Object),
			"obj", client.ObjectKeyFromObject(event.Object),
			"version", event.Object.GetResourceVersion(),
		)

		return true
	}
}

func (w *simpleWatch) watchGeneric() GenericFunc {
	return func(e event.GenericEvent) bool {
		return true
	}
}
