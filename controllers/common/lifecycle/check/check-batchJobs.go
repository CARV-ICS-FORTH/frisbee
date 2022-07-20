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

package check

import (
	"fmt"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common/lifecycle"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type test struct {
	expression bool
	lifecycle  v1alpha1.Lifecycle
	condition  metav1.Condition
}

func ScheduledJobs(queuedJobs int, state lifecycle.ClassifierReader, lf *v1alpha1.Lifecycle) bool {

	autotests := []test{
		{ // A job has failed during execution.
			expression: state.FailedJobsNum() > 0,
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseFailed,
				Reason:  "JobHasFailed",
				Message: fmt.Sprintf("failed jobs: %s", state.FailedJobsList()),
			},
			condition: metav1.Condition{
				Type:    v1alpha1.ConditionJobUnexpectedTermination.String(),
				Status:  metav1.ConditionTrue,
				Reason:  "JobHasFailed",
				Message: fmt.Sprintf("failed jobs: %s", state.FailedJobsList()),
			},
		},

		{ // All jobs are successfully completed
			expression: state.SuccessfulJobsNum() == queuedJobs,
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseSuccess,
				Reason:  "AllJobsCompleted",
				Message: fmt.Sprintf("successful jobs: %s", state.SuccessfulJobsList()),
			},
			condition: metav1.Condition{
				Type:    v1alpha1.ConditionAllJobsAreCompleted.String(),
				Status:  metav1.ConditionTrue,
				Reason:  "AllJobsCompleted",
				Message: fmt.Sprintf("successful jobs: %s", state.SuccessfulJobsList()),
			},
		},

		{ // All jobs are created, and at least one is still running
			expression: state.RunningJobsNum()+state.SuccessfulJobsNum() == queuedJobs,
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseRunning,
				Reason:  "AllJobsRunning",
				Message: fmt.Sprintf("running jobs: %s", state.RunningJobsList()),
			},
			condition: metav1.Condition{
				Type:    v1alpha1.ConditionAllJobsAreScheduled.String(),
				Status:  metav1.ConditionTrue,
				Reason:  "AllJobsRunning",
				Message: fmt.Sprintf("running jobs: %s", state.RunningJobsList()),
			},
		},

		{ // Not all Jobs are yet created
			expression: lf.Phase == v1alpha1.PhasePending,
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhasePending,
				Reason:  "JobIsPending",
				Message: "at least one jobs has not yet created",
			},
		},
	}

	for _, testcase := range autotests {
		if testcase.expression {
			*lf = testcase.lifecycle

			if testcase.condition != (metav1.Condition{}) {
				meta.SetStatusCondition(&lf.Conditions, testcase.condition)
			}

			return true
		}
	}

	return false
}
