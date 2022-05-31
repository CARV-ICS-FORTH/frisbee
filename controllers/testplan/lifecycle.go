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
	"fmt"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common/expressions"
	"github.com/carv-ics-forth/frisbee/controllers/common/lifecycle"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type test struct {
	expression bool
	lifecycle  v1alpha1.Lifecycle
	condition  metav1.Condition
}

func (r *Controller) updateLifecycle(t *v1alpha1.TestPlan, clusterView lifecycle.ClassifierReader) v1alpha1.Lifecycle {
	life := t.Status.Lifecycle

	// Step 1. Skip any CR which are already completed, or uninitialized.
	if life.Phase.Is(v1alpha1.PhaseUninitialized, v1alpha1.PhaseSuccess, v1alpha1.PhaseFailed) {
		return life
	}

	// Step 2. Check if metrics-driven assertions are fired
	if info, fired := expressions.FiredAlert(t); fired {
		life = v1alpha1.Lifecycle{
			Phase:   v1alpha1.PhaseFailed,
			Reason:  "MetricsAssertion",
			Message: info,
		}

		meta.SetStatusCondition(&life.Conditions, metav1.Condition{
			Type:    v1alpha1.ConditionTerminated.String(),
			Status:  metav1.ConditionTrue,
			Reason:  "MetricsAssertion",
			Message: info,
		})

		return life
	}

	// Step 3. Check if state-driven assertions are fired
	for _, assertion := range t.Status.ExecutedActions {
		if assertion.IsZero() {
			continue
		}

		if assertion.HasStateExpr() {
			info, fired, err := expressions.FiredState(assertion.State, clusterView)
			if err != nil {
				life = v1alpha1.Lifecycle{
					Phase:   v1alpha1.PhaseFailed,
					Reason:  "StateQueryError",
					Message: err.Error(),
				}

				meta.SetStatusCondition(&life.Conditions, metav1.Condition{
					Type:    v1alpha1.ConditionTerminated.String(),
					Status:  metav1.ConditionTrue,
					Reason:  "StateQueryError",
					Message: err.Error(),
				})

				return life
			}

			if fired {
				life = v1alpha1.Lifecycle{
					Phase:   v1alpha1.PhaseRunning,
					Reason:  "StateAssertion",
					Message: info,
				}

				meta.SetStatusCondition(&life.Conditions, metav1.Condition{
					Type:    v1alpha1.ConditionTerminated.String(),
					Status:  metav1.ConditionTrue,
					Reason:  "StateAssertion",
					Message: info,
				})

				return life
			}
		}
	}

	// we are only interested in the number of jobs in each category.
	expectedJobs := len(t.Spec.Actions)

	selftests := []test{
		{ // A job has failed during execution.
			expression: clusterView.FailedJobsNum() > 0,
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseFailed,
				Reason:  "JobHasFailed",
				Message: fmt.Sprintf("failed jobs: %s", clusterView.FailedJobsList()),
			},
			condition: metav1.Condition{
				Type:    v1alpha1.ConditionJobUnexpectedTermination.String(),
				Status:  metav1.ConditionTrue,
				Reason:  "JobHasFailed",
				Message: fmt.Sprintf("failed jobs: %s", clusterView.FailedJobsList()),
			},
		},
		{ // All jobs are created, and completed successfully
			expression: clusterView.SuccessfulJobsNum() == expectedJobs,
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseSuccess,
				Reason:  "AllJobsCompleted",
				Message: fmt.Sprintf("successful jobs: %s", clusterView.SuccessfulJobsList()),
			},
			condition: metav1.Condition{
				Type:    v1alpha1.ConditionAllJobsAreCompleted.String(),
				Status:  metav1.ConditionTrue,
				Reason:  "AllJobsCompleted",
				Message: fmt.Sprintf("successful jobs: %s", clusterView.SuccessfulJobsList()),
			},
		},
		{ // All jobs are created, and at least one is still running
			expression: len(t.Status.ExecutedActions) == expectedJobs,
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseRunning,
				Reason:  "JobIsRunning",
				Message: fmt.Sprintf("running jobs: %s", clusterView.RunningJobsList()),
			},
			condition: metav1.Condition{
				Type:    v1alpha1.ConditionAllJobsAreScheduled.String(),
				Status:  metav1.ConditionTrue,
				Reason:  "AllJobsRunning",
				Message: fmt.Sprintf("running jobs: %s", clusterView.RunningJobsList()),
			},
		},
		{ // Not all Jobs are yet created
			expression: len(t.Status.ExecutedActions) < expectedJobs,
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhasePending,
				Reason:  "JobIsPending",
				Message: "at least one jobs has not yet created",
			},
		},
	}

	for _, testcase := range selftests {
		if testcase.expression {
			life = testcase.lifecycle

			if testcase.condition != (metav1.Condition{}) {
				meta.SetStatusCondition(&life.Conditions, testcase.condition)
			}

			return life
		}
	}

	logrus.Warn("TestPlan Debug info \n",
		" phase ", life.Phase,
		" actions: ", expectedJobs,
		" executed: ", len(t.Status.ExecutedActions),
		" pending: ", clusterView.PendingJobsList(),
		" running: ", clusterView.RunningJobsList(),
		" successfulJobs: ", clusterView.SuccessfulJobsList(),
		" failedJobs: ", clusterView.FailedJobsList(),
		" cur status: ", t.Status,
	)

	panic("unhandled lifecycle conditions")
}
