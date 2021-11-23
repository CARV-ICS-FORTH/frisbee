/*
Copyright 2021 ICS-FORTH.

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

package workflow

import (
	"fmt"
	"reflect"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/utils"
	"github.com/pkg/errors"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func (r *Controller) WatchChaos() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc:  r.watchChaosCreate,
		DeleteFunc:  r.watchChaosDelete,
		UpdateFunc:  r.watchChaosUpdate,
		GenericFunc: r.watchChaosGeneric,
	}
}

func (r *Controller) watchChaosCreate(e event.CreateEvent) bool {
	if !utils.IsManagedByThisController(e.Object, r.gvk) {
		return false
	}

	if !e.Object.GetDeletionTimestamp().IsZero() {
		// on a restart of the controller manager, it's possible a new pod shows up in a state that
		// is already pending deletion. Prevent the pod from being a creation observation.
		return false
	}

	r.Logger.Info("** Detected",
		"Request", "Create",
		"kind", reflect.TypeOf(e.Object),
		"name", e.Object.GetName(),
		"version", e.Object.GetResourceVersion(),
	)

	return true
}

func (r *Controller) watchChaosUpdate(e event.UpdateEvent) bool {
	if !utils.IsManagedByThisController(e.ObjectNew, r.gvk) {
		return false
	}

	if e.ObjectOld.GetResourceVersion() >= e.ObjectNew.GetResourceVersion() {
		// Periodic resync will send update events for all known pods.
		// Two different versions of the same pod will always have different RVs.
		return false
	}

	if !e.ObjectNew.GetDeletionTimestamp().IsZero() {
		// when an object is deleted gracefully it's deletion timestamp is first modified to reflect a grace period,
		// and after such time has passed, the kubelet actually deletes it from the store. We receive an update
		// for modification of the deletion timestamp and expect the reconciler to act asap, not to wait until the
		// kubelet actually deletes the object.
		return true
	}

	// if the status is the same, there is no need to inform the service
	prev := e.ObjectOld.(*v1alpha1.Chaos)
	latest := e.ObjectNew.(*v1alpha1.Chaos)

	if prev.Status.Phase == latest.Status.Phase {
		// a controller never initiates a phase change, and so is never asleep waiting for the same.
		return false
	}

	r.Logger.Info("** Detected",
		"Request", "Update",
		"kind", reflect.TypeOf(e.ObjectNew),
		"name", e.ObjectNew.GetName(),
		"from", prev.Status.Phase,
		"to", latest.Status.Phase,
		"version", fmt.Sprintf("%s -> %s", prev.GetResourceVersion(), latest.GetResourceVersion()),
	)

	return true
}

func (r *Controller) watchChaosDelete(e event.DeleteEvent) bool {
	if !utils.IsManagedByThisController(e.Object, r.gvk) {
		return false
	}

	// an object was deleted but the watch deletion event was missed while disconnected from apiserver.
	// In this case we don't know the final "resting" state of the object,
	// so there's a chance the included `Obj` is stale.
	if e.DeleteStateUnknown {
		runtimeutil.HandleError(errors.Errorf("couldn't get object from tombstone %+v", e.Object))

		return false
	}

	r.Logger.Info("** Detected",
		"Request", "Delete",
		"kind", reflect.TypeOf(e.Object),
		"name", e.Object.GetName(),
		"version", e.Object.GetResourceVersion(),
	)

	return true
}

func (r *Controller) watchChaosGeneric(event.GenericEvent) bool {
	return true
}
