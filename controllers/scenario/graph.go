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
)

// Validate validates the execution workflow.
// 1. Ensures that action names are qualified (since they are used as generators to jobs)
// 2. Ensures that there are no two actions with the same name.
// 3. Ensure that dependencies point to a valid action.
// 4. Ensure that macros point to a valid action.
func (r *Controller) Validate(ctx context.Context, scenario *v1alpha1.Scenario) error {
	// Validate Reference Graph
	for _, action := range scenario.Spec.Actions {
		switch action.ActionType {
		case v1alpha1.ActionService:
			if _, err := serviceutils.GetServiceSpec(ctx, r.GetClient(), scenario, *action.Service); err != nil {
				return errors.Wrapf(err, "service '%s' error", action.Name)
			}
		case v1alpha1.ActionCluster:
			if _, err := serviceutils.GetServiceSpec(ctx, r.GetClient(), scenario, action.Cluster.GenerateObjectFromTemplate); err != nil {
				return errors.Wrapf(err, "cluster '%s' error", action.Name)
			}

		case v1alpha1.ActionChaos:
			if _, err := chaosutils.GetChaosSpec(ctx, r.GetClient(), scenario, *action.Chaos); err != nil {
				return errors.Wrapf(err, "chaos '%s' error", action.Name)
			}

		case v1alpha1.ActionCascade:
			if _, err := chaosutils.GetChaosSpec(ctx, r.GetClient(), scenario, action.Cascade.GenerateObjectFromTemplate); err != nil {
				return errors.Wrapf(err, "cascade '%s' error", action.Name)
			}

		case v1alpha1.ActionCall, v1alpha1.ActionDelete:
			// TODO: should we add something here?
			return nil
		}
	}

	return nil
}
