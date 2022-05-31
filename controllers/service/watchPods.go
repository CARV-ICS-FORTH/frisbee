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

package service

import (
	"fmt"
	"reflect"

	"github.com/carv-ics-forth/frisbee/controllers/testplan/grafana"
	"github.com/carv-ics-forth/frisbee/controllers/utils"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func printWrongType(logger logr.Logger, expected interface{}, got client.Object) {
	logger.Error(errors.New("invalid type"), "invalid conversion",
		"expected", reflect.TypeOf(expected),
		"got", reflect.TypeOf(got),
	)
}

func (r *Controller) Watchers() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc:  r.create,
		DeleteFunc:  r.delete,
		UpdateFunc:  r.update,
		GenericFunc: generic,
	}
}

func (r *Controller) create(e event.CreateEvent) bool {
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

	if e.Object.GetAnnotations()[grafana.DrawAs] == grafana.DrawAsRegion {
		annotation := &grafana.RangeAnnotation{}
		annotation.Add(e.Object)

		r.regionAnnotations.Set(e.Object.GetName(), annotation)
	} else {
		annotation := &grafana.PointAnnotation{}
		annotation.Add(e.Object)
	}

	return false
}

func (r *Controller) update(e event.UpdateEvent) bool {
	if !utils.IsManagedByThisController(e.ObjectNew, r.gvk) {
		return false
	}

	if e.ObjectOld.GetResourceVersion() == e.ObjectNew.GetResourceVersion() {
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
	prev, ok := e.ObjectOld.(*corev1.Pod)
	if !ok {
		printWrongType(r.Logger, corev1.Pod{}, e.ObjectOld)

		return false
	}

	latest, ok := e.ObjectNew.(*corev1.Pod)
	if !ok {
		printWrongType(r.Logger, corev1.Pod{}, e.ObjectNew)

		return false
	}

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

func (r *Controller) delete(e event.DeleteEvent) bool {
	if !utils.IsManagedByThisController(e.Object, r.gvk) {
		return false
	}

	// An object was deleted but the watch deletion event was missed while disconnected from apiserver.
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

	if e.Object.GetAnnotations()[grafana.DrawAs] == grafana.DrawAsRegion {
		annotation, ok := r.regionAnnotations.Get(e.Object.GetName())
		if !ok {
			// this is a stall condition that happens when the controller is restarted. just ignore it
			return false
		}

		annotation.(*grafana.RangeAnnotation).Delete(e.Object)

		r.regionAnnotations.Remove(e.Object.GetName())
	} else {
		annotation := &grafana.PointAnnotation{}
		annotation.Delete(e.Object)
	}

	return true
}

func generic(event.GenericEvent) bool {
	return true
}
