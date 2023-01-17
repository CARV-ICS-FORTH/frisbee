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

package chaos

import (
	"fmt"
	"reflect"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/carv-ics-forth/frisbee/pkg/grafana"
	"github.com/pkg/errors"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func (r *Controller) Watchers() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc:  r.create,
		DeleteFunc:  r.delete,
		UpdateFunc:  r.update,
		GenericFunc: generic,
	}
}

func (r *Controller) create(event event.CreateEvent) bool {
	if !common.IsManagedByThisController(event.Object, r.gvk) {
		return false
	}

	if !event.Object.GetDeletionTimestamp().IsZero() {
		// on a restart of the controller manager, it's possible a new pod shows up in a state that
		// is already pending deletion. Prevent the pod from being a creation observation.
		return false
	}

	r.Logger.Info("** Enqueue",
		"Request", "Create",
		"kind", reflect.TypeOf(event.Object),
		"obj", client.ObjectKeyFromObject(event.Object),
		"version", event.Object.GetResourceVersion(),
	)

	if v1alpha1.DrawRegion(event.Object) {
		annotation := &grafana.RangeAnnotation{}
		annotation.Add(event.Object, grafana.TagChaos)

		r.regionAnnotations.Set(event.Object.GetName(), annotation)
	} else {
		annotation := &grafana.PointAnnotation{}
		annotation.Add(event.Object, grafana.TagChaos)
	}

	return true
}

func (r *Controller) update(event event.UpdateEvent) bool {
	if !common.IsManagedByThisController(event.ObjectNew, r.gvk) {
		return false
	}

	if event.ObjectOld.GetResourceVersion() == event.ObjectNew.GetResourceVersion() {
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

	r.Logger.Info("** Enqueue",
		"Request", "Update",
		"kind", reflect.TypeOf(event.ObjectNew),
		"obj", client.ObjectKeyFromObject(event.ObjectNew),
		"version", fmt.Sprintf("%s -> %s", event.ObjectOld.GetResourceVersion(), event.ObjectNew.GetResourceVersion()),
	)

	return true
}

func (r *Controller) delete(event event.DeleteEvent) bool {
	if !common.IsManagedByThisController(event.Object, r.gvk) {
		return false
	}

	// An object was deleted but the watch deletion event was missed while disconnected from apiserver.
	// In this case we don't know the final "resting" state of the object,
	// so there's a chance the included `Obj` is stale.
	if event.DeleteStateUnknown {
		runtimeutil.HandleError(errors.Errorf("couldn't get object from tombstone %+v", event.Object))

		return false
	}

	r.Logger.Info("** Enqueue",
		"Request", "Delete",
		"kind", reflect.TypeOf(event.Object),
		"obj", client.ObjectKeyFromObject(event.Object),
		"version", event.Object.GetResourceVersion(),
	)

	if v1alpha1.DrawRegion(event.Object) {
		annotation, ok := r.regionAnnotations.Get(event.Object.GetName())
		if !ok {
			// this is a stall condition that happens when the controller is restarted. just ignore it
			return false
		}

		annotation.(*grafana.RangeAnnotation).Delete(event.Object, grafana.TagChaos)

		r.regionAnnotations.Remove(event.Object.GetName())
	} else {
		annotation := &grafana.PointAnnotation{}
		annotation.Delete(event.Object, grafana.TagChaos)
	}

	return true
}

func generic(event.GenericEvent) bool {
	return true
}
