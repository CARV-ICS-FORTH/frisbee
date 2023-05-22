/*
Copyright 2021-2023 ICS-FORTH.

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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *Controller) RunAction(ctx context.Context, scenario *v1alpha1.Scenario, action v1alpha1.Action) error {
	switch action.ActionType {
	case v1alpha1.ActionService:
		job, err := r.service(ctx, scenario, action)
		if err != nil {
			return errors.Wrapf(err, "preparation of action '%s' has failed", action.Name)
		}

		return common.Create(ctx, r, scenario, job)

	case v1alpha1.ActionCluster:
		job := r.cluster(scenario, action)

		return common.Create(ctx, r, scenario, job)

	case v1alpha1.ActionChaos:
		job, err := r.chaos(ctx, scenario, action)
		if err != nil {
			return errors.Wrapf(err, "preparation of action '%s' has failed", action.Name)
		}

		return common.Create(ctx, r, scenario, job)

	case v1alpha1.ActionCascade:
		job := r.cascade(scenario, action)

		return common.Create(ctx, r, scenario, job)

	case v1alpha1.ActionCall:
		job := r.call(scenario, action)

		return common.Create(ctx, r, scenario, job)

	case v1alpha1.ActionDelete:
		if err := r.delete(ctx, scenario, action); err != nil {
			return errors.Errorf("delete action '%s' has failed", action.Name)
		}

		// Some jobs are virtual and do not require something to be created.
		return nil

	default:
		panic("should never happen")
	}
}

func (r *Controller) service(ctx context.Context, scenario *v1alpha1.Scenario, action v1alpha1.Action) (*v1alpha1.Service, error) {
	// get the job template
	spec, err := serviceutils.GetServiceSpec(ctx, r.GetClient(), scenario, *action.Service)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot retrieve job spec")
	}

	var job v1alpha1.Service

	// Metadata
	job.SetGroupVersionKind(v1alpha1.GroupVersion.WithKind("Service"))
	job.SetNamespace(scenario.GetNamespace())
	job.SetName(action.Name)

	v1alpha1.SetScenarioLabel(&job.ObjectMeta, scenario.GetName())
	v1alpha1.SetActionLabel(&job.ObjectMeta, action.Name)

	// The job belongs to a SUT, unless the template is explicitly declared as a System job (SYS)
	if job.Spec.Decorators.Labels != nil &&
		job.Spec.Decorators.Labels[v1alpha1.LabelComponent] == string(v1alpha1.ComponentSys) {
		v1alpha1.SetComponentLabel(&job.ObjectMeta, v1alpha1.ComponentSys)
	} else {
		v1alpha1.SetComponentLabel(&job.ObjectMeta, v1alpha1.ComponentSUT)
	}

	// Spec
	spec.DeepCopyInto(&job.Spec)

	// Add shared storage
	if scenario.Spec.TestData != nil {
		serviceutils.AttachTestDataVolume(&job, scenario.Spec.TestData, true)
	}

	return &job, nil
}

func (r *Controller) cluster(scenario *v1alpha1.Scenario, action v1alpha1.Action) *v1alpha1.Cluster {
	var job v1alpha1.Cluster

	// Metadata
	job.SetGroupVersionKind(v1alpha1.GroupVersion.WithKind("Cluster"))
	job.SetNamespace(scenario.GetNamespace())
	job.SetName(action.Name)

	v1alpha1.SetScenarioLabel(&job.ObjectMeta, scenario.GetName())
	v1alpha1.SetActionLabel(&job.ObjectMeta, action.Name)
	v1alpha1.SetComponentLabel(&job.ObjectMeta, v1alpha1.ComponentSUT)

	// Spec
	action.Cluster.DeepCopyInto(&job.Spec)

	// Add shared storage
	job.Spec.TestData = scenario.Spec.TestData

	return &job
}

func (r *Controller) chaos(ctx context.Context, scenario *v1alpha1.Scenario, action v1alpha1.Action) (*v1alpha1.Chaos, error) {
	spec, err := chaosutils.GetChaosSpec(ctx, r.GetClient(), scenario, *action.Chaos)
	if err != nil {
		return nil, errors.Wrapf(err, "service spec")
	}

	var job v1alpha1.Chaos

	// Metadata
	job.SetGroupVersionKind(v1alpha1.GroupVersion.WithKind("Chaos"))
	job.SetNamespace(scenario.GetNamespace())
	job.SetName(action.Name)

	v1alpha1.SetScenarioLabel(&job.ObjectMeta, scenario.GetName())
	v1alpha1.SetActionLabel(&job.ObjectMeta, action.Name)
	v1alpha1.SetComponentLabel(&job.ObjectMeta, v1alpha1.ComponentSUT)

	// Spec
	spec.DeepCopyInto(&job.Spec)

	return &job, nil
}

func (r *Controller) cascade(scenario *v1alpha1.Scenario, action v1alpha1.Action) *v1alpha1.Cascade {
	var job v1alpha1.Cascade

	// Metadata
	job.SetGroupVersionKind(v1alpha1.GroupVersion.WithKind("Cascade"))
	job.SetNamespace(scenario.GetNamespace())
	job.SetName(action.Name)

	v1alpha1.SetScenarioLabel(&job.ObjectMeta, scenario.GetName())
	v1alpha1.SetActionLabel(&job.ObjectMeta, action.Name)
	v1alpha1.SetComponentLabel(&job.ObjectMeta, v1alpha1.ComponentSUT)

	// Spec
	action.Cascade.DeepCopyInto(&job.Spec)

	return &job
}

func (r *Controller) call(scenario *v1alpha1.Scenario, action v1alpha1.Action) *v1alpha1.Call {
	var job v1alpha1.Call

	// Metadata
	job.SetGroupVersionKind(v1alpha1.GroupVersion.WithKind("Call"))
	job.SetNamespace(scenario.GetNamespace())
	job.SetName(action.Name)

	v1alpha1.SetScenarioLabel(&job.ObjectMeta, scenario.GetName())
	v1alpha1.SetActionLabel(&job.ObjectMeta, action.Name)
	v1alpha1.SetComponentLabel(&job.ObjectMeta, v1alpha1.ComponentSUT)

	// Spec
	action.Call.DeepCopyInto(&job.Spec)

	return &job
}

func (r *Controller) delete(ctx context.Context, scenario *v1alpha1.Scenario, action v1alpha1.Action) error {
	r.Info("-> Delete", "obj", action.Name, "targets", action.Delete.Jobs)
	defer r.Info("<- Delete", "obj", action.Name, "targets", action.Delete.Jobs)

	// ensure that all references jobs are deletable
	jobsToDelete := make([]client.Object, 0, len(action.Delete.Jobs))

	/*---------------------------------------------------
	 * Filter jobs that are ready to be deleted
	 *---------------------------------------------------*/
	for _, refJob := range action.Delete.Jobs {
		switch {
		case r.view.IsSuccessful(refJob), r.view.IsFailed(refJob):
			r.Logger.Info("job '%s' is already completed.")

			continue

		case r.view.IsPending(refJob):
			job := r.view.GetPendingJobs(refJob)[0]

			if v1alpha1.GetComponentLabel(job) == v1alpha1.ComponentSys {
				return errors.Errorf("service '%s' belongs to the system and is not deletable", refJob)
			}

			jobsToDelete = append(jobsToDelete, job)

		case r.view.IsRunning(refJob):
			job := r.view.GetRunningJobs(refJob)[0]

			if v1alpha1.GetComponentLabel(job) == v1alpha1.ComponentSys {
				return errors.Errorf("service '%s' belongs to the system and is not deletable", refJob)
			}

			jobsToDelete = append(jobsToDelete, job)
		default:
			return errors.Errorf("service '%s' is not yet scheduled. Check your conditions", refJob)
		}
	}

	/*---------------------------------------------------
	 * Delete jobs and replace them with VirtualObjects
	 *---------------------------------------------------*/

	// Context of Delete Action
	//
	// Delete is an action itself and should be waited by the scenario controller.
	// However, since there is no dedicated controller, we need to create a virtual object that represents
	// the Delete action.
	deleteActionName := action.Name
	return lifecycle.CreateVirtualJob(ctx, r, scenario, deleteActionName, func(_ *v1alpha1.VirtualObject) error {
		for i := range jobsToDelete {
			job := jobsToDelete[i]
			// Context of Delete Job
			//
			// Deleting jobs would break the enumeration of completed actions. To avoid it,
			// every time we delete a 'physical' job, we replace it with a 'virtual object' that is
			// simply a dummy entry in the API server.
			// For the entry we use a descriptive name that makes it easy to follow the deletion flow from the cli.
			jobToDelete := fmt.Sprintf("%s-%s", action.Name, job.GetName())

			err := lifecycle.CreateVirtualJob(ctx, r, scenario, jobToDelete, func(_ *v1alpha1.VirtualObject) error {
				common.Delete(ctx, r, job)

				return nil
			})
			if err != nil {
				return errors.Wrapf(err, "Deletion error '%s'", jobToDelete)
			}
		}

		return nil
	})
}
