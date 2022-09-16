/*
Copyright 2021 ICS-FORTH.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package scenario

import (
	"context"
	"fmt"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	chaosutils "github.com/carv-ics-forth/frisbee/controllers/chaos/utils"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	serviceutils "github.com/carv-ics-forth/frisbee/controllers/service/utils"
	"github.com/carv-ics-forth/frisbee/pkg/lifecycle"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *Controller) RunAction(ctx context.Context, scenario *v1alpha1.Scenario, action v1alpha1.Action) error {
	var (
		job client.Object
		err error
	)

	switch action.ActionType {
	case v1alpha1.ActionService:
		job, err = r.service(ctx, scenario, action)
	case v1alpha1.ActionCluster:
		job, err = r.cluster(ctx, scenario, action)
	case v1alpha1.ActionChaos:
		job, err = r.chaos(ctx, scenario, action)
	case v1alpha1.ActionCascade:
		job, err = r.cascade(ctx, scenario, action)
	case v1alpha1.ActionCall:
		job, err = r.call(ctx, scenario, action)
	case v1alpha1.ActionDelete:
		// Some jobs are virtual and do not require something to be created.
		if err := r.delete(ctx, scenario, action); err != nil {
			return errors.Errorf("delete action '%s' has failed", action.Name)
		}

		return nil
	}

	if err != nil {
		return errors.Wrapf(err, "preparation of action '%s' has failed", action.Name)
	}

	return common.Create(ctx, r, scenario, job)
}

func (r *Controller) service(ctx context.Context, scenario *v1alpha1.Scenario, action v1alpha1.Action) (*v1alpha1.Service, error) {
	if err := expandMacros(ctx, r, scenario.GetNamespace(), &action.Service.Inputs); err != nil {
		return nil, errors.Wrapf(err, "input error")
	}

	// get the job template
	if err := action.Service.Prepare(false); err != nil {
		return nil, errors.Wrapf(err, "template validation")
	}

	spec, err := serviceutils.GetServiceSpec(ctx, r.GetClient(), scenario, *action.Service)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot retrieve job spec")
	}

	var job v1alpha1.Service

	job.SetGroupVersionKind(v1alpha1.GroupVersion.WithKind("Service"))
	job.SetNamespace(scenario.GetNamespace())
	job.SetName(action.Name)

	// set labels
	v1alpha1.SetScenarioLabel(&job.ObjectMeta, scenario.GetName())
	spec.DeepCopyInto(&job.Spec)

	// The job belongs to a SUT, unless the template is explicitly declared as a System job (SYS)
	if job.Spec.Decorators.Labels != nil &&
		job.Spec.Decorators.Labels[v1alpha1.LabelComponent] == string(v1alpha1.ComponentSys) {
		v1alpha1.SetComponentLabel(&job.ObjectMeta, v1alpha1.ComponentSys)
	} else {
		v1alpha1.SetComponentLabel(&job.ObjectMeta, v1alpha1.ComponentSUT)
	}

	// Add shared storage
	if scenario.Spec.TestData != nil {
		job.AttachTestDataVolume(scenario.Spec.TestData, true)
	}

	return &job, nil
}

func (r *Controller) cluster(ctx context.Context, scenario *v1alpha1.Scenario, action v1alpha1.Action) (*v1alpha1.Cluster, error) {
	// Validate job requirements
	if action.Cluster.Placement != nil {
		// ensure there are at least two physical nodes for placement to make sense
		var nodes corev1.NodeList

		if err := r.GetClient().List(ctx, &nodes); err != nil {
			return nil, errors.Wrapf(err, "cannot list physical nodes")
		}

		{ // Ensure that there are enough nodes for placement to make sense.
			var ready []string
			var notReady []string

			for _, node := range nodes.Items {
				// search at the node's condition for the "NodeReady".
				for _, cond := range node.Status.Conditions {
					if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
						ready = append(ready, node.GetName())

						goto next
					}
				}
				notReady = append(notReady, node.GetName())
			next:
			}

			if len(ready) < 2 {
				return nil, errors.Errorf("placement policies require at least two ready physical nodes."+
					" Ready:'%v', NotReady:'%v'", ready, notReady)
			}
		}
		// TODO: check compatibility with labels and taints
	}

	// Evaluate macros into concrete statements
	if err := expandMacros(ctx, r, scenario.GetNamespace(), &action.Cluster.Inputs); err != nil {
		return nil, errors.Wrapf(err, "input error")
	}

	var job v1alpha1.Cluster
	// Create job specification
	job.SetGroupVersionKind(v1alpha1.GroupVersion.WithKind("Cluster"))
	job.SetNamespace(scenario.GetNamespace())
	job.SetName(action.Name)

	// set labels
	v1alpha1.SetScenarioLabel(&job.ObjectMeta, scenario.GetName())
	v1alpha1.SetComponentLabel(&job.ObjectMeta, v1alpha1.ComponentSUT)

	action.Cluster.DeepCopyInto(&job.Spec)

	// Add shared storage
	job.Spec.TestData = scenario.Spec.TestData

	return &job, nil
}

func (r *Controller) chaos(ctx context.Context, scenario *v1alpha1.Scenario, action v1alpha1.Action) (*v1alpha1.Chaos, error) {
	if err := expandMacros(ctx, r, scenario.GetNamespace(), &action.Chaos.Inputs); err != nil {
		return nil, errors.Wrapf(err, "input error")
	}

	// get the service template
	if err := action.Chaos.Prepare(false); err != nil {
		return nil, errors.Wrapf(err, "template validation")
	}

	spec, err := chaosutils.GetChaosSpec(ctx, r.GetClient(), scenario, *action.Chaos)
	if err != nil {
		return nil, errors.Wrapf(err, "service spec")
	}

	var job v1alpha1.Chaos

	job.SetGroupVersionKind(v1alpha1.GroupVersion.WithKind("Chaos"))
	job.SetNamespace(scenario.GetNamespace())
	job.SetName(action.Name)

	v1alpha1.SetScenarioLabel(&job.ObjectMeta, scenario.GetName())
	v1alpha1.SetComponentLabel(&job.ObjectMeta, v1alpha1.ComponentSUT)

	spec.DeepCopyInto(&job.Spec)

	return &job, nil
}

func (r *Controller) cascade(ctx context.Context, scenario *v1alpha1.Scenario, action v1alpha1.Action) (*v1alpha1.Cascade, error) {
	if err := expandMacros(ctx, r, scenario.GetNamespace(), &action.Cascade.Inputs); err != nil {
		return nil, errors.Wrapf(err, "input error")
	}

	var job v1alpha1.Cascade

	job.SetGroupVersionKind(v1alpha1.GroupVersion.WithKind("Cascade"))
	job.SetNamespace(scenario.GetNamespace())
	job.SetName(action.Name)

	v1alpha1.SetScenarioLabel(&job.ObjectMeta, scenario.GetName())
	v1alpha1.SetComponentLabel(&job.ObjectMeta, v1alpha1.ComponentSUT)

	action.Cascade.DeepCopyInto(&job.Spec)

	return &job, nil
}

func (r *Controller) delete(ctx context.Context, scenario *v1alpha1.Scenario, action v1alpha1.Action) error {
	r.Info("-> Delete", "obj", action.Name, "targets", action.Delete.Jobs)
	defer r.Info("<- Delete", "obj", action.Name, "targets", action.Delete.Jobs)

	// ensure that all references jobs are deletable
	deletableJobs := make([]client.Object, 0, len(action.Delete.Jobs))

	for _, refJob := range action.Delete.Jobs {
		switch {
		case r.view.IsSuccessful(refJob), r.view.IsFailed(refJob), r.view.IsTerminating(refJob):
			r.Logger.Info("job '%s' is already completed.")

			continue

		case r.view.IsPending(refJob):
			job := r.view.GetPendingJobs(refJob)[0]

			if v1alpha1.GetComponentLabel(job) == v1alpha1.ComponentSys {
				return errors.Errorf("service '%s' belongs to the system and is not deletable", refJob)
			}

			deletableJobs = append(deletableJobs, job)

		case r.view.IsRunning(refJob):
			job := r.view.GetRunningJobs(refJob)[0]

			if v1alpha1.GetComponentLabel(job) == v1alpha1.ComponentSys {
				return errors.Errorf("service '%s' belongs to the system and is not deletable", refJob)
			}

			deletableJobs = append(deletableJobs, job)
		default:
			return errors.Errorf("service '%s' is not yet scheduled. Check your conditions", refJob)
		}
	}

	// Delete normally does not return anything. This however would break all the pipeline for
	// managing dependencies between jobs. For that, we return a dummy virtual object without dedicated controller.
	return lifecycle.VirtualExecution(ctx, r, scenario, action.Name, func(_ *v1alpha1.VirtualObject) error {
		for _, deletableJob := range deletableJobs {
			// a descriptive name makes it easy to follow the deletion flow from the cli.
			deleteJobName := fmt.Sprintf("%s-%s", action.Name, deletableJob.GetName())

			if err := lifecycle.VirtualExecution(ctx, r, scenario, deleteJobName, func(_ *v1alpha1.VirtualObject) error {
				common.Delete(ctx, r, deletableJob)

				return nil
			}); err != nil {
				return errors.Wrapf(err, "virtual-execution error '%s'", deleteJobName)
			}
		}

		return nil
	})
}

func (r *Controller) call(ctx context.Context, scenario *v1alpha1.Scenario, action v1alpha1.Action) (*v1alpha1.Call, error) {
	if err := expandSliceInputs(ctx, r, scenario.GetNamespace(), &action.Call.Services); err != nil {
		return nil, errors.Wrapf(err, "input error")
	}

	var job v1alpha1.Call

	job.SetGroupVersionKind(v1alpha1.GroupVersion.WithKind("Call"))
	job.SetNamespace(scenario.GetNamespace())
	job.SetName(action.Name)

	v1alpha1.SetScenarioLabel(&job.ObjectMeta, scenario.GetName())
	v1alpha1.SetComponentLabel(&job.ObjectMeta, v1alpha1.ComponentSUT)

	action.Call.DeepCopyInto(&job.Spec)

	return &job, nil
}
