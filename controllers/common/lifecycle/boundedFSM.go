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
	"github.com/fnikolai/frisbee/pkg/wait"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// BoundedFSM calculates the present state of the parent by using a set of predefined children.
// A BoundedFSM is useful in group where the parent becomes running only if all the (predefined) children are running.
type BoundedFSM struct {
	annotaror Annotator

	logger logr.Logger

	getChildLifecycle StatusAccessor

	parentRunning   chan struct{}
	childrenRunning map[string]chan struct{}

	parentComplete  chan struct{}
	childrenSuccess map[string]chan struct{}

	chaos  chan error
	failed chan error
}

func NewBoundedFSM(serviceNames []string) *BoundedFSM {
	if len(serviceNames) == 0 {
		return &BoundedFSM{}
	}

	parentRun, childrenRun := wait.ChannelWaitForChildren(len(serviceNames))
	parentExit, childrenExit := wait.ChannelWaitForChildren(len(serviceNames))

	t := &BoundedFSM{
		parentRunning:   parentRun,
		childrenRunning: make(map[string]chan struct{}),
		parentComplete:  parentExit,
		childrenSuccess: make(map[string]chan struct{}),
		failed:          make(chan error),
	}

	for i, name := range serviceNames {
		t.childrenRunning[name] = childrenRun[i]
		t.childrenSuccess[name] = childrenExit[i]
	}

	return t
}

// HandleEvent monitors the Phase of children and Update the channels of the parent.
// To deliver idempotent operation, we want to ensure exactly-once notifications. However, an object may
// be in the same Phase during different iterations. For this Reason, we close the channel only the first
// and then we nil it. If the second iterations finds a nil for the given child,
// it returns immediately as it assumes that the closing is already delivered.
func (n *BoundedFSM) HandleEvent(obj interface{}) {

	for _, event := range n.getChildLifecycle(obj) {
		if event == nil || *event == (v1alpha1.Lifecycle{}) { // ignore empty events (e.g, add)
			continue
		}

		if event.Name == "" || event.Kind == "" {
			panic(errors.Errorf("controller panic due to invalid args. name: %s, kind:%s", event.Name, event.Kind))
		}

		logrus.Warn("Handle Event ", event)

		switch event.Phase {
		case v1alpha1.PhaseUninitialized, v1alpha1.PhasePending, v1alpha1.PhaseChaos:
			return

		case v1alpha1.PhaseRunning:
			ch, ok := n.childrenRunning[event.Name]
			if !ok {
				panic(errors.Errorf("unexpected event %s -- expected list %v", event.String(), n.childrenRunning))
			}

			if ch != nil {
				n.childrenRunning[event.Name] = nil

				close(ch)
			}

		case v1alpha1.PhaseSuccess:
			ch, ok := n.childrenSuccess[event.Name]
			if !ok {
				panic(errors.Errorf("unexpected event %s -- expected list %v", event.String(), n.childrenSuccess))
			}

			if ch != nil {
				n.childrenSuccess[event.Name] = nil

				close(ch)
			}

		case v1alpha1.PhaseFailed:
			// stupid thing but we need for Kubernetes v.1.19.
			// In v.1.21 this is captured by the DeleteFunc of the watcher
			if event.EndTime != nil {
				n.annotaror.Delete(obj)
			}

			n.failed <- errors.New(event.String())
		}
	}
}

// getNextPhase blocks waiting for the next of the parent.
func (n *BoundedFSM) getNextPhase() (v1alpha1.Phase, error) {
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
func (n *BoundedFSM) getNextRunning() (phase v1alpha1.Phase, msg, valid error) {
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

var errIsFinal = errors.New("Phase is final")

// listen for all the expected transition from PhaseSuccess.
func (n *BoundedFSM) getNextSuccess() (phase v1alpha1.Phase, msg, valid error) {
	return v1alpha1.PhaseSuccess, nil, errIsFinal
}

// listen for all the expected transition from Failed.
func (n *BoundedFSM) getNextFailed() (phase v1alpha1.Phase, msg, valid error) {
	return v1alpha1.PhaseFailed, nil, errIsFinal
}

// listen for all the expected transition from Chaos.
func (n *BoundedFSM) getNextChaos() (phase v1alpha1.Phase, msg, valid error) {
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

func (n *BoundedFSM) Expect(expected v1alpha1.Phase) error {
	phase, err := n.getNextPhase()
	if err != nil {
		return err
	}

	var msg, valid error

	for {
		switch {
		case phase == expected: // correct Phase
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
		case v1alpha1.PhaseUninitialized, v1alpha1.PhaseChaos, v1alpha1.PhasePending:
			panic(errors.Errorf("cannot wait on Phase %s", expected))

		case v1alpha1.PhaseRunning:
			phase, msg, valid = n.getNextRunning()

		case v1alpha1.PhaseSuccess:
			phase, msg, valid = n.getNextSuccess()

		case v1alpha1.PhaseFailed:
			phase, msg, valid = n.getNextFailed()

		default:
			return errors.Errorf("invalid Phase %s ", phase)
		}
	}
}

// Update run in a loops and continuously updates the lifecycle of the parent. This method, however, does not
// convey object specific status. Only updates the lifecycle.
//
// When using this function, there are two rules that must be respected in order to avoid conflicts
// 1) This method should not be followed by status updates (e.g., Pending, Success).
// 2) When deleting the parent object, use the provided Delete() method of this package.
func (n *BoundedFSM) Update(ctx context.Context, parent InnerObject) error {

	var phase v1alpha1.Phase

	var msg error

	var valid error

	go func() {
		phase, msg = n.getNextPhase()

		for {
			if valid != nil {
				n.logger.Error(valid, "invalid transition", "parent", parent.GetName())

				return
			}

			n.logger.Info("Update Parent",
				"kind", reflect.TypeOf(parent),
				"Name", parent.GetName(),
				"Phase", phase,
			)

			switch phase {
			case v1alpha1.PhaseUninitialized, v1alpha1.PhasePending:
				panic("this should not happen")

			case v1alpha1.PhaseRunning:
				_, _ = Running(ctx, parent, "all children are running")
				phase, msg, valid = n.getNextRunning()

			case v1alpha1.PhaseChaos:
				_, _ = Chaos(ctx, parent, "at least one of the children is experiencing chaos")

				// If the parent is in PhaseChaos mode, it means that failing conditions are expected and therefore
				// do not cause a failure to the controller. Instead, they should be handled by the system under evaluation.
				n.logger.Info("Expected abnormal event",
					"parent", parent.GetName(),
					"event", msg.Error(),
				)

				phase, msg, valid = n.getNextChaos()

			case v1alpha1.PhaseSuccess:
				_, _ = Success(ctx, parent, "all children are complete")

				return

			case v1alpha1.PhaseFailed:
				// TODO: find a more graceful way to propagate failure. Combine it with expression
				_, _ = Failed(ctx, parent, msg)

				return

			default:
				valid = errors.Errorf("invalid Phase %s ", phase)
			}
		}
	}()

	return nil
}
