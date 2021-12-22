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

package workflow

import (
	"fmt"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/utils/assertions"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type test struct {
	expression bool
	lifecycle  v1alpha1.Lifecycle
	condition  metav1.Condition
}

func (r *Controller) updateLifecycle(w *v1alpha1.Workflow) {
	// Skip any CR which are already completed, or uninitialized.
	if w.Status.Phase == v1alpha1.PhaseUninitialized ||
		w.Status.Phase == v1alpha1.PhaseSuccess ||
		w.Status.Phase == v1alpha1.PhaseFailed {
		return
	}

	if info, fired := assertions.FiredAlert(w); fired {
		w.Status.Lifecycle = v1alpha1.Lifecycle{
			Phase:   v1alpha1.PhaseFailed,
			Reason:  "AssertionError",
			Message: info,
		}

		meta.SetStatusCondition(&w.Status.Conditions, metav1.Condition{
			Type:    v1alpha1.ConditionTerminated.String(),
			Status:  metav1.ConditionTrue,
			Reason:  "AssertionError",
			Message: info,
		})

		return
	}

	// handle assertions for successfully completed operations
	for _, job := range r.state.SuccessfulJobs() {
		for _, action := range w.Spec.Actions {
			if job.GetName() == action.Name {
				if action.Assert == nil {
					continue
				}

				info, fired, err := assertions.FiredState(action.Assert.State, r.state)
				if err != nil || fired {
					w.Status.Lifecycle = v1alpha1.Lifecycle{
						Phase:   v1alpha1.PhaseFailed,
						Reason:  info,
						Message: err.Error(),
					}

					meta.SetStatusCondition(&w.Status.Conditions, metav1.Condition{
						Type:    v1alpha1.ConditionTerminated.String(),
						Status:  metav1.ConditionTrue,
						Reason:  info,
						Message: err.Error(),
					})
				}
			}
		}
	}

	// we are only interested in the number of jobs in each category.
	expectedJobs := len(w.Spec.Actions)

	selftests := []test{
		{ // A job has failed during execution.
			expression: r.state.NumFailedJobs() > 0,
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseFailed,
				Reason:  "JobHasFailed",
				Message: fmt.Sprintf("failed jobs: %s", r.state.FailedList()),
			},
			condition: metav1.Condition{
				Type:    v1alpha1.ConditionJobFailed.String(),
				Status:  metav1.ConditionTrue,
				Reason:  "JobHasFailed",
				Message: fmt.Sprintf("failed jobs: %s", r.state.FailedList()),
			},
		},
		{ // All jobs are created, and completed successfully
			expression: r.state.NumSuccessfulJobs() == expectedJobs,
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseSuccess,
				Reason:  "AllJobsCompleted",
				Message: fmt.Sprintf("successful jobs: %s", r.state.SuccessfulList()),
			},
			condition: metav1.Condition{
				Type:    v1alpha1.ConditionAllJobsCompleted.String(),
				Status:  metav1.ConditionTrue,
				Reason:  "AllJobsCompleted",
				Message: fmt.Sprintf("successful jobs: %s", r.state.SuccessfulList()),
			},
		},
		{ // All jobs are created, and at least one is still running
			expression: r.state.NumRunningJobs()+r.state.NumSuccessfulJobs() == expectedJobs,
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseRunning,
				Reason:  "JobIsRunning",
				Message: fmt.Sprintf("running jobs: %s", r.state.RunningList()),
			},
			condition: metav1.Condition{
				Type:    v1alpha1.ConditionAllJobsScheduled.String(),
				Status:  metav1.ConditionTrue,
				Reason:  "AllJobsRunning",
				Message: fmt.Sprintf("running jobs: %s", r.state.RunningList()),
			},
		},
		{ // Not all Jobs are yet created
			expression: w.Status.Phase == v1alpha1.PhasePending,
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhasePending,
				Reason:  "JobIsPending",
				Message: "at least one jobs has not yet created",
			},
		},
	}

	for _, testcase := range selftests {
		if testcase.expression {
			w.Status.Lifecycle = testcase.lifecycle

			if testcase.condition != (metav1.Condition{}) {
				meta.SetStatusCondition(&w.Status.Conditions, testcase.condition)
			}

			return
		}
	}

	logrus.Warn("Workflow Debug info \n",
		" current ", w.Status.Lifecycle.Phase,
		" total actions: ", len(w.Spec.Actions),
		" activeJobs: ", r.state.ActiveList(),
		" successfulJobs: ", r.state.SuccessfulList(),
		" failedJobs: ", r.state.FailedList(),
		" cur status: ", w.Status,
	)

	panic("unhandled lifecycle conditions")
}
