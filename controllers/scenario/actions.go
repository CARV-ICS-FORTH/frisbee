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

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	chaosutils "github.com/carv-ics-forth/frisbee/controllers/chaos/utils"
	serviceutils "github.com/carv-ics-forth/frisbee/controllers/service/utils"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type endpoint func(context.Context, *v1alpha1.Scenario, v1alpha1.Action) (client.Object, error)

func (r *Controller) supportedActions() map[v1alpha1.ActionType]endpoint {
	return map[v1alpha1.ActionType]endpoint{
		v1alpha1.ActionService: r.service,
		v1alpha1.ActionCluster: r.cluster,
		v1alpha1.ActionChaos:   r.chaos,
		v1alpha1.ActionCascade: r.cascade,
		v1alpha1.ActionDelete:  r.delete,
		v1alpha1.ActionCall:    r.call,
	}
}

func (r *Controller) service(ctx context.Context, t *v1alpha1.Scenario, action v1alpha1.Action) (client.Object, error) {
	if err := expandMapInputs(ctx, r, t.GetNamespace(), &action.Service.Inputs); err != nil {
		return nil, errors.Wrapf(err, "input error")
	}

	// get the job template
	if err := action.Service.Prepare(false); err != nil {
		return nil, errors.Wrapf(err, "template validation")
	}

	spec, err := serviceutils.GetServiceSpec(ctx, r.GetClient(), t, *action.Service)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot retrieve job spec")
	}

	var job v1alpha1.Service

	job.SetGroupVersionKind(v1alpha1.GroupVersion.WithKind("Service"))
	job.SetNamespace(t.GetNamespace())
	job.SetName(action.Name)

	// set labels
	v1alpha1.SetScenario(&job.ObjectMeta, t.GetName())
	v1alpha1.SetAction(&job.ObjectMeta, action.Name)

	// The job belongs to a SUT, unless the template is explicitly declared as a System job (SYS)
	if spec.Decorators != nil && spec.Decorators.Labels != nil &&
		spec.Decorators.Labels[v1alpha1.LabelComponent] == string(v1alpha1.ComponentSys) {
		v1alpha1.SetComponent(&job.ObjectMeta, v1alpha1.ComponentSys)
	} else {
		v1alpha1.SetComponent(&job.ObjectMeta, v1alpha1.ComponentSUT)
	}

	spec.DeepCopyInto(&job.Spec)

	return &job, nil
}

func (r *Controller) cluster(ctx context.Context, t *v1alpha1.Scenario, action v1alpha1.Action) (client.Object, error) {
	if err := expandMapInputs(ctx, r, t.GetNamespace(), &action.Cluster.Inputs); err != nil {
		return nil, errors.Wrapf(err, "input error")
	}

	var job v1alpha1.Cluster

	job.SetGroupVersionKind(v1alpha1.GroupVersion.WithKind("Cluster"))
	job.SetNamespace(t.GetNamespace())
	job.SetName(action.Name)

	// set labels
	v1alpha1.SetScenario(&job.ObjectMeta, t.GetName())
	v1alpha1.SetAction(&job.ObjectMeta, action.Name)
	v1alpha1.SetComponent(&job.ObjectMeta, v1alpha1.ComponentSUT)

	action.Cluster.DeepCopyInto(&job.Spec)

	return &job, nil
}

func (r *Controller) chaos(ctx context.Context, t *v1alpha1.Scenario, action v1alpha1.Action) (client.Object, error) {
	if err := expandMapInputs(ctx, r, t.GetNamespace(), &action.Chaos.Inputs); err != nil {
		return nil, errors.Wrapf(err, "input error")
	}

	// get the service template
	if err := action.Chaos.Prepare(false); err != nil {
		return nil, errors.Wrapf(err, "template validation")
	}

	spec, err := chaosutils.GetChaosSpec(ctx, r.GetClient(), t, *action.Chaos)
	if err != nil {
		return nil, errors.Wrapf(err, "service spec")
	}

	var job v1alpha1.Chaos

	job.SetGroupVersionKind(v1alpha1.GroupVersion.WithKind("Chaos"))
	job.SetNamespace(t.GetNamespace())
	job.SetName(action.Name)

	v1alpha1.SetScenario(&job.ObjectMeta, t.GetName())
	v1alpha1.SetAction(&job.ObjectMeta, action.Name)
	v1alpha1.SetComponent(&job.ObjectMeta, v1alpha1.ComponentSUT)

	spec.DeepCopyInto(&job.Spec)

	return &job, nil
}

func (r *Controller) cascade(ctx context.Context, t *v1alpha1.Scenario, action v1alpha1.Action) (client.Object, error) {
	if err := expandMapInputs(ctx, r, t.GetNamespace(), &action.Cascade.Inputs); err != nil {
		return nil, errors.Wrapf(err, "input error")
	}

	var job v1alpha1.Cascade

	job.SetGroupVersionKind(v1alpha1.GroupVersion.WithKind("Cascade"))
	job.SetNamespace(t.GetNamespace())
	job.SetName(action.Name)

	v1alpha1.SetScenario(&job.ObjectMeta, t.GetName())
	v1alpha1.SetAction(&job.ObjectMeta, action.Name)
	v1alpha1.SetComponent(&job.ObjectMeta, v1alpha1.ComponentSUT)

	action.Cascade.DeepCopyInto(&job.Spec)

	return &job, nil
}

func (r *Controller) delete(ctx context.Context, t *v1alpha1.Scenario, action v1alpha1.Action) (client.Object, error) {
	// Delete normally does not return anything. This however would break all the pipeline for
	// managing dependencies between jobs. For that, we return a dummy virtual object without dedicated controller.
	var job v1alpha1.VirtualObject

	job.SetGroupVersionKind(v1alpha1.GroupVersion.WithKind("VirtualObject"))
	job.SetNamespace(t.GetNamespace())
	job.SetName(action.Name)

	v1alpha1.SetScenario(&job.ObjectMeta, t.GetName())
	v1alpha1.SetAction(&job.ObjectMeta, action.Name)
	v1alpha1.SetComponent(&job.ObjectMeta, v1alpha1.ComponentSUT)

	job.SetReconcileStatus(v1alpha1.Lifecycle{
		Phase:   v1alpha1.PhaseSuccess,
		Reason:  "AllJobsDeleted",
		Message: "",
	})

	// delete the jobs in foreground -- jobs are deleted before the function returns.
	propagation := metav1.DeletePropagationForeground
	options := client.DeleteOptions{
		PropagationPolicy: &propagation,
	}

	for _, name := range action.Delete.Jobs {
		job, deletable := r.clusterView.IsDeletable(name)
		if !deletable {
			return nil, errors.Errorf("job %s is not currently deletable", name)
		}

		if err := r.GetClient().Delete(ctx, job, &options); err != nil {
			return nil, errors.Wrapf(err, "unable to delete job %s", job.GetName())
		}
	}

	return &job, nil
}

func (r *Controller) call(ctx context.Context, t *v1alpha1.Scenario, action v1alpha1.Action) (client.Object, error) {
	if err := expandSliceInputs(ctx, r, t.GetNamespace(), &action.Call.Services); err != nil {
		return nil, errors.Wrapf(err, "input error")
	}

	var job v1alpha1.Call

	job.SetGroupVersionKind(v1alpha1.GroupVersion.WithKind("Call"))
	job.SetNamespace(t.GetNamespace())
	job.SetName(action.Name)

	v1alpha1.SetScenario(&job.ObjectMeta, t.GetName())
	v1alpha1.SetAction(&job.ObjectMeta, action.Name)
	v1alpha1.SetComponent(&job.ObjectMeta, v1alpha1.ComponentSUT)

	action.Call.DeepCopyInto(&job.Spec)

	return &job, nil
}
