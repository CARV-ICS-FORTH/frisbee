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

	// we are only interested in the number of jobs in each category.
	expectedJobs := len(w.Spec.Actions) + len(systemService)
	activeJobs := gs.NumActiveJobs()
	runningJobs := gs.NumRunningJobs()
	successfulJobs := gs.NumSuccessfulJobs()
	failedJobs := gs.NumFailedJobs()

	_ = runningJobs

	switch {
	case failedJobs > 0:
		// A job has failed during execution.
		return v1alpha1.Lifecycle{
			Phase:  v1alpha1.PhaseFailed,
			Reason: fmt.Sprintf("failed jobs: %d", failedJobs),
		}

	case successfulJobs == expectedJobs:
		// All jobs are created, and completed successfully
		return v1alpha1.Lifecycle{
			Phase:  v1alpha1.PhaseSuccess,
			Reason: fmt.Sprint("all jobs completed: ", w.Spec.Actions.ToString()),
		}

	case activeJobs+successfulJobs == expectedJobs:
		// All jobs are created, and at least one is still running
		return v1alpha1.Lifecycle{
			Phase:  v1alpha1.PhaseRunning,
			Reason: "Jobs are still running",
		}

	case status.Phase == v1alpha1.PhasePending:
		// Not all Jobs are constructed created
		return v1alpha1.Lifecycle{
			Phase:  v1alpha1.PhasePending,
			Reason: "Jobs are still pending",
		}

	default:
		logrus.Warn("Workflow Debug info \n",
			" current ", status.Lifecycle.Phase,
			" total actions: ", len(w.Spec.Actions),
			" activeJobs: ", gs.ActiveList(),
			" successfulJobs: ", gs.SuccessfulList(),
			" failedJobs: ", gs.FailedList(),
			" cur status: ", status,
		)

		return status.Lifecycle
	}

	/*


		case status.NextAction >= len(w.Spec.Actions):
			// All jobs are created, and at least one is still running
			return v1alpha1.Lifecycle{
				Kind:   "Workflow",
				Name:   w.GetName(),
				Phase:  v1alpha1.PhaseRunning,
				Reason: fmt.Sprint("all jobs completed: ", w.Spec.Actions.ToString()),
			}
		// There are still pending jobs to be created
		return v1alpha1.Lifecycle{
			Kind:   "Workflow",
			Name:   w.GetName(),
			Phase:  v1alpha1.PhasePending,
			Reason: "waiting for jobs to start",
		}

	*/
}
