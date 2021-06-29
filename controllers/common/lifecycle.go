package common

import (
	"context"
	"reflect"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/pkg/wait"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	cachetypes "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type eventHandlerFuncs struct {
	AddFunc    func(obj interface{})
	UpdateFunc func(obj interface{})
	DeleteFunc func(obj interface{})
}

func WatchLifecycle(ctx context.Context, parentUID types.UID, child client.Object, handlers eventHandlerFuncs) {
	informer, err := common.cache.GetInformer(ctx, child)
	if err != nil {
		runtimeutil.HandleError(err)

		return
	}

	// Set up an event handler for when ServiceGroup resources change. This
	// handler will lookup the owner of the given ServiceGroup, and if it is
	// owned by a Foo (ServiceGroup) resource will enqueue that Foo resource for
	// processing. This way, we don't need to implement custom logic for
	// handling ServiceGroup resources. More info on this pattern:
	// https://github.com/kubernetes/community/blob/8cafef897a22026d42f5e5bb3f104febe7e29830/contributors/devel/controllers.md
	informer.AddEventHandler(cachetypes.FilteringResourceEventHandler{
		FilterFunc: filter(parentUID),
		Handler: cachetypes.ResourceEventHandlerFuncs{
			AddFunc: handlers.AddFunc,
			UpdateFunc: func(oldObj, newObj interface{}) {
				old := oldObj.(metav1.Object)
				latest := newObj.(metav1.Object)

				if old.GetResourceVersion() == latest.GetResourceVersion() {
					// Periodic resync will send update events for all known services.
					// Two different versions of the same Deployment will always have different RVs.
					return
				}

				handlers.UpdateFunc(latest)
			},
			DeleteFunc: handlers.DeleteFunc,
		},
	},
	)
}

// filter applies the provided filter to all events coming in, and decides which events will be handled
// by this controller. It does this by looking at the objects metadata.ownerReferences field for an
// appropriate OwnerReference. It then enqueues that Foo resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func filter(parentUID types.UID) func(obj interface{}) bool {
	return func(obj interface{}) bool {
		if obj == nil {
			return false
		}

		object, ok := obj.(metav1.Object)
		if !ok {
			// an object was deleted but the watch deletion event was missed while disconnected from apiserver.
			// In this case we don't know the final "resting" state of the object,
			// so there's a chance the included `Obj` is stale.
			tombstone, ok := obj.(cachetypes.DeletedFinalStateUnknown)
			if !ok {
				runtimeutil.HandleError(errors.New("error decoding object, invalid type"))

				return false
			}

			object, ok = tombstone.Obj.(metav1.Object)
			if !ok {
				runtimeutil.HandleError(errors.New("error decoding object tombstone, invalid type"))

				return false
			}
		}

		// update locate view of the dependent services
		for _, owner := range object.GetOwnerReferences() {
			if owner.UID == parentUID {
				return true
			}
		}

		return false
	}
}

// UpdateLifecycle runs a background task to update the parent phase based on the phase of its children.
func UpdateLifecycle(ctx context.Context, parent InnerObject, child client.Object, childrenNames ...string) {
	if parent == nil || child == nil || len(childrenNames) == 0 {
		return
	}

	t := newNotifier(childrenNames)
	t.accessChildStatus = accessStatus(child)

	handlers := eventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			trackAdd(obj)
			t.watchChildrenPhase(obj)
		},
		UpdateFunc: t.watchChildrenPhase,
		DeleteFunc: func(obj interface{}) {
			trackDelete(obj)
			t.watchChildrenPhase(obj)
		},
	}

	WatchLifecycle(ctx, parent.GetUID(), unwrap(child), handlers)

	go t.updateParent(ctx, parent)

	common.logger.Info("Watch",
		"parent-kind", reflect.TypeOf(parent),
		"name", parent.GetName(),
		"child-kind", reflect.TypeOf(child),
		"children", childrenNames,
	)
}

// WaitLifecycle blocks until the parent has reached a define state
func WaitLifecycle(ctx context.Context, parentUID types.UID, child client.Object, childrenNames []string,
	expected v1alpha1.Phase) error {

	t := newNotifier(childrenNames)
	t.accessChildStatus = accessStatus(child)

	handlers := eventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			trackAdd(obj)
			t.watchChildrenPhase(obj)
		},
		UpdateFunc: t.watchChildrenPhase,
		DeleteFunc: func(obj interface{}) {
			trackDelete(obj)
			t.watchChildrenPhase(obj)
		},
	}

	WatchLifecycle(ctx, parentUID, child, handlers)

	return t.waitParent(ctx, expected)
}

type notifier struct {
	accessChildStatus func(obj interface{}) v1alpha1.EtherStatus

	parentRunning   chan struct{}
	childrenRunning map[string]chan struct{}

	parentComplete   chan struct{}
	childrenComplete map[string]chan struct{}

	assess chan error
	failed chan error
}

func newNotifier(serviceNames []string) *notifier {
	parentRun, childrenRun := wait.ChannelWaitForChildren(len(serviceNames))
	parentExit, childrenExit := wait.ChannelWaitForChildren(len(serviceNames))

	t := &notifier{
		parentRunning:    parentRun,
		childrenRunning:  make(map[string]chan struct{}),
		parentComplete:   parentExit,
		childrenComplete: make(map[string]chan struct{}),
		failed:           make(chan error),
	}

	for i, name := range serviceNames {
		t.childrenRunning[name] = childrenRun[i]
		t.childrenComplete[name] = childrenExit[i]
	}

	return t
}

// watchChildren monitors the phase of children and update the channels of the parent.
func (n *notifier) watchChildrenPhase(obj interface{}) {
	// To deliver idempotent operation, we want to ensure exactly-once notifications. However, an object may
	// be in the same phase during different iterations. For this reason, we close the channel only the first
	// and then we nil it. If the second iterations finds a nil for the given child,
	// it returns immediately as it assumes that the closing is already delivered.
	object := obj.(client.Object)

	status := n.accessChildStatus(obj)

	common.logger.Info("Child Changed",
		"kind", reflect.TypeOf(object),
		"name", object.GetName(),
		"phase", status.Phase,
	)

	switch status.Phase {
	case v1alpha1.Uninitialized:
		return

	case v1alpha1.Running:
		ch := n.childrenRunning[object.GetName()]
		if ch == nil {
			return
		}

		n.childrenRunning[object.GetName()] = nil

		close(ch)

	case v1alpha1.Complete:
		ch := n.childrenComplete[object.GetName()]
		if ch == nil {
			return
		}

		n.childrenComplete[object.GetName()] = nil

		close(ch)

	case v1alpha1.Failed:
		switch obj.(type) {
		case ExternalToInnerObject:
			n.failed <- errors.Errorf("external object %s has failed. error: %s", object.GetName(), status.Reason)
		case InnerObject:
			n.failed <- errors.Errorf("managed object %s has failed. error: %s", object.GetName(), status.Reason)
		default:
			n.failed <- errors.Errorf("unknown object %s has failed. error: %s", object.GetName(), status.Reason)
		}
		// close(n.failed)
	}
}

// updateParent updates the phase of the parent based on the phase of its children.
// Upon completion, the parent is automatically removed
func (n *notifier) updateParent(ctx context.Context, parent InnerObject) {
	failure := func(err error) {
		switch parent.GetStatus().Phase {
		case v1alpha1.Failed:
			// Nothing to do if the parent has already failed. Just log the error
			runtimeutil.HandleError(err)

		case v1alpha1.Chaos:
			// If the parent is in Chaos mode, it means that failing conditions are expected and therefore
			// do not cause a failure to the controller. Instead, they should be handled by the system under evaluation.
			common.logger.Info("Expected abnormal event",
				"parent", parent.GetName(),
				"event", err.Error(),
			)

		default:
			// In any other case, mark the parent as failed
			_, _ = Failed(ctx, parent, errors.Wrapf(err, "at least one of children for %s has failed",
				parent.GetName()))
		}
	}

	terminal := func() {
		select {
		case <-ctx.Done():
			runtimeutil.HandleError(ctx.Err())

			return

		case <-n.parentComplete:
			if status := parent.GetStatus(); status.Phase != v1alpha1.Complete {
				_, _ = Success(ctx, parent)
			}

		case err := <-n.failed:
			failure(err)
		}
	}

	select {
	case <-ctx.Done():
		runtimeutil.HandleError(ctx.Err())

		return

	case <-n.parentRunning:
		if status := parent.GetStatus(); status.Phase != v1alpha1.Running {
			_, _ = Running(ctx, parent)
		}

		common.logger.Info("Update Parent",
			"kind", reflect.TypeOf(parent),
			"name", parent.GetName(),
			"phase", parent.GetStatus().Phase,
		)

		terminal()

	case <-n.assess:
		if status := parent.GetStatus(); status.Phase != v1alpha1.Chaos {
			_, _ = Chaos(ctx, parent)
		}

		common.logger.Info("Update Parent",
			"kind", reflect.TypeOf(parent),
			"name", parent.GetName(),
			"phase", parent.GetStatus().Phase,
		)

		terminal()

	case <-n.parentComplete:
		if status := parent.GetStatus(); status.Phase != v1alpha1.Complete {
			_, _ = Success(ctx, parent)
		}

	case err := <-n.failed:
		failure(err)
	}
}

// waitParent blocks waiting for the next phase of the parent. If it is the one expected, it returns nil.
// Otherwise it returns an error stating what was expected and what was got.
func (n *notifier) waitParent(ctx context.Context, cond v1alpha1.Phase) error {
	switch cond {
	case v1alpha1.Uninitialized:
		return errors.Errorf("Uninitialized phase cannot be waited")

	case v1alpha1.Chaos:
		return errors.Errorf("Chaos phase cannot be waited")

	case v1alpha1.Running:
		select {
		case <-ctx.Done():
			return errors.Wrapf(ctx.Err(), "operation cancelled")
		case <-n.parentRunning:
			return nil
		case <-n.parentComplete:
			return errors.Errorf("expected %s but got %s", cond, v1alpha1.Complete)
		case err := <-n.failed:
			return errors.Errorf("expected %s but got %s due to %s", cond, v1alpha1.Failed, err)
		}

	case v1alpha1.Complete: // To reach completed, it must have first be running
		select {
		case <-ctx.Done():
			return errors.Wrapf(ctx.Err(), "operation cancelled")
		case <-n.parentRunning:
			select {
			case <-ctx.Done():
				return errors.Wrapf(ctx.Err(), "operation cancelled")
			case <-n.parentComplete:
				return nil
			case err := <-n.failed:
				return errors.Errorf("expected %s but got %s due to %s", cond, v1alpha1.Failed, err)
			}
		}

	case v1alpha1.Failed: // To reach failed, it may or may not first be running
		select {
		case <-ctx.Done():
			return errors.Wrapf(ctx.Err(), "operation cancelled")
		case <-n.parentRunning:
			select {
			case <-ctx.Done():
				return errors.Wrapf(ctx.Err(), "operation cancelled")
			case <-n.parentComplete:
				return errors.Errorf("expected %s but got %s", cond, v1alpha1.Complete)
			case <-n.failed:
				return nil
			}
		}

	default:
		return errors.Errorf("unknown phase %s", cond)
	}
}

func trackAdd(obj interface{}) {
	objMeta := obj.(metav1.Object)

	common.logger.Info("Add Child",
		"kind", reflect.TypeOf(obj),
		"name", objMeta.GetName(),
		"labels", objMeta.GetLabels())

	// todo: add Grafana annotations
}

func trackDelete(obj interface{}) {
	objMeta := obj.(metav1.Object)

	common.logger.Info("Delete Child",
		"kind", reflect.TypeOf(obj),
		"name", objMeta.GetName(),
		"labels", objMeta.GetLabels())

	// todo: add Grafana annotations
}
