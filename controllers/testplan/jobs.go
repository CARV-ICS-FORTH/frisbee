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

package testplan

import (
	"context"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type endpoint func(context.Context, *v1alpha1.TestPlan, v1alpha1.Action) (client.Object, error)

func (r *Controller) supportedActions() map[string]endpoint {
	return map[string]endpoint{
		"Service": r.service,
		"Cluster": r.cluster,
		"Chaos":   r.chaos,
		"Cascade": r.cascade,
		"Delete":  r.delete,
		"Call":    r.call,
	}
}

func (r *Controller) service(ctx context.Context, t *v1alpha1.TestPlan, action v1alpha1.Action) (client.Object, error) {
	if err := expandMapInputs(ctx, r, t.GetNamespace(), &action.Service.Inputs); err != nil {
		return nil, errors.Wrapf(err, "input error")
	}

	// get the service template
	if err := action.Service.Prepare(false); err != nil {
		return nil, errors.Wrapf(err, "template validation")
	}

	spec, err := r.serviceControl.GetServiceSpec(ctx, t.GetNamespace(), *action.Service)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot retrieve service spec")
	}

	var service v1alpha1.Service

	service.SetGroupVersionKind(v1alpha1.GroupVersion.WithKind("Service"))
	service.SetNamespace(t.GetNamespace())
	service.SetName(action.Name)

	spec.DeepCopyInto(&service.Spec)

	return &service, nil
}

func (r *Controller) cluster(ctx context.Context, t *v1alpha1.TestPlan, action v1alpha1.Action) (client.Object, error) {
	if err := expandMapInputs(ctx, r, t.GetNamespace(), &action.Cluster.Inputs); err != nil {
		return nil, errors.Wrapf(err, "input error")
	}

	var cluster v1alpha1.Cluster

	cluster.SetGroupVersionKind(v1alpha1.GroupVersion.WithKind("Cluster"))
	cluster.SetNamespace(t.GetNamespace())
	cluster.SetName(action.Name)

	action.Cluster.DeepCopyInto(&cluster.Spec)

	return &cluster, nil
}

func (r *Controller) chaos(ctx context.Context, t *v1alpha1.TestPlan, action v1alpha1.Action) (client.Object, error) {
	if err := expandMapInputs(ctx, r, t.GetNamespace(), &action.Chaos.Inputs); err != nil {
		return nil, errors.Wrapf(err, "input error")
	}

	// get the service template
	if err := action.Chaos.Prepare(false); err != nil {
		return nil, errors.Wrapf(err, "template validation")
	}

	spec, err := r.chaosControl.GetChaosSpec(ctx, t.GetNamespace(), *action.Chaos)
	if err != nil {
		return nil, errors.Wrapf(err, "service spec")
	}

	if spec.Type == v1alpha1.FaultKill {
		if action.DependsOn.Success != nil {
			return nil, errors.Errorf("kill is a inject-only chaos. it does not have success. only running")
		}
	}

	var chaos v1alpha1.Chaos

	chaos.SetGroupVersionKind(v1alpha1.GroupVersion.WithKind("Chaos"))
	chaos.SetNamespace(t.GetNamespace())
	chaos.SetName(action.Name)

	spec.DeepCopyInto(&chaos.Spec)

	return &chaos, nil
}

func (r *Controller) cascade(ctx context.Context, t *v1alpha1.TestPlan, action v1alpha1.Action) (client.Object, error) {
	if err := expandMapInputs(ctx, r, t.GetNamespace(), &action.Cascade.Inputs); err != nil {
		return nil, errors.Wrapf(err, "input error")
	}

	var cascade v1alpha1.Cascade

	cascade.SetGroupVersionKind(v1alpha1.GroupVersion.WithKind("Cascade"))
	cascade.SetNamespace(t.GetNamespace())
	cascade.SetName(action.Name)

	action.Cascade.DeepCopyInto(&cascade.Spec)

	return &cascade, nil
}

func (r *Controller) delete(ctx context.Context, t *v1alpha1.TestPlan, action v1alpha1.Action) (client.Object, error) {
	// Delete normally does not return anything. This however would break all the pipeline for
	// managing dependencies between jobs. For that, we return a dummy virtual object without dedicated controller.
	var deletionJob v1alpha1.VirtualObject

	deletionJob.SetGroupVersionKind(v1alpha1.GroupVersion.WithKind("VirtualObject"))
	deletionJob.SetNamespace(t.GetNamespace())
	deletionJob.SetName(action.Name)

	deletionJob.SetReconcileStatus(v1alpha1.Lifecycle{
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
		job, deletable := r.state.IsDeletable(name)
		if !deletable {
			return nil, errors.Errorf("job %s is not currently deletable", name)
		}

		if err := r.GetClient().Delete(ctx, job, &options); err != nil {
			return nil, errors.Wrapf(err, "unable to delete job %s", job.GetName())
		}
	}

	return &deletionJob, nil
}

func (r *Controller) call(ctx context.Context, t *v1alpha1.TestPlan, action v1alpha1.Action) (client.Object, error) {
	if err := expandSliceInputs(ctx, r, t.GetNamespace(), &action.Call.Services); err != nil {
		return nil, errors.Wrapf(err, "input error")
	}

	var call v1alpha1.Call

	call.SetGroupVersionKind(v1alpha1.GroupVersion.WithKind("Call"))
	call.SetNamespace(t.GetNamespace())
	call.SetName(action.Name)

	action.Call.DeepCopyInto(&call.Spec)

	return &call, nil
}
