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

package cluster

import (
	"fmt"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/utils/lifecycle"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type test struct {
	expression bool
	lifecycle  v1alpha1.Lifecycle
	condition  metav1.Condition
}

// calculateLifecycle returns the update lifecycle of the cluster.
func calculateLifecycle(cluster *v1alpha1.Cluster, gs lifecycle.Classifier) v1alpha1.ClusterStatus {
	status := cluster.Status

	// Skip any CR which are already completed, or uninitialized.
	if status.Phase == v1alpha1.PhaseUninitialized ||
		status.Phase == v1alpha1.PhaseSuccess ||
		status.Phase == v1alpha1.PhaseFailed {
		return status
	}

	expectedJobs := len(cluster.Status.Expected)

	autotests := []test{
		{ // A job has failed during execution.
			expression: gs.NumFailedJobs() > 0 && gs.NumFailedJobs() > cluster.Spec.Tolerate.FailedServices,
			lifecycle: v1alpha1.Lifecycle{
				Phase:  v1alpha1.PhaseFailed,
				Reason: "TolerateFailuresExceeded",
				Message: fmt.Sprintf("tolerate: %s. failed jobs: %s",
					cluster.Spec.Tolerate.String(), gs.FailedList()),
			},
			condition: metav1.Condition{
				Type:    v1alpha1.ConditionJobFailed.String(),
				Status:  metav1.ConditionTrue,
				Reason:  "JobHasFailed",
				Message: fmt.Sprintf("failed jobs: %s", gs.FailedList()),
			},
		},
		{ // All jobs are successfully completed
			expression: gs.NumSuccessfulJobs() == expectedJobs,
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseSuccess,
				Reason:  "AllJobsCompleted",
				Message: fmt.Sprintf("successful jobs: %s", gs.SuccessfulList()),
			},
			condition: metav1.Condition{
				Type:    v1alpha1.ConditionAllJobsDone.String(),
				Status:  metav1.ConditionTrue,
				Reason:  "AllJobsCompleted",
				Message: fmt.Sprintf("successful jobs: %s", gs.SuccessfulList()),
			},
		},
		{ // All jobs are created, and at least one is still running
			expression: gs.NumRunningJobs()+gs.NumSuccessfulJobs() == expectedJobs,
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseRunning,
				Reason:  "JobIsRunning",
				Message: fmt.Sprintf("running jobs: %s", gs.RunningList()),
			},
			condition: metav1.Condition{
				Type:    v1alpha1.ConditionAllJobs.String(),
				Status:  metav1.ConditionTrue,
				Reason:  "AllJobsRunning",
				Message: fmt.Sprintf("running jobs: %s", gs.RunningList()),
			},
		},
		{ // Not all Jobs are yet created
			expression: status.Phase == v1alpha1.PhasePending,
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhasePending,
				Reason:  "JobIsPending",
				Message: "at least one jobs has not yet created",
			},
		},
	}

	for _, testcase := range autotests {
		if testcase.expression {
			status.Lifecycle = testcase.lifecycle

			if testcase.condition != (metav1.Condition{}) {
				meta.SetStatusCondition(&status.Conditions, testcase.condition)
			}

			return status
		}
	}

	panic(errors.Errorf(`unhandled lifecycle conditions. 
		current: %v
		total: %d,
		activeJobs: %s,
		successfulJobs: %s,
		failedJobs: %s
	`, status, expectedJobs, gs.ActiveList(), gs.SuccessfulList(), gs.FailedList()))
}
