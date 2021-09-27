package chaos

import (
	"github.com/fnikolai/frisbee/controllers/utils/lifecycle"
	"github.com/pkg/errors"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
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

func (r *Controller) create(e event.CreateEvent) bool {
	if !e.Object.GetDeletionTimestamp().IsZero() {
		// on a restart of the controller manager, it's possible a new pod shows up in a state that
		// is already pending deletion. Prevent the pod from being a creation observation.
		return false
	}

	r.Logger.Info("Detected: fault creation", "name ", e.Object.GetName())

	// because the range annotator has state (uid), we need to save in the controller's store.
	annotator := &lifecycle.RangeAnnotation{}
	annotator.Add(e.Object)

	r.annotators.Set(e.Object.GetName(), annotator)

	return true
}

func (r *Controller) update(e event.UpdateEvent) bool {
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
	oldFault := e.ObjectOld.(*Fault)
	newFault := e.ObjectNew.(*Fault)

	oldLF := CalculateLifecycle(oldFault)
	newLF := CalculateLifecycle(newFault)

	if oldLF.Phase == newLF.Phase {
		return false
	}

	return true
}

func (r *Controller) delete(e event.DeleteEvent) bool {
	// An object was deleted but the watch deletion event was missed while disconnected from apiserver.
	// In this case we don't know the final "resting" state of the object,
	// so there's a chance the included `Obj` is stale.
	if e.DeleteStateUnknown {
		runtimeutil.HandleError(errors.Errorf("couldn't get object from tombstone %+v", e.Object))

		return false
	}

	r.Logger.Info("Detected: fault deletion", "name ", e.Object.GetName())

	annotator, ok := r.annotators.Get(e.Object.GetName())
	if !ok {
		panic("this should never happen")
	}

	annotator.(*lifecycle.RangeAnnotation).Delete(e.Object)

	r.annotators.Remove(e.Object.GetName())

	return true
}

func generic(event.GenericEvent) bool {
	return true
}
