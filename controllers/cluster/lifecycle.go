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
// Summary:
// * if all services are successfully complete the cluster will terminate successfully.
// * If there is a failed job, the cluster will fail itself.
func calculateLifecycle(cluster *v1alpha1.Cluster, activeJobs, successfulJobs, failedJobs v1alpha1.SList) v1alpha1.Lifecycle {
	status := cluster.Status

	allJobs := len(cluster.Status.Expected)

	/*
		logrus.Warnf("Cluster - instances: (%d), activeJobs (%d): %s, successfulJobs (%d): %s, failedJobs (%d): %s",
			len(status.Expected),
			len(activeJobs), activeJobs.ToString(),
			len(successfulJobs), successfulJobs.ToString(),
			len(failedJobs), failedJobs.ToString(),
		)

	*/

	switch {
	case status.Phase == v1alpha1.PhaseUninitialized ||
		status.Phase == v1alpha1.PhaseSuccess ||
		status.Phase == v1alpha1.PhaseFailed:
		return status.Lifecycle

	case len(failedJobs) > 0:
		// A job has failed during execution.
		return v1alpha1.Lifecycle{
			Phase:  v1alpha1.PhaseFailed,
			Reason: failedJobs.ToString(),
		}

	case len(successfulJobs) == allJobs:
		// All jobs are successfully completed
		return v1alpha1.Lifecycle{
			Phase:  v1alpha1.PhaseSuccess,
			Reason: successfulJobs.ToString(),
		}

	case len(activeJobs)+len(successfulJobs) == allJobs:
		// All jobs are created, and at least one is still running
		return v1alpha1.Lifecycle{
			Phase:  v1alpha1.PhaseRunning,
			Reason: activeJobs.ToString(),
		}

	case status.Phase == v1alpha1.PhasePending:
		// Not all Jobs are constructed created
		return v1alpha1.Lifecycle{
			Phase:  v1alpha1.PhasePending,
			Reason: "Jobs are not yet created",
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

		panic("this should never happen")
	}

	/*

		// There are still pending jobs to be created
		return v1alpha1.Lifecycle{
			Kind:   "Cluster",
			Name:   cluster.GetName(),
			Phase:  v1alpha1.PhasePending,
			Reason: "waiting for jobs to start",
		}

	*/

	// TODO: validate the transition. For example, we cannot go from running to pending
}
