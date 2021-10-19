package cluster

import (
	"fmt"
	"reflect"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/utils"
	"github.com/pkg/errors"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func (r *Controller) WatchServices() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc:  r.watchServiceCreate,
		DeleteFunc:  r.watchServiceDelete,
		UpdateFunc:  r.watchServiceUpdate,
		GenericFunc: r.watchServiceGeneric,
	}
}

func (r *Controller) watchServiceCreate(e event.CreateEvent) bool {
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

func (r *Controller) watchServiceUpdate(e event.UpdateEvent) bool {
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
	oldService := e.ObjectOld.(*v1alpha1.Service)
	newService := e.ObjectNew.(*v1alpha1.Service)

	if oldService.Status.Phase == newService.Status.Phase {
		// a controller never initiates a phase change, and so is never asleep waiting for the same.
		return false
	}

	r.Logger.Info("** Detected",
		"Request", "Update",
		"kind", reflect.TypeOf(e.ObjectNew),
		"name", e.ObjectNew.GetName(),
		"from", oldService.Status.Phase,
		"to", newService.Status.Phase,
		"version", fmt.Sprintf("%s -> %s", oldService.GetResourceVersion(), newService.GetResourceVersion()),
	)

	return true
}

func (r *Controller) watchServiceDelete(e event.DeleteEvent) bool {
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

func (r *Controller) watchServiceGeneric(event.GenericEvent) bool {
	return true
}
