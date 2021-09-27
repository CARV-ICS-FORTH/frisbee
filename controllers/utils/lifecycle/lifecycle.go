// Licensed to FORTH/ICS under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. FORTH/ICS licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package lifecycle

import (
	"context"
	"reflect"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/utils"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cachetypes "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// InnerObject is an object that is managed and recognized by Frisbee (including services, servicegroups, ...)
type InnerObject interface {
	client.Object
	GetLifecycle() []*v1alpha1.Lifecycle
	SetLifecycle(v1alpha1.Lifecycle)
}

/******************************************************
			Lifecycle Manager
******************************************************/

// Options is a set of options that will be used to instantiate a new Watchdog
type Options struct {
	// Filters is used to filter out events before they reach the lifecycle handler
	Filters []FilterFunc

	// WatchType indicate the type of objects to watch
	WatchType InnerObject

	// ChildrenNames is a list of children names to watch
	ChildrenNames []string

	// Annotator indicate whether events shall be recorded in Grafana or not
	Annotator Annotator

	// Logger is a logger for recording asynchronous operations
	Logger logr.Logger

	// ExpectedPhase indicate the next expected phase
	ExpectedPhase v1alpha1.Phase

	// UpdateParentObjectStatus will update the parent object
	UpdateParentObjectStatus InnerObject
}

// Option is a wrapper for Functional Options.
type Option func(*Options)

// Watch listens of events that happen on object of the given type and names.
func Watch(kind InnerObject, names ...string) Option {
	return func(s *Options) {
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

type StatusAccessor func(obj interface{}) []*v1alpha1.Lifecycle

// WatchExternal wraps an object that does not belong to this controller.
func WatchExternal(kind client.Object, convertor StatusAccessor, names ...string) Option {
	if kind == nil || convertor == nil || len(names) == 0 {
		panic(errors.Errorf("invalid arguments. kind:%s convertor:%p names:%v", reflect.TypeOf(kind), convertor, names))
	}

	return func(s *Options) {
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

// WithFilters add a filter that allows only specific object to reach function like Expect and Wait.
// Most commonly, this is used in conjunction with ParentFilter that filters only events that belongs
// to the given parent.
func WithFilters(filters ...FilterFunc) Option {
	return func(s *Options) {
		if len(filters) == 0 {
			panic("empty filter list")
		}

		if s.Filters != nil {
			panic("filterlist already exists")
		}

		s.Filters = filters
	}
}

// WithAnnotator will send annotations to grafana whenever an object is added or delete.
func WithAnnotator(annotator Annotator) Option {
	return func(s *Options) {
		if annotator == nil {
			panic("empty annotator")
		}

		s.Annotator = annotator
	}
}

// WithLogger appends a new logger for the lifecycle.
func WithLogger(logger logr.Logger) Option {
	return func(s *Options) {
		if s.Logger != nil {
			panic("logger already exists")
		}

		s.Logger = logger
	}
}

// WithExpectedPhase blocks waiting for the next Phase of the parent. If it is the one expected, it returns nil.
// Otherwise, it returns an error stating what was expected and what was got.
func WithExpectedPhase(expected v1alpha1.Phase) Option {
	return func(s *Options) {
		s.ExpectedPhase = expected
	}
}

// WithUpdateParentStatus ...

type ManagedLifecycle struct {
	options Options

	notifier *BoundedFSM

	logger logr.Logger

	filterFunc func(obj interface{}) bool

	eventHandlers eventHandlerFuncs

	expectPhase *v1alpha1.Phase

	parentObject InnerObject
}

func New(opts ...Option) *ManagedLifecycle {
	options := Options{}

	// call option functions on instance to set options on it
	for _, apply := range opts {
		apply(&options)
	}

	// if some stuff
	lifecycle := ManagedLifecycle{
		options:      options,
		logger:       options.Logger,
		parentObject: options.UpdateParentObjectStatus,
	}

	// validate fields
	if lifecycle.logger == nil {
		lifecycle.logger = logr.Discard()
	}

	// if there are no filters, this function will act as passthrough
	lifecycle.filterFunc = func(obj interface{}) bool {
		for _, f := range options.Filters {
			if !f(obj) {
				return false
			}
		}

		return true
	}

	if options.ExpectedPhase != v1alpha1.PhaseUninitialized {
		if lifecycle.parentObject != nil {
			panic("expected conflicts with update parent")
		}

		lifecycle.expectPhase = &options.ExpectedPhase
	}

	var fsm *BoundedFSM

	// set children
	if len(options.ChildrenNames) == 0 {
		panic("empty children")
	} else {
		fsm = NewBoundedFSM(options.ChildrenNames)
		fsm.logger = lifecycle.logger
		fsm.annotaror = options.Annotator
	}

	// set watchtype
	if options.WatchType == nil {
		panic("empty watchtype")
	} else {
		fsm.getChildLifecycle = accessStatus(options.WatchType)
	}

	if options.Annotator != nil {
		lifecycle.eventHandlers = eventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				options.Annotator.Add(obj.(client.Object))

				fsm.HandleEvent(obj)
			},
			UpdateFunc: fsm.HandleEvent,
			DeleteFunc: func(obj interface{}) {
				logrus.Warn("----------Deleted object ---------")
				options.Annotator.Delete(obj.(client.Object))

				fsm.HandleEvent(obj)
			},
		}
	} else {
		lifecycle.eventHandlers = eventHandlerFuncs{
			AddFunc:    fsm.HandleEvent,
			UpdateFunc: fsm.HandleEvent,
			DeleteFunc: fsm.HandleEvent,
		}
	}

	lifecycle.notifier = fsm

	return &lifecycle
}

type eventHandlerFuncs struct {
	AddFunc    func(obj interface{})
	UpdateFunc func(obj interface{})
	DeleteFunc func(obj interface{})
}

func (lc *ManagedLifecycle) Run(ctx context.Context, r utils.Reconciler) error {
	informer, err := utils.Globals.Cache.GetInformer(ctx, unwrap(lc.options.WatchType))
	if err != nil {
		return errors.Wrapf(err, "unable to get informer for %s", unwrap(lc.options.WatchType).GetName())
	}

	// Set up an event handler for when ByCluster resources change. This
	// handler will check the owner of the given ByCluster, and if it is
	// owned by a Foo (ByCluster) resource will enqueue that Foo resource for
	// processing. This way, we don't need to implement custom logic for
	// handling ByCluster resources. More info on this pattern:
	// https://github.com/kubernetes/community/blob/8cafef897a22026d42f5e5bb3f104febe7e29830/contributors/devel/controllers.md
	informer.AddEventHandler(cachetypes.FilteringResourceEventHandler{
		FilterFunc: lc.filterFunc,
		Handler: cachetypes.ResourceEventHandlerFuncs{
			AddFunc: lc.eventHandlers.AddFunc,
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

				lc.eventHandlers.UpdateFunc(latest)
			},
			DeleteFunc: lc.eventHandlers.DeleteFunc,
		},
	})

	if lc.expectPhase != nil {
		return lc.notifier.Expect(*lc.expectPhase)
	}

	if lc.parentObject != nil {
		return lc.notifier.Update(ctx, r, lc.parentObject)
	}

	return nil
}
