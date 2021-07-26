package lifecycle

import (
	"context"
	"reflect"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/pkg/wait"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cachetypes "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// InnerObject is an object that is managed and recognized by Frisbee (including services, servicegroups, ...)
type InnerObject interface {
	client.Object
	GetLifecycle() v1alpha1.Lifecycle
	SetLifecycle(v1alpha1.Lifecycle)
}

/******************************************************
			Lifecycle Manager
******************************************************/

// LifecycleOptions is a set of options that will be used to instantiate a new Watchdog
type LifecycleOptions struct {
	// FilterFunc is used to filter out events before they reach the lifecycle handler
	Filter FilterFunc

	// WatchType indicate the type of objects to watch
	WatchType InnerObject

	// ChildrenNames is a list of children names to watch
	ChildrenNames []string

	// Annotator indicate whether events shall be recorded in Grafana or not
	Annotator Annotator

	// Logger is a logger for recording asynchronous operations
	Logger logr.Logger
}

// LifecycleOption is a wrapper for Functional Options
type LifecycleOption func(*LifecycleOptions)

// NewWatchdog listens of events that happen on object of the given type and names.
func NewWatchdog(kind InnerObject, names ...string) LifecycleOption {
	return func(s *LifecycleOptions) {
		if kind == nil || len(names) == 0 {
			panic("invalid arguments")
		}

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
		if kind == nil || convertor == nil || len(names) == 0 {
			panic("invalid arguments")
		}

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

// WithFilter add a filter that allows only specific object to reach function like Expect and Wait.
// Most commonly, this is used in conjunction with ParentFilter that filters only events that belongs
// to the given parent.
func WithFilter(filter FilterFunc) LifecycleOption {
	return func(s *LifecycleOptions) {
		if s.Filter != nil {
			panic("filter already exists")
		}

		s.Filter = filter
	}
}

// WithAnnotator will send annotations to grafana whenever an object is added or delete.
func WithAnnotator(annotator Annotator) LifecycleOption {
	return func(s *LifecycleOptions) {
		if annotator == nil {
			panic("empty annotator")
		}

		s.Annotator = annotator
	}
}

// WithLogger appends a new logger for the lifecycle
func WithLogger(logger logr.Logger) LifecycleOption {
	return func(s *LifecycleOptions) {
		if s.Logger != nil {
			panic("logger already exists")
		}

		s.Logger = logger
	}
}

func New(ctx context.Context, opts ...LifecycleOption) *ManagedLifecycle {
	var options LifecycleOptions

	// call option functions on instance to set options on it
	for _, apply := range opts {
		apply(&options)
	}

	var lifecycle ManagedLifecycle
	lifecycle.ctx = ctx

	// set logger
	if options.Logger == nil {
		lifecycle.logger = logr.Discard()
	} else {
		lifecycle.logger = options.Logger
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

	if options.Annotator != nil {
		handlers = eventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				options.Annotator.Add(obj)

				n.watchChildrenPhase(obj)
			},
			UpdateFunc: n.watchChildrenPhase,
			DeleteFunc: func(obj interface{}) {
				options.Annotator.Delete(obj)

				n.watchChildrenPhase(obj)
			},
		}
	} else {
		handlers = eventHandlerFuncs{
			AddFunc:    n.watchChildrenPhase,
			UpdateFunc: n.watchChildrenPhase,
			DeleteFunc: n.watchChildrenPhase,
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

	logger logr.Logger
}

func (lc *ManagedLifecycle) Run() error {
	if err := lc.watchChildren(); err != nil {
		return errors.Wrapf(err, "unable to watch children")
	}

	return nil
}

// Update run in a loops and continuously Update the status of the parent.
// When using this function, there are two rules that must be respected in order to avoid conflicts
// 1) This method should not be followed by status updates (e.g., Pending, Success).
// 2) When deleting the parent object, use the provided Delete() method of this package.
func (lc *ManagedLifecycle) Update(parent InnerObject) error {
	if err := lc.Run(); err != nil {
		return errors.Wrapf(err, "run failed")
	}

	var phase v1alpha1.Phase

	var msg error

	var valid error

	go func() {
		phase, valid = lc.notifier.getNextPhase()

		for {
			if valid != nil {
				lc.logger.Error(valid, "Update parent failed", "parent", parent.GetName())

				return
			}

			lc.logger.Info("Update Parent",
				"kind", reflect.TypeOf(parent),
				"name", parent.GetName(),
				"phase", phase,
			)

			switch phase {
			case v1alpha1.PhaseUninitialized, v1alpha1.PhaseDiscoverable, v1alpha1.PhasePending:
				panic("this should not happen")

			case v1alpha1.PhaseRunning:
				_, _ = Running(lc.ctx, parent, "all children are running")
				phase, msg, valid = lc.notifier.getNextRunning()

			case v1alpha1.PhaseChaos:
				_, _ = Chaos(lc.ctx, parent, "at least one of the children is experiencing chaos")

				// If the parent is in PhaseChaos mode, it means that failing conditions are expected and therefore
				// do not cause a failure to the controller. Instead, they should be handled by the system under evaluation.
				lc.logger.Info("Expected abnormal event",
					"parent", parent.GetName(),
					"event", msg.Error(),
				)

				phase, msg, valid = lc.notifier.getNextChaos()

			case v1alpha1.PhaseSuccess:
				_, _ = Success(lc.ctx, parent, "all children are complete")

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
	if err := lc.Run(); err != nil {
		return errors.Wrapf(err, "run failed")
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

	informer, err := common.Common.Cache.GetInformer(ctx, child)
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

// Delete is a wrapper that addresses a circular dependency issue with the lifecycle monitoring.
// By default, Kubernetes deletes Children before the parent. When a Child is removed,
// the lifecycle watchdog detects that a child is deleted (failed) and updates the parent. However,
// the parent used in the lifecycle is a stalled copy of the actual parent object. Hence, the update
// causes a conflict between the stalled and the actual object.
//
// This deletion method addresses this issue by first deleting the parent, and then the children.
func Delete(ctx context.Context, c client.Client, obj client.Object) error {
	// There are three different options for the deletion propagation policy:
	//
	//    Foreground: Children are deleted before the parent (post-order)
	//    Background: Parent is deleted before the children (pre-order)
	//    Orphan: Owner references are ignored
	deletePolicy := metav1.DeletePropagationBackground

	if err := c.Delete(ctx, obj, &client.DeleteOptions{PropagationPolicy: &deletePolicy}); err != nil {
		return errors.Wrapf(err, "unable to delete object %s", obj.GetName())
	}

	return nil
}
