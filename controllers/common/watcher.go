package common

import (
	"context"
	"time"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/pkg/wait"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	cachetypes "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InnerObject interface {
	client.Object
	GetStatus() *v1alpha1.EtherStatus
}

// WaitForPhase blocks until the parent has reached a define state
func WaitForPhase(ctx context.Context, parentUID types.UID, childrenType client.Object, childrenNames []string, cond v1alpha1.Phase) error {
	t := watchForChildrenPhase(ctx, parentUID, childrenType, childrenNames)

	return t.waitParentPhase(ctx, cond)
}

// WatchStatusUpdates runs a background task to update the parent phase based on the phase of its children.
func WatchStatusUpdates(ctx context.Context, parent InnerObject, childrenType client.Object, childrenNames []string) {
	t := watchForChildrenPhase(ctx, parent.GetUID(), childrenType, childrenNames)

	go t.updateParentPhase(ctx, parent)
}

func watchForChildrenPhase(ctx context.Context, parentUID types.UID, childrenType client.Object, childrenNames []string) *notifier {
	informer, err := common.cache.GetInformer(ctx, childrenType)
	if err != nil {
		common.logger.Error(err, "unable to get service informer")

		return nil
	}

	t := newNotifier(childrenNames)

	// Set up an event handler for when ServiceGroup resources change. This
	// handler will lookup the owner of the given ServiceGroup, and if it is
	// owned by a Foo (ServiceGroup) resource will enqueue that Foo resource for
	// processing. This way, we don't need to implement custom logic for
	// handling ServiceGroup resources. More info on this pattern:
	// https://github.com/kubernetes/community/blob/8cafef897a22026d42f5e5bb3f104febe7e29830/contributors/devel/controllers.md
	informer.AddEventHandler(cachetypes.ResourceEventHandlerFuncs{
		AddFunc: filter(parentUID, nil),
		UpdateFunc: func(oldObj, newObj interface{}) {
			old := oldObj.(InnerObject)
			latest := newObj.(InnerObject)

			if old.GetResourceVersion() == latest.GetResourceVersion() {
				// Periodic resync will send update events for all known services.
				// Two different versions of the same Deployment will always have different RVs.
				return
			}

			filter(parentUID, t.watchChildren)(latest)
		},
		DeleteFunc: filter(parentUID, nil),
	})

	return t
}

// filter will take any resource implementing metav1.Object and attempt
// to find the Foo resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that Foo resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func filter(parentUID types.UID, handler func(obj interface{})) func(obj interface{}) {
	return func(obj interface{}) {
		if handler == nil {
			return
		}

		object, ok := obj.(metav1.Object)
		if !ok {
			// an object was deleted but the watch deletion event was missed while disconnected from apiserver.
			// In this case we don't know the final "resting" state of the object,
			// so there's a chance the included `Obj` is stale.
			tombstone, ok := obj.(cachetypes.DeletedFinalStateUnknown)
			if !ok {
				runtimeutil.HandleError(errors.New("error decoding object, invalid type"))

				return
			}

			object, ok = tombstone.Obj.(metav1.Object)
			if !ok {
				runtimeutil.HandleError(errors.New("error decoding object tombstone, invalid type"))

				return
			}
		}

		// update locate view of the dependent services
		for _, owner := range object.GetOwnerReferences() {
			if owner.UID == parentUID {
				handler(obj)

				return
			}
		}
	}
}

type notifier struct {
	parentRunning   chan struct{}
	childrenRunning map[string]chan struct{}

	parentSucceeded   chan struct{}
	childrenSucceeded map[string]chan struct{}

	failed chan error
}

func newNotifier(serviceNames []string) *notifier {
	parentRun, childrenRun := wait.ChannelWaitForChildren(len(serviceNames))
	parentExit, childrenExit := wait.ChannelWaitForChildren(len(serviceNames))

	t := &notifier{
		parentRunning:     parentRun,
		childrenRunning:   make(map[string]chan struct{}),
		parentSucceeded:   parentExit,
		childrenSucceeded: make(map[string]chan struct{}),
		failed:            make(chan error),
	}

	for i, name := range serviceNames {
		t.childrenRunning[name] = childrenRun[i]
		t.childrenSucceeded[name] = childrenExit[i]
	}

	return t
}

func (n *notifier) watchChildren(obj interface{}) {
	// To deliver idempotent operation, we want to ensure exactly-once notifications. However, an object may
	// be in the same phase during different iterations. For this reason, we close the channel only the first
	// and then we remove it. If the second iterations find that there is no channel for the given child,
	// it returns immediately as it assumes that the closing is already delivered.

	child := obj.(InnerObject)

	switch child.GetStatus().Phase {
	case v1alpha1.Uninitialized:
		return

	case v1alpha1.Running:
		ch, ok := n.childrenRunning[child.GetName()]
		if !ok {
			return
		}

		close(ch)

		delete(n.childrenRunning, child.GetName())

	case v1alpha1.Succeed:
		ch, ok := n.childrenSucceeded[child.GetName()]
		if !ok {
			return
		}

		close(ch)

		delete(n.childrenSucceeded, child.GetName())

	case v1alpha1.Failed:
		err := errors.Errorf(child.GetStatus().Reason)
		common.logger.Error(err, "Failed", "child", child.GetName())

		n.failed <- err
		close(n.failed)
	}
}

func (n *notifier) updateParentPhase(ctx context.Context, parent InnerObject) {
	// give it one minute for all instances to be up and running
	timeCtx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	select {
	case <-timeCtx.Done():
		common.logger.Error(ctx.Err(), "graceful period expired")

		return

	case <-n.parentRunning:
		if status := parent.GetStatus(); status.Phase != v1alpha1.Running {
			Running(ctx, parent)
		}

		select {
		case <-ctx.Done():
			common.logger.Error(ctx.Err(), "common failed")

		case <-n.parentSucceeded:
			if status := parent.GetStatus(); status.Phase != v1alpha1.Succeed {
				Success(ctx, parent)
			}

		case <-n.failed:
			if status := parent.GetStatus(); status.Phase != v1alpha1.Failed {
				Failed(ctx, parent, errors.New("at least one of the objects has failed"))
			}
		}
	}
}

// expect checks if the parent reaches the expected state. if not, it returns false.
func (n *notifier) waitParentPhase(ctx context.Context, cond v1alpha1.Phase) error {
	errCh := make(chan error, 1)
	defer close(errCh)

	stopCh := make(chan struct{})
	defer close(stopCh)

	// watch for faulty scenarios
	go func() {
		select {
		case <-stopCh:
			return
		case <-ctx.Done():
			errCh <- ctx.Err()
		case err := <-n.failed:
			errCh <- err
		}
	}()

	switch cond {
	case v1alpha1.Uninitialized:
		errCh <- errors.Errorf("unexpected phase")

	case v1alpha1.Running:
		<-n.parentRunning

	case v1alpha1.Succeed:
		<-n.parentSucceeded

	case v1alpha1.Failed:
		err := <-n.failed
		errCh <- err
	}

	return <-errCh
}
