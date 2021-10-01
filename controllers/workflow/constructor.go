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

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (r *Controller) runJob(ctx context.Context, w *v1alpha1.Workflow, action v1alpha1.Action) error {
	r.Logger.Info("Exec Action", "type", action.ActionType, "name", action.Name, "depends", action.DependsOn)

	logrus.Warn("Handle job ", action.Name)

	switch action.ActionType {
	case "Wait": // expect command will block the entire controller
		return r.wait(ctx, action, *action.Wait)

	case "Service":
		return r.service(ctx, w, action)

	case "Cluster":
		return r.cluster(ctx, w, action)

	case "Stop":
		return r.stop(ctx, w, action)

	case "Chaos":
		return r.chaos(ctx, w, action)

	default:
		return errors.Errorf("unknown action %s", action.ActionType)
	}

}

func (r *Controller) wait(ctx context.Context, action v1alpha1.Action, spec v1alpha1.WaitSpec) error {

	/*
		if len(spec.Running) > 0 {
			logrus.Warnf("-> Action %s waiting for running of %v", action.Name, spec.Running)

			for _, waitFor := range spec.Running {
				lf, exists := r.state.activeJobs[waitFor]
				if !exists {

					// the phase has not been reached yet.
					return true, nil
				}

				switch {
				case lf.Phase.Equals(v1alpha1.PhaseRunning):
					// reached the desired phase
					continue

				case lf.Phase.IsValid(v1alpha1.PhaseRunning):
					// the phase has not been reached yet.
					return true, nil

				default:
					return false, errors.Errorf("phase violation [%s] <- [%s] <- [%s]",
						action.Name,
						lf.Phase,
						waitFor,
					)
				}
			}

			logrus.Warnf("<- Action %s waiting for running of %v", action.Name, spec.Running)
		}

		if len(spec.Success) > 0 {
			logrus.Warnf("-> Action %s waiting for Success of %v", action.Name, spec.Success)

			for _, waitFor := range spec.Success {
				lf, exists := r.state.successfulJobs[waitFor]
				if !exists {
					// the phase has not been reached yet.
					return true, nil
				}

				switch {
				case lf.Phase.Equals(v1alpha1.PhaseSuccess):
					// reached the desired phase
					continue

				case lf.Phase.IsValid(v1alpha1.PhaseSuccess):
					// the phase has not been reached yet.
					return true, nil

				default:
					return false, errors.Errorf("phase violation [%s] <- [%s] <- [%s]",
						action.Name,
						lf.Phase,
						waitFor,
					)
				}
			}

			logrus.Warnf("<- Action %s waiting for Success of %v", action.Name, spec.Success)
		}

		/*
			if spec.Duration != nil {
				logrus.Warnf("-> Action %s waiting for duration of %v", action.Name, spec.Duration.Duration.IsBefore())

				select {
				case <-ctx.Done():
					return errors.Wrapf(ctx.Err(), "wait error")
				case <-time.After(spec.Duration.Duration):
				}

				logrus.Warnf("<- Action %s waiting for duration of %v", action.Name, spec.Duration.Duration.IsBefore())
			}

	*/

	return nil
}

func (r *Controller) service(ctx context.Context, w *v1alpha1.Workflow, action v1alpha1.Action) error {
	var service v1alpha1.Service

	utils.SetOwner(r, w, &service)
	service.SetName(action.Name)

	action.Service.DeepCopyInto(&service.Spec)

	if err := utils.CreateUnlessExists(ctx, r, &service); err != nil {
		return errors.Wrapf(err, "action %s execution failed", action.Name)
	}

	return nil
}

func (r *Controller) cluster(ctx context.Context, w *v1alpha1.Workflow, action v1alpha1.Action) error {
	var cluster v1alpha1.Cluster

	utils.SetOwner(r, w, &cluster)
	cluster.SetName(action.Name)

	action.Cluster.DeepCopyInto(&cluster.Spec)

	if err := utils.CreateUnlessExists(ctx, r, &cluster); err != nil {
		return errors.Wrapf(err, "action %s execution failed", action.Name)
	}

	return nil
}

func (r *Controller) chaos(ctx context.Context, w *v1alpha1.Workflow, action v1alpha1.Action) error {
	var chaos v1alpha1.Chaos

	utils.SetOwner(r, w, &chaos)
	chaos.SetName(action.Name)

	action.Chaos.DeepCopyInto(&chaos.Spec)

	if err := utils.CreateUnlessExists(ctx, r, &chaos); err != nil {
		return errors.Wrapf(err, "action %s execution failed", action.Name)
	}

	return nil
}

func (r *Controller) stop(ctx context.Context, w *v1alpha1.Workflow, action v1alpha1.Action) error {

	panic("STOP IS NOT SUPPORTED")

	/*
		// Resolve affected services
		services := service.Select(ctx, action.Stop.Selector)
		if len(services) == 0 {
			r.Logger.Info("no services to stop", "action", action.Name)

			return nil
		}

		// Without Schedule
		if action.Stop.Schedule == nil {
			for _, instance := range services {
				discovery := corev1.Pod{}

				discovery.SetNamespace(instance.Namespace)
				discovery.SetName(instance.Name)

				logrus.Warn("DELETE ")

				err := lifecycle.Delete(ctx, r.Client, &discovery)
				if err != nil && !k8errors.IsNotFound(err) {
					return errors.Wrapf(err, "cannot delete instance %s", instance.NamespacedName)
				}

				r.Logger.Info("stop", "instance", instance.NamespacedName)
			}
		} else { // With Schedule
			r.Logger.Info("Yield with Schedule", "services", services)

			for instance := range common.YieldByTime(ctx, action.Stop.Schedule.Cron, services...) {
				discovery := corev1.Pod{}

				discovery.SetNamespace(instance.Namespace)
				discovery.SetName(instance.Name)

				err := lifecycle.Delete(ctx, r.Client, &discovery)
				if err != nil && !k8errors.IsNotFound(err) {
					return errors.Wrapf(err, "cannot delete instance %s", instance.NamespacedName)
				}

				r.Logger.Info("stop", "instance", instance)
			}
		}

	*/

	// return nil
}
