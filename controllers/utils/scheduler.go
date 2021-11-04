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

package utils

import (
	"time"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetNextScheduleTime figure out the next times that we need to create jobs at (or anything we missed).
//
// We'll start calculating appropriate times from our last run, or the creation
// of the CronJob if we can't find a last run. This gets the time of next schedule
// after last scheduled and before now.
//
// If there are too many missed runs, and we don't have any deadlines set, we'll
// bail so that we don't cause issues on controller restarts or wedges.
// Otherwise, we'll just return the missed runs (of which we'll just use the latest),
// and the next run, so that we can know when it's time to reconcile again.
func GetNextScheduleTime(
	obj metav1.Object,
	scheduler *v1alpha1.SchedulerSpec,
	lastScheduleTime *metav1.Time,
) (lastMissed time.Time, next time.Time, err error) {
	cur := time.Now()

	// start the job immediately if there is no defined scheduler.
	if scheduler == nil {
		return time.Now(), time.Time{}, nil
	}

	sched, err := cron.ParseStandard(scheduler.Cron)
	if err != nil {
		return time.Time{}, time.Time{}, errors.Wrapf(err, "unparseable schedule %q", scheduler.Cron)
	}

	var earliestTime time.Time

	if lastScheduleTime != nil {
		// for optimization purposes, cheat a bit and start from our last observed run time
		// we could reconstitute this here, but there's not much point, since we've
		// just updated it.
		earliestTime = lastScheduleTime.Time
	} else {
		// If none found, then this is either a recently created cronJob,
		// or the active/completed info was somehow lost (contract for status
		// in kubernetes says it may need to be recreated), or that we have
		// started a job, but have not noticed it yet (distributed systems can
		// have arbitrary delays).  In any case, use the creation time of the
		// object as last known start time.
		earliestTime = obj.GetCreationTimestamp().Time
	}

	if scheduler.StartingDeadlineSeconds != nil {
		// controller is not going to schedule anything below this point
		schedulingDeadline := cur.Add(-time.Second * time.Duration(*scheduler.StartingDeadlineSeconds))

		if schedulingDeadline.After(earliestTime) {
			earliestTime = schedulingDeadline
		}
	}

	if earliestTime.After(cur) {
		// the earliest time is later than now.
		// return the next activation time (used for re-queuing the request)
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
				errors.New("too many missed start times (> 100). Set or decrease .spec.startingDeadlineSeconds or check clock skew")
		}
	}

	return lastMissed, sched.Next(cur), nil
}
