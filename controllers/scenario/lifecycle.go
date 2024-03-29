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
	"fmt"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/pkg/expressions"
	"github.com/carv-ics-forth/frisbee/pkg/lifecycle"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getActionOrDie returns the spec of the referenced action.
// if the action is not found, it panics.
func getActionOrDie(t *v1alpha1.Scenario, actionName string) *v1alpha1.Action {
	for i, match := range t.Spec.Actions {
		if actionName == match.Name {
			return &t.Spec.Actions[i]
		}
	}

	panic(errors.Errorf("cannot find action '%s'", actionName))
}

func (r *Controller) updateLifecycle(scenario *v1alpha1.Scenario) bool {
	// Step 1. Skip any scenario which are already completed, or uninitialized.
	if scenario.Status.Lifecycle.Phase.Is(v1alpha1.PhaseUninitialized, v1alpha1.PhaseSuccess, v1alpha1.PhaseFailed) {
		return false
	}

	for _, actionName := range scenario.Status.ScheduledJobs {
		action := getActionOrDie(scenario, actionName)

		if !action.Assert.IsZero() {
			eval := expressions.Condition{Expr: action.Assert}

			if !eval.IsTrue(r.view, scenario) {
				scenario.Status.Lifecycle.Phase = v1alpha1.PhaseFailed
				scenario.Status.Lifecycle.Reason = "AssertError"
				scenario.Status.Lifecycle.Message = fmt.Sprintf("action '%s' failed due to:'%s'", action.Name, eval.Info)

				meta.SetStatusCondition(&scenario.Status.Lifecycle.Conditions, metav1.Condition{
					Type:    v1alpha1.ConditionAssertionError.String(),
					Status:  metav1.ConditionTrue,
					Reason:  "AssertError",
					Message: fmt.Sprintf("action '%s' failed due to:'%s'", action.Name, eval.Info),
				})

				return true
			}
		}
	}

	// Step 4. Check if scheduling goes as expected.
	totalJobs := len(scenario.Spec.Actions)

	return lifecycle.GroupedJobs(totalJobs, r.view, &scenario.Status.Lifecycle, nil)
}
