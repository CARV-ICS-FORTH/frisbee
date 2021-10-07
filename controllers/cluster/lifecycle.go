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
//
// Most of this part is adapted from:
// https://github.com/kubernetes-sigs/kubebuilder/blob/master/docs/book/src/cronjob-tutorial/testdata/project/controllers/cronjob_controller.go

package cluster

import (
	"fmt"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/utils"
	"github.com/sirupsen/logrus"
)

// calculateLifecycle returns the update lifecycle of the cluster.
// Summary:
// * if all services are successfully complete the cluster will terminate successfully.
// * If there is a failed job, the cluster will fail itself.
func calculateLifecycle(cluster *v1alpha1.Cluster, gs utils.LifecycleClassifier) v1alpha1.Lifecycle {
	status := cluster.Status

	allJobs := len(cluster.Status.Expected)

	// we are only interested in the number of jobs in each category.
	// expectedJobs := len(w.Spec.Actions) + len(systemService)
	activeJobs := gs.NumActiveJobs()
	runningJobs := gs.NumRunningJobs()
	successfulJobs := gs.NumSuccessfulJobs()
	failedJobs := gs.NumFailedJobs()

	_ = runningJobs

	switch {
	case status.Phase == v1alpha1.PhaseUninitialized ||
		status.Phase == v1alpha1.PhaseSuccess ||
		status.Phase == v1alpha1.PhaseFailed:
		return status.Lifecycle

	case failedJobs > 0:
		if tolerate := cluster.Spec.Tolerate; tolerate != nil {
			if failedJobs > tolerate.FailedServices {
				return v1alpha1.Lifecycle{
					Phase:   v1alpha1.PhaseFailed,
					Reason:  "TolerateLimitsExceeded",
					Message: fmt.Sprintf("tolerate: %d. failed jobs: %s", tolerate.FailedServices, gs.FailedList()),
				}
			}

			// Ignore the failure.
			return status.Lifecycle
		}

	case successfulJobs == allJobs:
		// All jobs are successfully completed
		return v1alpha1.Lifecycle{
			Phase:   v1alpha1.PhaseSuccess,
			Reason:  "AllJobsCompleted",
			Message: fmt.Sprintf("successful jobs: %s", gs.SuccessfulList()),
		}

	case activeJobs+successfulJobs == allJobs:
		// All jobs are created, and at least one is still running
		return v1alpha1.Lifecycle{
			Phase:   v1alpha1.PhaseRunning,
			Reason:  "JobIsRunning",
			Message: fmt.Sprintf("active jobs: %s", gs.ActiveList()),
		}

	case status.Phase == v1alpha1.PhasePending:
		// Not all Jobs are constructed created
		return v1alpha1.Lifecycle{
			Phase:   v1alpha1.PhasePending,
			Reason:  "JobIsPending",
			Message: "at least one jobs has not yet created",
		}

	default:
		logrus.Warn("Cluster Debug info \n",
			" current ", status.Lifecycle.Phase,
			" total jobs: ", allJobs,
			" activeJobs: ", activeJobs,
			" successfulJobs: ", successfulJobs,
			" failedJobs: ", failedJobs,
			" cur status: ", status,
		)

		panic("unhandled lifecycle condition")
	}

	panic("this should never happen")

	// TODO: validate the transition. For example, we cannot go from running to pending
	/*

		// There are still pending jobs to be created
		return v1alpha1.Lifecycle{
			Kind:   "Cluster",
			Name:   cluster.GetName(),
			Phase:  v1alpha1.PhasePending,
			Reason: "waiting for jobs to start",
		}

	*/
}
