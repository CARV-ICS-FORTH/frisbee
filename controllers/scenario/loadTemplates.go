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

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	chaosutils "github.com/carv-ics-forth/frisbee/controllers/chaos/utils"
	serviceutils "github.com/carv-ics-forth/frisbee/controllers/service/utils"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func (r *Controller) LoadTemplates(ctx context.Context, scenario *v1alpha1.Scenario) error {
	// list the available nodes
	readyNodes, err := r.getReadyNodes(ctx)
	if err != nil {
		return errors.Wrapf(err, "cannot list nodes")
	}

	// list the total allocatable resources from all nodes
	allocatableResources := totalAllocatableResources(readyNodes...)

	// LoadTemplates Reference Graph
	for i := 0; i < len(scenario.Spec.Actions); i++ {
		action := &scenario.Spec.Actions[i]

		switch action.ActionType {
		case v1alpha1.ActionService:
			if err := expandMacros(ctx, r, scenario.GetNamespace(), &action.Service.Inputs); err != nil {
				return errors.Wrapf(err, "input error")
			}

			if _, err := serviceutils.GetServiceSpec(ctx, r.GetClient(), scenario, *action.Service); err != nil {
				return errors.Wrapf(err, "service '%s' error", action.Name)
			}

		case v1alpha1.ActionCluster:
			if err := expandMacros(ctx, r, scenario.GetNamespace(), &action.Cluster.Inputs); err != nil {
				return errors.Wrapf(err, "input error")
			}

			if _, err := serviceutils.GetServiceSpecList(ctx, r.GetClient(), scenario, action.Cluster.GenerateObjectFromTemplate); err != nil {
				return errors.Wrapf(err, "cluster '%s' error", action.Name)
			}

			// LoadTemplates Placement Policies
			if action.Cluster.Placement != nil {
				// ensure there are at least two physical nodes for placement to make sense
				if len(readyNodes) < 2 {
					return errors.Errorf("Placement requires at least two ready nodes. Found: %v", readyNodes)
				}
			}

			// LoadTemplates Resource Policies
			if action.Cluster.Resources != nil {
				if err := resourceRequestIsWithinLimits(action.Cluster.Resources.TotalResources, allocatableResources); err != nil {
					return errors.Wrapf(err, "Overprovisioning error for Cluster '%s'", action.Name)
				}
			}

		case v1alpha1.ActionChaos:
			if err := expandMacros(ctx, r, scenario.GetNamespace(), &action.Chaos.Inputs); err != nil {
				return errors.Wrapf(err, "input error")
			}

			if _, err := chaosutils.GetChaosSpec(ctx, r.GetClient(), scenario, *action.Chaos); err != nil {
				return errors.Wrapf(err, "chaos '%s' error", action.Name)
			}

		case v1alpha1.ActionCascade:
			if err := expandMacros(ctx, r, scenario.GetNamespace(), &action.Cascade.Inputs); err != nil {
				return errors.Wrapf(err, "input error")
			}

			if _, err := chaosutils.GetChaosSpecList(ctx, r.GetClient(), scenario, action.Cascade.GenerateObjectFromTemplate); err != nil {
				return errors.Wrapf(err, "cascade '%s' error", action.Name)
			}

		case v1alpha1.ActionCall:
			if err := expandSliceInputs(ctx, r, scenario.GetNamespace(), &action.Call.Services); err != nil {
				return errors.Wrapf(err, "input error")
			}

			// TODO: now that the templates are loaded, ensure that the referenced callables exist.

		case v1alpha1.ActionDelete:
			// calls and deletes do not involve templates.
			return nil
		}
	}

	return nil
}

func (r *Controller) getReadyNodes(ctx context.Context) ([]corev1.Node, error) {
	var nodes corev1.NodeList

	if err := r.GetClient().List(ctx, &nodes); err != nil {
		return nil, errors.Wrapf(err, "cannot list physical nodes")
	}

	ready := make([]corev1.Node, 0, len(nodes.Items))

	for _, node := range nodes.Items {
		// search at the node's condition for the "NodeReady".
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
				ready = append(ready, node)

				r.Info("Node", "name", node.GetName(), "ready", true)

				goto next
			}
		}

		r.Info("Node", "name", node.GetName(), "ready", false)
	next:
	}

	// TODO: check compatibility with labels and taints

	return ready, nil
}

func totalAllocatableResources(nodeList ...corev1.Node) corev1.ResourceList {
	var (
		cpu       resource.Quantity
		memory    resource.Quantity
		pods      resource.Quantity
		storage   resource.Quantity
		ephemeral resource.Quantity
	)

	for _, node := range nodeList {
		cpu.Add(*node.Status.Allocatable.Cpu())
		memory.Add(*node.Status.Allocatable.Memory())
		pods.Add(*node.Status.Allocatable.Pods())
		storage.Add(*node.Status.Allocatable.Storage())
		ephemeral.Add(*node.Status.Allocatable.StorageEphemeral())
	}

	return corev1.ResourceList{
		corev1.ResourceCPU:              cpu,
		corev1.ResourceMemory:           memory,
		corev1.ResourcePods:             pods,
		corev1.ResourceStorage:          storage,
		corev1.ResourceEphemeralStorage: ephemeral,
	}
}

func resourceRequestIsWithinLimits(ask corev1.ResourceList, allocatable corev1.ResourceList) error {
	var merr *multierror.Error

	if cmp := allocatable.Cpu().Cmp(*ask.Cpu()); cmp < 0 {
		merr = multierror.Append(merr,
			errors.Errorf("CPU: ask[%s] allocatable[%s]", ask.Cpu().String(), allocatable.Cpu().String()),
		)
	}

	if cmp := allocatable.Memory().Cmp(*ask.Memory()); cmp < 0 {
		merr = multierror.Append(merr,
			errors.Errorf("Memory: ask[%s] allocatable[%s]", ask.Memory().String(), allocatable.Memory().String()),
		)
	}

	if cmp := allocatable.Pods().Cmp(*ask.Pods()); cmp < 0 {
		merr = multierror.Append(merr,
			errors.Errorf("Pods: ask[%s] allocatable[%s]", ask.Pods().String(), allocatable.Pods().String()),
		)
	}

	if cmp := allocatable.Storage().Cmp(*ask.Storage()); cmp < 0 {
		merr = multierror.Append(merr,
			errors.Errorf("Storage: ask[%s] allocatable[%s]", ask.Storage().String(), allocatable.Storage().String()),
		)
	}

	if cmp := allocatable.StorageEphemeral().Cmp(*ask.StorageEphemeral()); cmp < 0 {
		merr = multierror.Append(merr,
			errors.Errorf("StorageEphemeral: ask[%s] allocatable[%s]", ask.StorageEphemeral().String(), allocatable.StorageEphemeral().String()),
		)
	}

	return merr.ErrorOrNil()
}
