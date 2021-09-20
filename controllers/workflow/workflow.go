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

package workflow

import (
	"context"
	"time"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/controllers/common/lifecycle"
	"github.com/fnikolai/frisbee/controllers/common/selector/service"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
)

type Workflow struct {
	*v1alpha1.Workflow

	waitableActions map[string]lifecycle.InnerObject
}

func (r *Reconciler) scheduleActions(topCtx context.Context, obj *v1alpha1.Workflow) {
	ctx, cancel := context.WithCancel(topCtx)
	defer cancel()

	// keep an index of names and objects. This is used for wait to identify the type of object to wait for.
	w := Workflow{
		Workflow:        obj,
		waitableActions: make(map[string]lifecycle.InnerObject),
	}

	var err error

	for _, action := range obj.Spec.Actions {
		r.Logger.Info("Exec Action", "type", action.ActionType, "name", action.Name, "depends", action.Depends)

		switch action.ActionType {
		case "Wait": // expect command will block the entire controller
			err = r.wait(ctx, w, action, *action.Wait)

		case "DistributedGroup":
			err = r.distributedGroup(ctx, w, action)

		case "CollocatedGroup":
			err = r.collocatedGroup(ctx, w, action)

		case "Stop":
			err = r.stop(ctx, w, action)

		case "Chaos":
			err = r.chaos(ctx, w, action)

		default:
			err = errors.Errorf("unknown action %s", action.ActionType)
		}

		if err != nil {
			_, _ = lifecycle.Failed(ctx, w.Workflow, errors.Wrapf(err, "action %s failed", action.Name))

			return
		}
	}

	_, _ = lifecycle.Success(ctx, w.Workflow, "all actions are complete")
}

func (r *Reconciler) wait(ctx context.Context, w Workflow, action v1alpha1.Action, spec v1alpha1.WaitSpec) error {
	if len(spec.Success) > 0 {
		logrus.Warnf("-> Action %s waiting for success of %v", action.Name, spec.Success)

		// confirm that the referenced action have already happened. otherwise, it is possible to block forever.
		for _, waitFor := range spec.Success {
			_, ok := w.waitableActions[waitFor]
			if !ok {
				return errors.Errorf("action %s has not happened yet", spec.Success[0])
			}
		}

		// assume that all action to wait are of the same type
		kind := w.waitableActions[spec.Success[0]]

		err := lifecycle.New(
			lifecycle.Watch(kind, spec.Success...),
			lifecycle.WithFilters(lifecycle.FilterByParent(w), lifecycle.FilterByNames(spec.Success...)),
			lifecycle.WithLogger(r.Logger),
			lifecycle.WithExpectedPhase(v1alpha1.PhaseSuccess),
		).Run(ctx)
		if err != nil {
			return errors.Wrapf(err, "wait error")
		}

		logrus.Warnf("<- Action %s waiting for success of %v", action.Name, spec.Success)
	}

	if len(spec.Running) > 0 {
		logrus.Warnf("-> Action %s waiting for running of %v", action.Name, spec.Running)

		// confirm that the referenced action have already happened. otherwise, it is possible to block forever.
		for _, waitFor := range spec.Running {
			_, ok := w.waitableActions[waitFor]
			if !ok {
				return errors.Errorf("action %s has not happened yet", spec.Success[0])
			}
		}

		// assume that all action to wait are of the same type
		kind := w.waitableActions[spec.Running[0]]

		err := lifecycle.New(
			lifecycle.Watch(kind, spec.Running...),
			lifecycle.WithFilters(lifecycle.FilterByParent(w), lifecycle.FilterByNames(spec.Running...)),
			lifecycle.WithLogger(r.Logger),
			lifecycle.WithExpectedPhase(v1alpha1.PhaseRunning),
		).Run(ctx)
		if err != nil {
			return errors.Wrapf(err, "wait error")
		}

		logrus.Warnf("<- Action %s waiting for running of %v", action.Name, spec.Running)
	}

	if spec.Duration != nil {
		logrus.Warnf("-> Action %s waiting for duration of %v", action.Name, spec.Duration.Duration.String())

		select {
		case <-ctx.Done():
			return errors.Wrapf(ctx.Err(), "wait error")
		case <-time.After(spec.Duration.Duration):
		}

		logrus.Warnf("<- Action %s waiting for duration of %v", action.Name, spec.Duration.Duration.String())
	}

	return nil
}

func (r *Reconciler) distributedGroup(ctx context.Context, w Workflow, action v1alpha1.Action) error {
	if action.Depends != nil {
		if err := r.wait(ctx, w, action, *action.Depends); err != nil {
			return errors.Wrapf(err, "dependencies failed")
		}
	}

	group := v1alpha1.DistributedGroup{}
	group.SetName(action.Name)
	action.DistributedGroup.DeepCopyInto(&group.Spec)

	if err := common.SetOwner(w.Workflow, &group); err != nil {
		return errors.Wrapf(err, "ownership failed")
	}

	if err := r.Client.Create(ctx, &group); err != nil {
		return errors.Wrapf(err, "create failed")
	}

	// TODO: Fix it with respect to threads
	// common.Update(ctx, w, &v1alpha1.DistributedGroup{}, group.GetName())

	w.waitableActions[action.Name] = &group

	return nil
}

func (r *Reconciler) collocatedGroup(ctx context.Context, w Workflow, action v1alpha1.Action) error {
	if action.Depends != nil {
		if err := r.wait(ctx, w, action, *action.Depends); err != nil {
			return errors.Wrapf(err, "dependencies failed")
		}
	}

	group := v1alpha1.CollocatedGroup{}
	group.SetName(action.Name)
	action.CollocatedGroup.DeepCopyInto(&group.Spec)

	if err := common.SetOwner(w.Workflow, &group); err != nil {
		return errors.Wrapf(err, "ownership failed")
	}

	if err := r.Client.Create(ctx, &group); err != nil {
		return errors.Wrapf(err, "create failed")
	}

	// TODO: Fix it with respect to threads
	// common.Update(ctx, w, &v1alpha1.CollocatedGroup{}, group.GetName())

	w.waitableActions[action.Name] = &group

	return nil
}

func (r *Reconciler) stop(ctx context.Context, w Workflow, action v1alpha1.Action) error {
	if action.Depends != nil {
		if err := r.wait(ctx, w, action, *action.Depends); err != nil {
			return errors.Wrapf(err, "dependencies failed")
		}
	}

	// Resolve affected services
	services := service.Select(ctx, action.Stop.Selector)
	if len(services) == 0 {
		r.Logger.Info("no services to stop", "action", action.Name)

		return nil
	}

	// Without Schedule
	if action.Stop.Schedule == nil {
		for _, service := range services {
			discovery := corev1.Pod{}

			discovery.SetNamespace(service.Namespace)
			discovery.SetName(service.Name)

			logrus.Warn("DELETE ")

			err := lifecycle.Delete(ctx, r.Client, &discovery)
			if err != nil && !k8errors.IsNotFound(err) {
				return errors.Wrapf(err, "cannot delete service %s", service.NamespacedName)
			}

			r.Logger.Info("stop", "service", service.NamespacedName)
		}
	} else { // With Schedule
		r.Logger.Info("Yield with Schedule", "services", services)

		for service := range common.YieldByTime(ctx, action.Stop.Schedule.Cron, services...) {
			discovery := corev1.Pod{}

			discovery.SetNamespace(service.Namespace)
			discovery.SetName(service.Name)

			err := lifecycle.Delete(ctx, r.Client, &discovery)
			if err != nil && !k8errors.IsNotFound(err) {
				return errors.Wrapf(err, "cannot delete service %s", service.NamespacedName)
			}

			r.Logger.Info("stop", "service", service)
		}
	}

	return nil
}

func (r *Reconciler) chaos(ctx context.Context, w Workflow, action v1alpha1.Action) error {
	chaos := v1alpha1.Chaos{}
	chaos.SetName(action.Name)
	action.Chaos.DeepCopyInto(&chaos.Spec)

	if err := common.SetOwner(w.Workflow, &chaos); err != nil {
		return errors.Wrapf(err, "ownership failed")
	}

	if action.Depends != nil {
		if err := r.wait(ctx, w, action, *action.Depends); err != nil {
			return errors.Wrapf(err, "dependencies failed")
		}
	}

	if err := r.Client.Create(ctx, &chaos); err != nil {
		return errors.Wrapf(err, "chaos injection failed")
	}

	w.waitableActions[action.Name] = &chaos

	return nil
}
