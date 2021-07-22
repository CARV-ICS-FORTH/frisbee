package common

import (
	"context"
	"reflect"
	"time"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/pkg/wait"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	cachetypes "k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

/******************************************************
			Lifecycle Setters
/******************************************************/

// InnerObject is an object that is managed and recognized by Frisbee (including services, servicegroups, ...)
type InnerObject interface {
	client.Object
	GetLifecycle() v1alpha1.Lifecycle
	SetLifecycle(v1alpha1.Lifecycle)
}

// Discoverable is a wrapper that sets phase to Discoverable and does not requeue the request.
func Discoverable(ctx context.Context, obj InnerObject) (ctrl.Result, error) {
	status := obj.GetLifecycle()

	status.Phase = v1alpha1.PhaseDiscoverable
	status.Reason = "waiting for dependencies"

	obj.SetLifecycle(status)

	return UpdateStatus(ctx, obj)
}

// Pending is a wrapper that sets phase to Pending and does not requeue the request.
func Pending(ctx context.Context, obj InnerObject) (ctrl.Result, error) {
	status := obj.GetLifecycle()

	status.Phase = v1alpha1.PhasePending
	status.Reason = "waiting for dependencies"

	obj.SetLifecycle(status)

	return UpdateStatus(ctx, obj)
}

// Running is a wrapper that sets phase to Running and does not requeue the request.
func Running(ctx context.Context, obj InnerObject) (ctrl.Result, error) {
	status := obj.GetLifecycle()

	status.Phase = v1alpha1.PhaseRunning
	status.Reason = "OK"

	obj.SetLifecycle(status)

	return UpdateStatus(ctx, obj)
}

// Success is a wrapper that sets phase to Success and does not requeue the request.
func Success(ctx context.Context, obj InnerObject) (ctrl.Result, error) {
	status := obj.GetLifecycle()

	status.Phase = v1alpha1.PhaseSuccess
	status.Reason = "All children are succeeded"

	obj.SetLifecycle(status)

	return UpdateStatus(ctx, obj)
}

// Chaos is a wrapper that sets phase to Chaos and does not requeue the request.
func Chaos(ctx context.Context, obj InnerObject) (ctrl.Result, error) {
	status := obj.GetLifecycle()

	status.Phase = v1alpha1.PhaseChaos
	status.Reason = "Expect controlled failures"

	obj.SetLifecycle(status)

	return UpdateStatus(ctx, obj)
}

// Failed is a wrap that logs the error, updates the status, and does not requeue the request.
func Failed(ctx context.Context, obj InnerObject, err error) (ctrl.Result, error) {
	runtimeutil.HandleError(errors.Wrapf(err, "object %s has failed", obj.GetName()))

	status := obj.GetLifecycle()
	status.Phase = v1alpha1.PhaseFailed
	status.Reason = err.Error()

	obj.SetLifecycle(status)

	return UpdateStatus(ctx, obj)
}

/******************************************************
	Wrappers and Unwrappers for InnerObjects
/******************************************************/

// externalToInnerObject is a wrapper for converting external objects (e.g, Pods) to InnerObjects managed
// by the Frisbee controller
type externalToInnerObject struct {
	client.Object

	LifecycleFunc func(obj interface{}) v1alpha1.Lifecycle
}

func (d *externalToInnerObject) GetLifecycle() v1alpha1.Lifecycle {
	return d.LifecycleFunc(d.Object)
}

func (d *externalToInnerObject) SetLifecycle(v1alpha1.Lifecycle) {
	panic(errors.Errorf("cannot set status on external object"))
}

func unwrap(obj client.Object) client.Object {
	wrapped, ok := obj.(*externalToInnerObject)
	if ok {
		return wrapped.Object
	}

	return obj
}

func accessStatus(obj interface{}) func(interface{}) v1alpha1.Lifecycle {
	external, ok := obj.(*externalToInnerObject)
	if ok {
		return external.LifecycleFunc
	}

	return func(inner interface{}) v1alpha1.Lifecycle {
		return inner.(InnerObject).GetLifecycle()
	}
}

/******************************************************
			Lifecycle Getters
******************************************************/

type LifecycleOptions struct {
	// Timeout defines an expiration deadline for the request
	Timeout *time.Duration

	// FilterFunc is used to filter out events before they reach the lifecycle handler
	Filter FilterFunc

	// WatchType indicate the type of objects to watch
	WatchType InnerObject

	// ChildrenNames is a list of children names to watch
	ChildrenNames []string

	// Annotation indicate whether events shall be recorded in Grafana or not
	Annotator bool
}

type LifecycleOption func(*LifecycleOptions)

// Obligatory
func Watch(kind InnerObject, names ...string) LifecycleOption {
	return func(s *LifecycleOptions) {
		if s.WatchType != nil && s.WatchType != kind {
			panic("watch type is already defined")
		} else {
			s.WatchType = kind
		}

		// if the watch is of the same type, just append new names
		s.ChildrenNames = append(s.ChildrenNames, names...)
	}
}

// WatchExternal wraps an object that does not belong to this controller
func WatchExternal(kind client.Object, convertor func(obj interface{}) v1alpha1.Lifecycle, names ...string) LifecycleOption {
	return func(s *LifecycleOptions) {
		if s.WatchType != nil && s.WatchType != kind {
			panic("watch type is already defined")
		} else {
			s.WatchType = &externalToInnerObject{
				Object:        kind,
				LifecycleFunc: convertor,
			}
		}

		// if the watch is of the same type, just append new names
		s.ChildrenNames = append(s.ChildrenNames, names...)
	}
}

// Optional
func WithTimeout(duration time.Duration) LifecycleOption {
	return func(s *LifecycleOptions) {
		s.Timeout = &duration
	}
}

// Optional
func WithFilter(filter FilterFunc) LifecycleOption {
	return func(s *LifecycleOptions) {
		if s.Filter != nil {
			panic("filter already exists")
		}

		s.Filter = filter
	}
}

// WithAnnotator will send annotations to grafana whenever an Add or a Delete event takes place.
func WithAnnotator(do bool) LifecycleOption {
	return func(s *LifecycleOptions) {
		s.Annotator = do
	}
}

// WaitReady is a lifecycle wrapper that waits until an object has reached the Running phase.
// If another phase is reached (e.g, failed), it returns error.
func WaitReady(ctx context.Context, obj InnerObject) error {
	return GetLifecycle(ctx,
		Watch(obj, obj.GetName()),
	).Expect(v1alpha1.PhaseRunning)
}

// WaitSuccess is a lifecycle wrapper that waits until an object has reached the Running phase.
// If another phase is reached (e.g, failed), it returns error.
func WaitSuccess(ctx context.Context, obj InnerObject) error {
	return GetLifecycle(ctx,
		Watch(obj, obj.GetName()),
	).Expect(v1alpha1.PhaseSuccess)
}

func GetLifecycle(ctx context.Context, opts ...LifecycleOption) *ManagedLifecycle {
	var options LifecycleOptions

	// call option functions on instance to set options on it
	for _, apply := range opts {
		apply(&options)
	}

	var lifecycle ManagedLifecycle

	// set context
	if options.Timeout == nil {
		lifecycle.ctx = ctx
	} else {
		timeout, cancel := context.WithTimeout(ctx, *options.Timeout)

		lifecycle.ctx = timeout
		defer cancel()
	}

	// set filter
	if options.Filter == nil {
		lifecycle.filterFunc = NoFilter
	} else {
		lifecycle.filterFunc = options.Filter
	}

	var n *notifier

	// set children
	if len(options.ChildrenNames) == 0 {
		panic("empty children")
	} else {
		n = newNotifier(options.ChildrenNames)
	}

	// set watchtype
	if options.WatchType == nil {
		panic("empty watchtype")
	} else {
		n.accessChildStatus = accessStatus(options.WatchType)
	}

	var handlers eventHandlerFuncs

	if options.Annotator {
		handlers = eventHandlerFuncs{
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
	} else {
		handlers = eventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				n.watchChildrenPhase(obj)
			},
			UpdateFunc: n.watchChildrenPhase,
			DeleteFunc: func(obj interface{}) {
				n.watchChildrenPhase(obj)
			},
		}
	}

	lifecycle.notifier = n
	lifecycle.watchChildren = func() error {
		return watchLifecycle(ctx, lifecycle.filterFunc, unwrap(options.WatchType), handlers)
	}

	return &lifecycle
}

type ManagedLifecycle struct {
	ctx context.Context

	*notifier

	filterFunc func(obj interface{}) bool

	watchChildren func() error
}

// UpdateParentLifecycle run in a loops and continuously Update the status of the parent.
func (lc *ManagedLifecycle) UpdateParentLifecycle(parent InnerObject) error {
	if err := lc.watchChildren(); err != nil {
		return errors.Wrapf(err, "unable to watch children")
	}

	var phase v1alpha1.Phase

	var msg error

	var valid error

	go func() {
		phase, valid = lc.notifier.getNextPhase()

		for {
			if valid != nil {
				common.logger.Error(valid, "Update parent failed", "parent", parent.GetName())

				return
			}

			common.logger.Info("Update Parent",
				"kind", reflect.TypeOf(parent),
				"name", parent.GetName(),
				"phase", phase,
			)

			switch phase {
			case v1alpha1.PhaseUninitialized, v1alpha1.PhaseDiscoverable, v1alpha1.PhasePending:
				panic("this should not happen")

			case v1alpha1.PhaseRunning:
				_, _ = Running(lc.ctx, parent)
				phase, msg, valid = lc.notifier.getNextRunning()

			case v1alpha1.PhaseChaos:
				_, _ = Chaos(lc.ctx, parent)

				// If the parent is in PhaseChaos mode, it means that failing conditions are expected and therefore
				// do not cause a failure to the controller. Instead, they should be handled by the system under evaluation.
				common.logger.Info("Expected abnormal event",
					"parent", parent.GetName(),
					"event", msg.Error(),
				)

				phase, msg, valid = lc.notifier.getNextChaos()

			case v1alpha1.PhaseSuccess:
				_, _ = Success(lc.ctx, parent)

				return

			case v1alpha1.PhaseFailed:
				_, _ = Failed(lc.ctx, parent, msg)

				return

			default:
				valid = errors.Errorf("invalid phase %s ", phase)
			}
		}
	}()

	return nil
}

// Expect blocks waiting for the next phase of the parent. If it is the one expected, it returns nil.
// Otherwise it returns an error stating what was expected and what was got.
func (lc *ManagedLifecycle) Expect(expected v1alpha1.Phase) error {
	if err := lc.watchChildren(); err != nil {
		common.logger.Error(err, "unable to watch children")

		return err
	}

	phase, err := lc.notifier.getNextPhase()
	if err != nil {
		return err
	}

	var msg, valid error

	for {
		switch {
		case phase == expected: // correct phase
			return nil

		case errors.Is(valid, errIsFinal): // the transition is valid, but not the expected one
			if msg == nil {
				return errors.Errorf("expected %s but got %s", expected, phase)
			}

			return errors.Errorf("expected %s but got %s (%s)", expected, phase, msg)

		case valid != nil:
			return errors.Wrapf(valid, "invalid transition")
		}

		switch phase {
		case v1alpha1.PhaseUninitialized, v1alpha1.PhaseChaos, v1alpha1.PhaseDiscoverable, v1alpha1.PhasePending:
			panic(errors.Errorf("cannot wait on phase %s", expected))

		case v1alpha1.PhaseRunning:
			phase, msg, valid = lc.notifier.getNextRunning()

		case v1alpha1.PhaseSuccess:
			phase, msg, valid = lc.notifier.getNextComplete()

		case v1alpha1.PhaseFailed:
			phase, msg, valid = lc.notifier.getNextFailed()

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

func watchLifecycle(ctx context.Context, filterFunc func(obj interface{}) bool, child client.Object, handlers eventHandlerFuncs) error {
	if filterFunc == nil {
		panic("empty filter")
	}

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
		FilterFunc: filterFunc,
		Handler: cachetypes.ResourceEventHandlerFuncs{
			AddFunc: handlers.AddFunc,
			UpdateFunc: func(oldObj, newObj interface{}) {
				old, ok := oldObj.(metav1.Object)
				if !ok {
					panic("this should never happen")
				}

				latest, ok := newObj.(metav1.Object)
				if !ok {
					panic("this should never happen")
				}

				if old.GetResourceVersion() == latest.GetResourceVersion() {
					// Periodic resync will send Update events for all known services.
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

type notifier struct {
	accessChildStatus func(obj interface{}) v1alpha1.Lifecycle

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

// watchChildren monitors the phase of children and Update the channels of the parent.
func (n *notifier) watchChildrenPhase(obj interface{}) {
	// To deliver idempotent operation, we want to ensure exactly-once notifications. However, an object may
	// be in the same phase during different iterations. For this reason, we close the channel only the first
	// and then we nil it. If the second iterations finds a nil for the given child,
	// it returns immediately as it assumes that the closing is already delivered.
	object, ok := obj.(client.Object)
	if !ok {
		panic("this should never happen")
	}

	status := n.accessChildStatus(obj)

	switch status.Phase {
	case v1alpha1.PhaseUninitialized, v1alpha1.PhaseDiscoverable, v1alpha1.PhasePending, v1alpha1.PhaseChaos:
		return

	case v1alpha1.PhaseRunning:
		ch := n.childrenRunning[object.GetName()]
		if ch == nil {
			return
		}

		n.childrenRunning[object.GetName()] = nil

		close(ch)

	case v1alpha1.PhaseSuccess:
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

		n.failed <- errors.Errorf("object:%s, name:%s, error:%s",
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
		return v1alpha1.PhaseSuccess, nil

	case err := <-n.failed:
		return v1alpha1.PhaseFailed, err

	case err := <-n.chaos:
		return v1alpha1.PhaseChaos, err
	}
}

// listen for all the expected transition from Unitialized.
func (n *notifier) getNextUninitialized() (phase v1alpha1.Phase, msg, valid error) {
	select {
	case <-n.parentRunning:
		return v1alpha1.PhaseRunning, nil, nil

	case <-n.parentComplete:
		return v1alpha1.PhaseSuccess, nil,
			errors.Errorf("invalid transitiom %s -> %s", v1alpha1.PhaseUninitialized, v1alpha1.PhaseSuccess)

	case err := <-n.failed:
		return v1alpha1.PhaseFailed, err, nil

	case err := <-n.chaos:
		return v1alpha1.PhaseChaos, err, nil
	}
}

// listen for all the expected transition from Running.
func (n *notifier) getNextRunning() (phase v1alpha1.Phase, msg, valid error) {
	select {
	// ignore this case as it will lead to a loop
	// case <-n.parentRunning:  return v1alpha1.PhaseRunning, nil, nil

	case <-n.parentComplete:
		return v1alpha1.PhaseSuccess, nil, nil

	case err := <-n.failed:
		return v1alpha1.PhaseFailed, err, nil

	case err := <-n.chaos:
		return v1alpha1.PhaseChaos, err, nil
	}
}

var errIsFinal = errors.New("phase is final")

// listen for all the expected transition from PhaseSuccess.
func (n *notifier) getNextComplete() (phase v1alpha1.Phase, msg, valid error) {
	return v1alpha1.PhaseSuccess, nil, errIsFinal
}

// listen for all the expected transition from Failed.
func (n *notifier) getNextFailed() (phase v1alpha1.Phase, msg, valid error) {
	return v1alpha1.PhaseFailed, nil, errIsFinal
}

// listen for all the expected transition from Chaos.
func (n *notifier) getNextChaos() (phase v1alpha1.Phase, msg, valid error) {
	select {
	case <-n.parentRunning:
		return v1alpha1.PhaseRunning, nil,
			errors.Errorf("invalid transition %s -> %s", v1alpha1.PhaseChaos, v1alpha1.PhaseRunning)

	case <-n.parentComplete:
		return v1alpha1.PhaseSuccess, nil, nil

	case err := <-n.failed:
		return v1alpha1.PhaseFailed, err,
			errors.Errorf("invalid transition %s -> %s", v1alpha1.PhaseChaos, v1alpha1.PhaseFailed)

		// ignore this case as it will lead to a loop due to the active channel
		// case err := <-n.chaos:
		//	return v1alpha1.PhaseChaos, err, nil
	}
}

/******************************************************
			Lifecycle Filters
******************************************************/

type FilterFunc func(obj interface{}) bool

// FilterParent applies the provided FilterParent to all events coming in, and decides which events will be handled
// by this controller. It does this by looking at the objects metadata.ownerReferences field for an
// appropriate OwnerReference. It then enqueues that Foo resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped. If the parent is empty, the object is passed
// as if it belongs to this parent.
func FilterParent(parentUID types.UID) FilterFunc {
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

		if len(parentUID) == 0 {
			return true
		}

		// Update locate view of the dependent services
		for _, owner := range object.GetOwnerReferences() {
			if owner.UID == parentUID {
				return true
			}
		}

		return false
	}
}

// NoFilter is a passthrough filter that allows all events to pass to the handler
func NoFilter(obj interface{}) bool { return true }
