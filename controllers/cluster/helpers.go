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
	"time"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

func getNextSchedule(cluster *v1alpha1.Cluster, cur time.Time) (lastMissed time.Time, next time.Time, err error) {
	scheduler := cluster.Spec.Schedule

	// if there is no scheduler defined, start the job immediately.
	if scheduler == nil {
		return time.Now(), time.Time{}, nil
	}

	sched, err := cron.ParseStandard(scheduler.Cron)
	if err != nil {
		return time.Time{}, time.Time{}, errors.Wrapf(err, "unparseable schedule %q", scheduler.Cron)
	}

	// for optimization purposes, cheat a bit and start from our last observed run time
	// we could reconstitute this here, but there's not much point, since we've
	// just updated it.
	var earliestTime time.Time
	if cluster.Status.LastScheduleTime != nil {
		earliestTime = cluster.Status.LastScheduleTime.Time
	} else {
		earliestTime = cluster.ObjectMeta.CreationTimestamp.Time
	}
	if scheduler.StartingDeadlineSeconds != nil {
		// controller is not going to schedule anything below this point
		schedulingDeadline := cur.Add(-time.Second * time.Duration(*scheduler.StartingDeadlineSeconds))

		if schedulingDeadline.After(earliestTime) {
			earliestTime = schedulingDeadline
		}
	}
	if earliestTime.After(cur) {
		return time.Time{}, sched.Next(cur), nil
	}

	starts := 0
	for t := sched.Next(earliestTime); !t.After(cur); t = sched.Next(t) {
		lastMissed = t
		// An object might miss several starts. For example, if
		// controller gets wedged on Friday at 5:01pm when everyone has
		// gone home, and someone comes in on Tuesday AM and discovers
		// the problem and restarts the controller, then all the hourly
		// jobs, more than 80 of them for one hourly scheduledJob, should
		// all start running with no further intervention (if the scheduledJob
		// allows concurrency and late starts).
		//
		// However, if there is a bug somewhere, or incorrect clock
		// on controller's server or apiservers (for setting creationTimestamp)
		// then there could be so many missed start times (it could be off
		// by decades or more), that it would eat up all the CPU and memory
		// of this controller. In that case, we want to not try to list
		// all the missed start times.
		starts++
		if starts > 100 {
			// We can't get the most recent times so just return an empty slice
			return time.Time{}, time.Time{},
				errors.New("too many missed start times (> 100). Set or decrease .spec.startingDeadlineSeconds or check clock skew.")
		}
	}
	return lastMissed, sched.Next(cur), nil
}

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
			Phase:  v1alpha1.PhaseInitialized,
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
