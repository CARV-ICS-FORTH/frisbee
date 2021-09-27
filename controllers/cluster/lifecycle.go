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
	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/sirupsen/logrus"
)

// calculateLifecycle returns the update lifecycle of the cluster.
func calculateLifecycle(cluster *v1alpha1.Cluster, activeJobs, successfulJobs, failedJobs v1alpha1.SList) v1alpha1.Lifecycle {
	allJobs := cluster.Status.Expected
	newStatus := v1alpha1.Lifecycle{}

	logrus.Warnf("instances: (%d), activeJobs (%d): %s, successfulJobs (%d): %s, failedJobs (%d): %s",
		len(cluster.Status.Expected),
		len(activeJobs), activeJobs.ToString(),
		len(successfulJobs), successfulJobs.ToString(),
		len(failedJobs), failedJobs.ToString(),
	)

	switch {
	case len(activeJobs) == 0 &&
		len(successfulJobs) == 0 &&
		len(failedJobs) == 0:
		return v1alpha1.Lifecycle{
			Kind:   "Cluster",
			Name:   cluster.GetName(),
			Phase:  v1alpha1.PhaseInitializing,
			Reason: "submitting job request",
		}

	case len(failedJobs) > 0:
		newStatus = v1alpha1.Lifecycle{ // One job has failed
			Kind:   "Cluster",
			Name:   cluster.GetName(),
			Phase:  v1alpha1.PhaseFailed,
			Reason: failedJobs.ToString(),
		}

		return newStatus

	case len(successfulJobs) == len(allJobs): // All jobs are successfully completed
		newStatus = v1alpha1.Lifecycle{
			Kind:   "Cluster",
			Name:   cluster.GetName(),
			Phase:  v1alpha1.PhaseSuccess,
			Reason: successfulJobs.ToString(),
		}

		return newStatus

	case len(activeJobs)+len(successfulJobs) == len(allJobs): // All jobs are created, and at least one is still running
		newStatus = v1alpha1.Lifecycle{
			Kind:   "Cluster",
			Name:   cluster.GetName(),
			Phase:  v1alpha1.PhaseRunning,
			Reason: activeJobs.ToString(),
		}

		return newStatus

	default: // There are still pending jobs to be created
		newStatus = v1alpha1.Lifecycle{
			Kind:   "Cluster",
			Name:   cluster.GetName(),
			Phase:  v1alpha1.PhasePending,
			Reason: "waiting for jobs to start",
		}

		return newStatus
	}

	// TODO: validate the transition. For example, we cannot go from running to pending
}
