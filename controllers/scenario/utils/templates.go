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

package utils

import (
	"context"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	chaosutils "github.com/carv-ics-forth/frisbee/controllers/chaos/utils"
	serviceutils "github.com/carv-ics-forth/frisbee/controllers/service/utils"
	"github.com/carv-ics-forth/frisbee/pkg/infrastructure"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func LoadTemplates(ctx context.Context, cli client.Client, scenario *v1alpha1.Scenario) error {
	// list the available nodes
	readyNodes, err := infrastructure.GetReadyNodes(ctx, cli)
	if err != nil {
		return errors.Wrapf(err, "cannot list nodes")
	}

	// list the total allocatable resources from all nodes
	allocatableResources := infrastructure.TotalAllocatableResources(readyNodes...)

	// LoadTemplates Reference Graph
	for i := 0; i < len(scenario.Spec.Actions); i++ {
		action := &scenario.Spec.Actions[i]

		switch action.ActionType {
		case v1alpha1.ActionService:
			if err := ExpandMacros(ctx, cli, scenario.GetNamespace(), &action.Service.Inputs); err != nil {
				return errors.Wrapf(err, "input error")
			}

			if _, err := serviceutils.GetServiceSpec(ctx, cli, scenario, *action.Service); err != nil {
				return errors.Wrapf(err, "service '%s' error", action.Name)
			}

		case v1alpha1.ActionCluster:
			if err := ExpandMacros(ctx, cli, scenario.GetNamespace(), &action.Cluster.Inputs); err != nil {
				return errors.Wrapf(err, "input error")
			}

			if _, err := serviceutils.GetServiceSpecList(ctx, cli, scenario, action.Cluster.GenerateObjectFromTemplate); err != nil {
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
				if err := infrastructure.RequestIsWithinLimits(action.Cluster.Resources.TotalResources, allocatableResources); err != nil {
					return errors.Wrapf(err, "Overprovisioning error for Cluster '%s'", action.Name)
				}
			}

		case v1alpha1.ActionChaos:
			if err := ExpandMacros(ctx, cli, scenario.GetNamespace(), &action.Chaos.Inputs); err != nil {
				return errors.Wrapf(err, "input error")
			}

			if _, err := chaosutils.GetChaosSpec(ctx, cli, scenario, *action.Chaos); err != nil {
				return errors.Wrapf(err, "chaos '%s' error", action.Name)
			}

		case v1alpha1.ActionCascade:
			if err := ExpandMacros(ctx, cli, scenario.GetNamespace(), &action.Cascade.Inputs); err != nil {
				return errors.Wrapf(err, "input error")
			}

			if _, err := chaosutils.GetChaosSpecList(ctx, cli, scenario, action.Cascade.GenerateObjectFromTemplate); err != nil {
				return errors.Wrapf(err, "cascade '%s' error", action.Name)
			}

		case v1alpha1.ActionCall:
			if err := ExpandSliceInputs(ctx, cli, scenario.GetNamespace(), &action.Call.Services); err != nil {
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
