/*
Copyright 2022 ICS-FORTH.

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

package lifecycle

import (
	"fmt"
	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
	"github.com/r3labs/diff/v3"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Reasons for Failure
const (
	// AtLeastOneJobHasFailed is used when at least one job has failed, and there is toleration defined.
	AtLeastOneJobHasFailed = "AtLeastOneJobHasFailed"

	// TooManyJobsHaveFailed is used when the number of failures exceed the number of toleration.
	TooManyJobsHaveFailed = "TooManyJobsHaveFailed"

	// ExactlyOneJobIsFailed indicate that the only scheduled job is in the Failed Phase
	ExactlyOneJobIsFailed = "ExactlyOneJobIsFailed"
)

// Reasons for Success
const (
	// AllJobsAreSuccessful is when all the scheduled jobs are successfully completed.
	AllJobsAreSuccessful = "AllJobsAreSuccessful"

	// ExactlyOneJobIsSuccessful indicate that the only scheduled job is in the Success Phase
	ExactlyOneJobIsSuccessful = "ExactlyOneJobIsSuccessful"
)

// Reasons for Running
const (
	// AtLeastOneJobIsRunning indicate that  all jobs are created, and at least one is still running.
	AtLeastOneJobIsRunning = "AtLeastOneJobIsRunning"

	// ExactlyOneJobIsRunning indicate that the only scheduled job is in the Running phase
	ExactlyOneJobIsRunning = "ExactlyOneJobIsRunning"
)

// Reasons for Pending
const (
	// AtLeastOneJobIsNotScheduled indicate that there is at least one job that is not yet scheduled.
	AtLeastOneJobIsNotScheduled = "AtLeastOneJobIsNotScheduled"

	// ExactlyOneJobIsPending indicate that the only scheduled job is in the Pending phase
	ExactlyOneJobIsPending = "ExactlyOneJobIsPending"
)

type test struct {
	expression bool
	lifecycle  v1alpha1.Lifecycle
	condition  metav1.Condition
}

func NothingScheduled(state ClassifierReader) bool {
	return state.NumPendingJobs()+
		state.NumRunningJobs()+
		state.NumSuccessfulJobs()+
		state.NumFailedJobs() == 0
}

// GroupedJobs calculate the lifecycle for action with multiple sub-jobs, such as Clusters, Cascade, Calls, ...
func GroupedJobs(queuedJobs int, state ClassifierReader, lf *v1alpha1.Lifecycle, tolerate *v1alpha1.TolerateSpec) bool {
	var testSequence []test

	// The job is just received and hasn't scheduled anything yet.
	if NothingScheduled(state) {
		return false
	}

	// When there are failed jobs, we need to differentiate the number of tolerated failures.
	if tolerate != nil {
		message := fmt.Sprintf("tolerate: %d. failed: %d (%s)",
			tolerate.FailedJobs, state.NumFailedJobs(), state.ListFailedJobs())

		// A job has been failed, but it is within the expected toleration.
		testSequence = append(testSequence, test{
			expression: state.NumFailedJobs() > tolerate.FailedJobs,
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseFailed,
				Reason:  TooManyJobsHaveFailed,
				Message: message,
			},
			condition: metav1.Condition{
				Type:    v1alpha1.ConditionJobUnexpectedTermination.String(),
				Status:  metav1.ConditionTrue,
				Reason:  TooManyJobsHaveFailed,
				Message: message,
			},
		})
	} else {
		message := fmt.Sprintf("failed: %s", state.ListFailedJobs())

		// A job has failed during execution.
		testSequence = append(testSequence, test{
			expression: state.NumFailedJobs() > 0,
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseFailed,
				Reason:  AtLeastOneJobHasFailed,
				Message: message,
			},
			condition: metav1.Condition{
				Type:    v1alpha1.ConditionJobUnexpectedTermination.String(),
				Status:  metav1.ConditionTrue,
				Reason:  AtLeastOneJobHasFailed,
				Message: message,
			},
		})
	}

	// Generic sequence
	testSequence = append(testSequence, []test{
		{ // All jobs are successfully completed
			expression: state.NumSuccessfulJobs() == queuedJobs,
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseSuccess,
				Reason:  AllJobsAreSuccessful,
				Message: fmt.Sprintf("successful: %s", state.ListSuccessfulJobs()),
			},
			condition: metav1.Condition{
				Type:    v1alpha1.ConditionAllJobsAreCompleted.String(),
				Status:  metav1.ConditionTrue,
				Reason:  AllJobsAreSuccessful,
				Message: fmt.Sprintf("successful: %s", state.ListSuccessfulJobs()),
			},
		},

		{ // All jobs are created, and at least one is still running
			expression: state.NumRunningJobs()+state.NumSuccessfulJobs() == queuedJobs,
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseRunning,
				Reason:  AtLeastOneJobIsRunning,
				Message: fmt.Sprintf("running: %s", state.ListRunningJobs()),
			},
			condition: metav1.Condition{
				Type:    v1alpha1.ConditionAllJobsAreScheduled.String(),
				Status:  metav1.ConditionTrue,
				Reason:  AtLeastOneJobIsRunning,
				Message: fmt.Sprintf("running: %s", state.ListRunningJobs()),
			},
		},

		{ // Not all Jobs are yet created
			expression: lf.Phase == v1alpha1.PhasePending,
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhasePending,
				Reason:  AtLeastOneJobIsNotScheduled,
				Message: fmt.Sprintf("scheduled:%d/%d", state.Count(), queuedJobs),
			},
		},
	}...)

	for _, testcase := range testSequence {
		if testcase.expression { // Check if any lifecycle condition is met
			if diff.Changed(*lf, testcase.lifecycle) { // Update only if there is any change
				*lf = testcase.lifecycle

				if testcase.condition != (metav1.Condition{}) {
					meta.SetStatusCondition(&lf.Conditions, testcase.condition)
				}

				return true
			} else {
				// do nothing
				return false
			}
		}
	}

	panic(errors.Errorf(`unhandled lifecycle conditions.
		current: '%v',
		jobs: '%s',
	`, lf, state.ListAll()))
}

func SingleJob(state ClassifierReader, lf *v1alpha1.Lifecycle) bool {
	// The object exists in more than one known states
	if state.Count() > 1 {
		panic(errors.Errorf("invalid state transition: %s", state.ListAll()))
	}

	// The job is just received and hasn't scheduled anything yet.
	if NothingScheduled(state) {
		return false
	}

	testSequence := []test{
		{ // Failed
			expression: state.NumFailedJobs() == 1,
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseFailed,
				Reason:  ExactlyOneJobIsFailed,
				Message: fmt.Sprintf("failed: %s", state.ListFailedJobs()),
			},
			condition: metav1.Condition{
				Type:    v1alpha1.ConditionJobUnexpectedTermination.String(),
				Status:  metav1.ConditionTrue,
				Reason:  ExactlyOneJobIsFailed,
				Message: fmt.Sprintf("failed: %s", state.ListFailedJobs()),
			},
		},
		{ // Successful
			expression: state.NumSuccessfulJobs() == 1,
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseSuccess,
				Reason:  ExactlyOneJobIsSuccessful,
				Message: fmt.Sprintf("successful: %s", state.ListSuccessfulJobs()),
			},
			condition: metav1.Condition{
				Type:    v1alpha1.ConditionAllJobsAreCompleted.String(),
				Status:  metav1.ConditionTrue,
				Reason:  ExactlyOneJobIsSuccessful,
				Message: fmt.Sprintf("successful: %s", state.ListSuccessfulJobs()),
			},
		},
		{ // Running
			expression: state.NumRunningJobs() == 1,
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseRunning,
				Reason:  ExactlyOneJobIsRunning,
				Message: fmt.Sprintf("running: %s", state.ListRunningJobs()),
			},
			condition: metav1.Condition{
				Type:    v1alpha1.ConditionAllJobsAreScheduled.String(),
				Status:  metav1.ConditionTrue,
				Reason:  ExactlyOneJobIsRunning,
				Message: fmt.Sprintf("running: %s", state.ListRunningJobs()),
			},
		},
		{ // Pending
			expression: state.NumPendingJobs() == 1,
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhasePending,
				Reason:  ExactlyOneJobIsPending,
				Message: fmt.Sprintf("pending: %s", state.ListPendingJobs()),
			},
			condition: metav1.Condition{
				Type:    v1alpha1.ConditionCRInitialized.String(),
				Status:  metav1.ConditionTrue,
				Reason:  ExactlyOneJobIsPending,
				Message: fmt.Sprintf("pending: %s", state.ListPendingJobs()),
			},
		},
	}

	for _, testcase := range testSequence {
		if testcase.expression { // Check if any lifecycle condition is met
			if diff.Changed(*lf, testcase.lifecycle) { // Update only if there is any change
				*lf = testcase.lifecycle

				if testcase.condition != (metav1.Condition{}) {
					meta.SetStatusCondition(&lf.Conditions, testcase.condition)
				}

				return true
			} else {
				// do nothing
				return false
			}
		}
	}

	panic(errors.Errorf(`unhandled lifecycle conditions.
		current: '%v',
		jobs: '%s',
	`, lf, state.ListAll()))
}
