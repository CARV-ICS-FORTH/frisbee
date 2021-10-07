// Licensed to FORTH/ICS under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. FORTH/ICS licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package workflow

import (
	"fmt"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/utils"
	"github.com/sirupsen/logrus"
)

var systemService = []string{"prometheus", "grafana"}

func calculateLifecycle(w *v1alpha1.Workflow, gs utils.LifecycleClassifier) v1alpha1.Lifecycle {
	status := w.Status

	if status.Phase == v1alpha1.PhaseUninitialized ||
		status.Phase == v1alpha1.PhaseSuccess ||
		status.Phase == v1alpha1.PhaseFailed {
		return status.Lifecycle
	}

	// we are only interested in the number of jobs in each category.
	expectedJobs := len(w.Spec.Actions) + len(systemService)
	activeJobs := gs.NumActiveJobs()
	runningJobs := gs.NumRunningJobs()
	successfulJobs := gs.NumSuccessfulJobs()
	failedJobs := gs.NumFailedJobs()

	_ = runningJobs

	type test struct {
		condition bool
		outcome   v1alpha1.Lifecycle
	}

	tests := []test{
		{ // A job has failed during execution.
			condition: failedJobs > 0,
			outcome: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseFailed,
				Reason:  "JobHasFailed",
				Message: fmt.Sprintf("failed jobs: %s", gs.FailedList()),
			},
		},
		{ // All jobs are created, and completed successfully
			condition: successfulJobs == expectedJobs,
			outcome: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseSuccess,
				Reason:  "AllJobsCompleted",
				Message: fmt.Sprintf("successful jobs: %s", gs.SuccessfulList()),
			},
		},
		{ // All jobs are created, and at least one is still running
			condition: activeJobs+successfulJobs == expectedJobs,
			outcome: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseRunning,
				Reason:  "JobIsRunning",
				Message: fmt.Sprintf("active jobs: %s", gs.ActiveList()),
			},
		},
		{ // Not all Jobs are constructed created
			condition: status.Phase == v1alpha1.PhasePending,
			outcome: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhasePending,
				Reason:  "JobIsPending",
				Message: "at least one jobs has not yet created",
			},
		},
	}

	for _, testcase := range tests {
		if testcase.condition {
			return testcase.outcome
		}
	}

	logrus.Warn("Workflow Debug info \n",
		" current ", status.Lifecycle.Phase,
		" total actions: ", len(w.Spec.Actions),
		" activeJobs: ", gs.ActiveList(),
		" successfulJobs: ", gs.SuccessfulList(),
		" failedJobs: ", gs.FailedList(),
		" cur status: ", status,
	)

	panic("unhandled lifecycle conditions")
}
