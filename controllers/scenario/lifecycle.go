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
	"fmt"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common/expressions"
	"github.com/carv-ics-forth/frisbee/controllers/common/lifecycle"
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

	panic("this should never happen")
}

func (r *Controller) updateLifecycle(t *v1alpha1.Scenario) v1alpha1.Lifecycle {
	cycle := t.Status.Lifecycle
	gs := r.view

	// Step 1. Skip any scenario which are already completed, or uninitialized.
	if cycle.Phase.Is(v1alpha1.PhaseUninitialized, v1alpha1.PhaseSuccess, v1alpha1.PhaseFailed) {
		return cycle
	}

	for _, actionName := range t.Status.ScheduledJobs {
		action := getActionOrDie(t, actionName)

		if !action.Assert.IsZero() {
			eval := expressions.Condition{Expr: action.Assert}

			if !eval.IsTrue(gs, t) {
				cycle = v1alpha1.Lifecycle{
					Phase:   v1alpha1.PhaseFailed,
					Reason:  "AssertError",
					Message: fmt.Sprintf("AssertError for actionName '%s'. Info: %s", action.Name, eval.Info),
				}

				meta.SetStatusCondition(&cycle.Conditions, metav1.Condition{
					Type:    v1alpha1.ConditionAssert.String(),
					Status:  metav1.ConditionTrue,
					Reason:  "AssertError",
					Message: fmt.Sprintf("AssertError for actionName '%s'. Info: %s", action.Name, eval.Info),
				})

				return cycle
			}
		}
	}

	// Step 4. Check if scheduling goes as expected.
	queuedJobs := len(t.Spec.Actions)

	if lifecycle.GroupedJobs(queuedJobs, gs, &cycle, nil) {
		return cycle
	}

	panic(errors.Errorf(`unhandled lifecycle conditions.
		current: %v,
		total: %d,
		pendingJobs: %s,
		runningJobs: %s,
		successfulJobs: %s,
		failedJobs: %s
	`, cycle, queuedJobs, gs.ListPendingJobs(), gs.ListRunningJobs(), gs.ListSuccessfulJobs(), gs.ListFailedJobs()))
}
