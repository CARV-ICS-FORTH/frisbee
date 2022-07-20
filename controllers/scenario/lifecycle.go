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
	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common/expressions"
	"github.com/carv-ics-forth/frisbee/controllers/common/lifecycle/check"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *Controller) updateLifecycle(t *v1alpha1.Scenario) v1alpha1.Lifecycle {
	cycle := t.Status.Lifecycle
	gs := r.clusterView

	// Step 1. Skip any scenario which are already completed, or uninitialized.
	if cycle.Phase.Is(v1alpha1.PhaseUninitialized, v1alpha1.PhaseSuccess, v1alpha1.PhaseFailed) {
		return cycle
	}

	// Step 2. Because the assertions belong to the scenario, we must check if the scenario is the
	// recipient of any fired alert.
	if _, info, fired := expressions.AlertIsFired(t); fired {
		cycle = v1alpha1.Lifecycle{
			Phase:   v1alpha1.PhaseFailed,
			Reason:  "MetricsAssertion",
			Message: info,
		}

		meta.SetStatusCondition(&cycle.Conditions, metav1.Condition{
			Type:    v1alpha1.ConditionTerminated.String(),
			Status:  metav1.ConditionTrue,
			Reason:  "MetricsAssertion",
			Message: info,
		})

		return cycle
	}

	// Step 3. Check if state-driven assertions are fired
	for _, assertion := range t.Status.ExecutedActions {
		if assertion.IsZero() {
			continue
		}

		if assertion.HasStateExpr() {
			info, fired, err := expressions.FiredState(assertion.State, gs)
			if err != nil {
				cycle = v1alpha1.Lifecycle{
					Phase:   v1alpha1.PhaseFailed,
					Reason:  "StateQueryError",
					Message: err.Error(),
				}

				meta.SetStatusCondition(&cycle.Conditions, metav1.Condition{
					Type:    v1alpha1.ConditionTerminated.String(),
					Status:  metav1.ConditionTrue,
					Reason:  "StateQueryError",
					Message: err.Error(),
				})

				return cycle
			}

			if fired {
				cycle = v1alpha1.Lifecycle{
					Phase:   v1alpha1.PhaseRunning,
					Reason:  "StateAssertion",
					Message: info,
				}

				meta.SetStatusCondition(&cycle.Conditions, metav1.Condition{
					Type:    v1alpha1.ConditionTerminated.String(),
					Status:  metav1.ConditionTrue,
					Reason:  "StateAssertion",
					Message: info,
				})

				return cycle
			}
		}
	}

	// Step 4. Check if scheduling goes as expected.
	queuedJobs := len(t.Spec.Actions)

	if check.ScheduledJobs(queuedJobs, gs, &cycle) {
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
