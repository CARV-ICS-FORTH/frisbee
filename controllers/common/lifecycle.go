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

// ErrInvalidArgs indicate an error with calling arguments
var ErrInvalidArgs = errors.New("invalid arguments")

// InnerObject is an object that is managed and recognized by Frisbee (including services, servicegroups, ...)
type InnerObject interface {
	client.Object
	GetStatus() v1alpha1.EtherStatus
	SetStatus(v1alpha1.EtherStatus)
}

// ExternalToInnerObject is a wrapper for converting external objects (e.g, Pods) to InnerObjects managed
// by the Frisbee controller
type ExternalToInnerObject struct {
	client.Object

	StatusFunc func(obj interface{}) v1alpha1.EtherStatus
}

func (d *ExternalToInnerObject) GetStatus() v1alpha1.EtherStatus {
	return d.StatusFunc(d.Object)
}

func (d *ExternalToInnerObject) SetStatus(v1alpha1.EtherStatus) {
	panic(errors.Errorf("cannot set status on external object"))
}

func unwrap(obj client.Object) client.Object {
	wrapped, ok := obj.(*ExternalToInnerObject)
	if ok {
		return wrapped.Object
	}

	return obj
}

func accessStatus(obj interface{}) func(interface{}) v1alpha1.EtherStatus {
	external, ok := obj.(*ExternalToInnerObject)
	if ok {
		return external.StatusFunc
	}

	return func(inner interface{}) v1alpha1.EtherStatus {
		return inner.(InnerObject).GetStatus()
	}
}

func GetLifecycle(ctx context.Context, parentUID types.UID, child InnerObject, childrenNames ...string) *ManagedLifecycle {
	if child == nil || len(childrenNames) == 0 {
		common.logger.Error(ErrInvalidArgs, "lifecycle error")

		return nil
	}

	common.logger.Info("Watch children",
		"kind", reflect.TypeOf(unwrap(child)),
		"names", childrenNames,
	)

	n := newNotifier(childrenNames)
	n.accessChildStatus = accessStatus(child)

	handlers := eventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			annotateAdd(obj)
			n.watchChildrenPhase(obj)
		},
		UpdateFunc: n.watchChildrenPhase,
		DeleteFunc: func(obj interface{}) {
			annotateDelete(obj)
			n.watchChildrenPhase(obj)
		},
	}

	return &ManagedLifecycle{
		ctx:           ctx,
		n:             n,
		watchChildren: func() error { return watchLifecycle(ctx, parentUID, unwrap(child), handlers) },
	}
}

type ManagedLifecycle struct {
	ctx context.Context

	n *notifier

	watchChildren func() error
}

// UpdateParent run in a loops and continuously update the status of the parent.
func (lc *ManagedLifecycle) UpdateParent(parent InnerObject) error {
	if err := lc.watchChildren(); err != nil {
		return errors.Wrapf(err, "unable to watch children")
	}

	go func() {
		var phase v1alpha1.Phase
		var msg error
		var valid error

		phase, valid = lc.n.getNextPhase()

		for {
			if valid != nil {
				common.logger.Error(valid, "update parent failed", "parent", parent.GetName())

				return
			}

			common.logger.Info("Update Parent",
				"kind", reflect.TypeOf(parent),
				"name", parent.GetName(),
				"phase", phase,
			)

			switch phase {
			case v1alpha1.PhaseUninitialized:
				panic("this should not happen")

			case v1alpha1.PhaseRunning:
				_, _ = Running(lc.ctx, parent)
				phase, msg, valid = lc.n.getNextRunning()

			case v1alpha1.PhaseComplete:
				_, _ = Success(lc.ctx, parent)

				return

			case v1alpha1.PhaseFailed:
				_, _ = Failed(lc.ctx, parent, errors.Wrapf(msg, "at least one child has failed"))

				return

			case v1alpha1.PhaseChaos:
				_, _ = Chaos(lc.ctx, parent)

				// If the parent is in PhaseChaos mode, it means that failing conditions are expected and therefore
				// do not cause a failure to the controller. Instead, they should be handled by the system under evaluation.
				common.logger.Info("Expected abnormal event",
					"parent", parent.GetName(),
					"event", msg.Error(),
				)

				phase, msg, valid = lc.n.getNextChaos()

			default:
				valid = errors.Errorf("invalid phase %s ", phase)
			}
		}
	}()

	return nil
}

// Expect blocks waiting for the next phase of the parent. If it is the one expected, it returns nil.
// Otherwise it returns an error stating what was expected and what was got.
func (lc *ManagedLifecycle) Expect(expect v1alpha1.Phase) error {
	if err := lc.watchChildren(); err != nil {
		common.logger.Error(err, "unable to watch children")
		return err
	}

	var phase v1alpha1.Phase
	var msg error
	var valid error

	phase, _ = lc.n.getNextPhase()

	for {
		switch {
		case phase == expect:
			return nil // correct phase

		case errors.Is(valid, errIsFinal): // the transition is valid, but is not the expected one
			if msg == nil {
				return errors.Errorf("expected %s but got %s", expect, phase)
			}

			return errors.Errorf("expected %s but got %s (%s)", expect, phase, msg)

		case valid != nil:
			return valid // the transition is invalid
		}

		switch phase {
		case v1alpha1.PhaseUninitialized, v1alpha1.PhaseChaos:
			panic(errors.Errorf("cannot wait on phase %s", expect))

		case v1alpha1.PhaseRunning:
			phase, msg, valid = lc.n.getNextRunning()

		case v1alpha1.PhaseComplete:
			phase, msg, valid = lc.n.getNextComplete()

		case v1alpha1.PhaseFailed:
			phase, msg, valid = lc.n.getNextFailed()

		default:
			return errors.Errorf("invalid phase %s ", phase)
		}
	}
}

type eventHandlerFuncs struct {
	AddFunc    func(obj interface{})
	UpdateFunc func(obj interface{})
	DeleteFunc func(obj interface{})
}

func watchLifecycle(ctx context.Context, parentUID types.UID, child client.Object, handlers eventHandlerFuncs) error {
	informer, err := common.cache.GetInformer(ctx, child)
	if err != nil {
		return errors.Wrapf(err, "unable to get informer")
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
	})

	return nil
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

type notifier struct {
	accessChildStatus func(obj interface{}) v1alpha1.EtherStatus

	parentRunning   chan struct{}
	childrenRunning map[string]chan struct{}

	parentComplete   chan struct{}
	childrenComplete map[string]chan struct{}

	chaos  chan error
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
		"child", reflect.TypeOf(object),
		"name", object.GetName(),
		"phase", status.Phase,
	)

	switch status.Phase {
	case v1alpha1.PhaseUninitialized, v1alpha1.PhaseChaos:
		return

	case v1alpha1.PhaseRunning:
		ch := n.childrenRunning[object.GetName()]
		if ch == nil {
			return
		}

		n.childrenRunning[object.GetName()] = nil

		close(ch)

	case v1alpha1.PhaseComplete:
		ch := n.childrenComplete[object.GetName()]
		if ch == nil {
			return
		}

		n.childrenComplete[object.GetName()] = nil

		close(ch)

	case v1alpha1.PhaseFailed:
		if status.Reason == "" {
			status.Reason = "see the logs of failing pod."
		}

		n.failed <- errors.Errorf("object failed. object:%s, name:%s, error:%s",
			reflect.TypeOf(object),
			object.GetName(),
			status.Reason)
	}
}

// getNextPhase blocks waiting for the next of the parent.
func (n *notifier) getNextPhase() (v1alpha1.Phase, error) {
	select {
	case <-n.parentRunning:
		return v1alpha1.PhaseRunning, nil

	case <-n.parentComplete:
		return v1alpha1.PhaseComplete, nil

	case err := <-n.failed:
		return v1alpha1.PhaseFailed, err

	case err := <-n.chaos:
		return v1alpha1.PhaseChaos, err
	}
}

// listen for all the expected transition from Unitialized.
func (n *notifier) getNextUninitialized() (phase v1alpha1.Phase, msg error, valid error) {
	select {
	case <-n.parentRunning:
		return v1alpha1.PhaseRunning, nil, nil

	case <-n.parentComplete:
		return v1alpha1.PhaseComplete, nil,
			errors.Errorf("invalid transitiom %s -> %s", v1alpha1.PhaseUninitialized, v1alpha1.PhaseComplete)

	case err := <-n.failed:
		return v1alpha1.PhaseFailed, err, nil

	case err := <-n.chaos:
		return v1alpha1.PhaseChaos, err, nil
	}
}

// listen for all the expected transition from Running.
func (n *notifier) getNextRunning() (phase v1alpha1.Phase, msg error, valid error) {
	select {
	// ignore this case as it will lead to a loop
	// case <-n.parentRunning:  return v1alpha1.PhaseRunning, nil, nil

	case <-n.parentComplete:
		return v1alpha1.PhaseComplete, nil, nil

	case err := <-n.failed:
		return v1alpha1.PhaseFailed, err, nil

	case err := <-n.chaos:
		return v1alpha1.PhaseChaos, err, nil
	}
}

var errIsFinal = errors.New("phase is final")

// listen for all the expected transition from PhaseComplete.
func (n *notifier) getNextComplete() (phase v1alpha1.Phase, msg error, valid error) {
	return v1alpha1.PhaseComplete, nil, errIsFinal
}

// listen for all the expected transition from Failed.
func (n *notifier) getNextFailed() (phase v1alpha1.Phase, msg error, valid error) {
	return v1alpha1.PhaseFailed, nil, errIsFinal
}

// listen for all the expected transition from Chaos.
func (n *notifier) getNextChaos() (phase v1alpha1.Phase, msg error, valid error) {
	select {
	case <-n.parentRunning:
		return v1alpha1.PhaseRunning, nil,
			errors.Errorf("invalid transition %s -> %s", v1alpha1.PhaseChaos, v1alpha1.PhaseRunning)

	case <-n.parentComplete:
		return v1alpha1.PhaseComplete, nil, nil

	case err := <-n.failed:
		return v1alpha1.PhaseFailed, err,
			errors.Errorf("invalid transition %s -> %s", v1alpha1.PhaseChaos, v1alpha1.PhaseFailed)

		// ignore this case as it will lead to a loop
		// case err := <-n.chaos:
		//	return v1alpha1.PhaseChaos, err, nil
	}
}
