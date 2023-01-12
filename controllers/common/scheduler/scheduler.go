/*
Copyright 2021-2023 ICS-FORTH.

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

package scheduler

import (
	"fmt"
	"time"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/pkg/expressions"
	"github.com/carv-ics-forth/frisbee/pkg/lifecycle"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const DefaultTickDelay = 5 * time.Second

type Parameters struct {
	// ScheduleSpec is the scheduling options
	ScheduleSpec *v1alpha1.TaskSchedulerSpec

	// State is the real state of the system.
	State lifecycle.Classifier

	// LastScheduleTime is the time the controller last scheduled an object.
	LastScheduleTime metav1.Time

	//
	// Used for Timeline mode
	//

	// ExpectedTime is the evaluation of a timeline distribution defined in the ScheduleSpec.
	ExpectedTimeline v1alpha1.Timeline

	//
	// Used for Sequential mode
	//

	// JobName is used a prefix for finding children tasks.
	// We assume the standard naming pattern Job-Task.
	JobName string

	// ScheduledJobs normally points to the next QueuedJobs.
	// In this case, we use it in conjunction with JobName to find the children's name,
	// and from that to extract the status of the child.
	ScheduledJobs int `json:"scheduledJobs,omitempty"`
}

// Schedule calculate the next scheduled run, and whether we've got a run that we haven't processed yet  (or anything we missed).
// If we've missed a run, and we're still within the deadline to start it, we'll need to run a job.
// time-based and event-driven scheduling can be used in conjunction.
func Schedule(log logr.Logger, obj client.Object, params Parameters) (goToNextJob bool, nextTick time.Time, err error) {
	// no scheduling constraint.
	if params.ScheduleSpec == nil {
		// no constraints. start everything as fast as possible.
		return true, time.Time{}, nil
	}

	// due to the delay between API server and the controller, and the lack of local state,
	// we may serve a request twice if the next reconciliation cycle starts too fast.
	// In that case, we may violate sequential and event-based scheduling.
	// Until we find a better solution, we simply delay the next tick.
	nextTick = time.Now().Add(DefaultTickDelay)

	if params.ScheduleSpec.StartingDeadlineSeconds == nil {
		delay := time.Duration(*params.ScheduleSpec.StartingDeadlineSeconds) * time.Second
		nextTick = time.Now().Add(delay)
	}

	// Sequential scheduling
	if params.ScheduleSpec.Sequential != nil {
		// if nothing is running, start a new job
		if params.ScheduledJobs == -1 {
			return true, nextTick, nil
		}

		// if a job is running, make sure that it is complete
		lastJob := fmt.Sprintf("%s-%d", params.JobName, params.ScheduledJobs)

		if params.State.IsSuccessful(lastJob) {
			return true, nextTick, nil
		}

		// otherwise, do not schedule anything
		return false, nextTick, nil
	}

	// Cron-based scheduling
	if params.ScheduleSpec.Cron != nil {
		missed, cronTick, err := cronWithDeadline(log, obj, params)

		return !missed.IsZero(), cronTick, err
	}

	// Timeline-based scheduling
	if params.ScheduleSpec.Timeline != nil {
		missed, fixedTick, err := timelineWithDeadline(log, obj, params)

		return !missed.IsZero(), fixedTick, err
	}

	// Event-based scheduling
	if !params.ScheduleSpec.Event.IsZero() {
		eval := expressions.Condition{Expr: params.ScheduleSpec.Event}

		return eval.IsTrue(&params.State, obj), nextTick, nil
	}

	panic("this should never happen")
}

func cronWithDeadline(_ logr.Logger, obj client.Object, params Parameters) (lastMissed time.Time, next time.Time, err error) {
	timeline, err := cron.ParseStandard(*params.ScheduleSpec.Cron)
	if err != nil {
		return time.Time{}, time.Time{}, errors.Wrapf(err, "unparseable timeline %q", *params.ScheduleSpec.Cron)
	}

	lastMissed, next, err = getNextScheduleTime(obj.GetCreationTimestamp().Time, timeline, params)
	if err != nil {
		return lastMissed, next, errors.Wrapf(err, "scheduling error")
	}

	/*
		deadline := params.ScheduleSpec.StartingDeadlineSeconds
		if !lastMissed.IsZero() && !honorDeadline(log, lastMissed, deadline) {
			return lastMissed, next, errors.Errorf("scheduling violation. deadline of '%d' seconds is too strict.", *deadline)
		}
	*/

	return lastMissed, next, nil
}

func timelineWithDeadline(_ logr.Logger, obj client.Object, params Parameters) (lastMissed time.Time, next time.Time, err error) {
	timeline := params.ExpectedTimeline

	lastMissed, next, err = getNextScheduleTime(obj.GetCreationTimestamp().Time, timeline, params)
	if err != nil {
		return lastMissed, next, errors.Wrapf(err, "timeline error")
	}

	/*
		deadline := params.ScheduleSpec.StartingDeadlineSeconds
		if !lastMissed.IsZero() && !honorDeadline(log, lastMissed, deadline) {
			return lastMissed, next, errors.Errorf("scheduling violation. deadline of '%d' seconds is too strict.", *deadline)
		}
	*/

	return lastMissed, next, nil
}

// Timeline describes a job's duty cycle.
type Timeline interface {
	// Next returns the next activation time, later than the given time.
	// Next is invoked initially, and then each time the job is run.
	Next(time.Time) time.Time
}

// getNextScheduleTime figure out the next times that we need to create jobs at (or anything we missed).
//
// We'll start calculating appropriate times from our last run, or the creation
// of the CronJob if we can't find a last run. This gets the time of next schedule
// after last scheduled and before now.
//
// If there are too many missed runs, and we don't have any deadlines set, we'll
// bail so that we don't cause issues on controller restarts or wedges.
// Otherwise, we'll just return the missed runs (of which we'll just use the latest),
// and the next run, so that we can know when it's time to reconcile again.
func getNextScheduleTime(earliest time.Time, timeline Timeline, params Parameters) (lastMissed time.Time, next time.Time, err error) {
	now := time.Now()

	var earliestTime time.Time

	if params.LastScheduleTime.IsZero() {
		// If none found, then this is either a recently created cronJob,
		// or the active/completed info was somehow lost (contract for status
		// in kubernetes says it may need to be recreated), or that we have
		// started a job, but have not noticed it yet (distributed systems can
		// have arbitrary delays).  In any case, use the creation time of the
		// object as last known start time.
		earliestTime = earliest
	} else {
		// for optimization purposes, cheat a bit and start from our last observed run time
		// we could reconstitute this here, but there's not much point, since we've
		// just updated it.
		earliestTime = params.LastScheduleTime.Time
	}

	if params.ScheduleSpec.StartingDeadlineSeconds != nil {
		// controller is not going to schedule anything below this point
		schedulingDeadline := now.Add(-time.Second * time.Duration(*params.ScheduleSpec.StartingDeadlineSeconds))

		if schedulingDeadline.After(earliestTime) {
			earliestTime = schedulingDeadline
		}
	}

	if earliestTime.After(now) {
		// the earliest time is later than now.
		// return the next activation time (used for re-queuing the request)
		return time.Time{}, timeline.Next(now), nil
	}

	starts := 0

	for t := timeline.Next(earliestTime); !t.After(now); t = timeline.Next(t) {
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

	return lastMissed, timeline.Next(now), nil
}

/*
func honorDeadline(log logr.Logger, lastMissed time.Time, deadline *int64) bool {
	// if there is a missed run, make sure we're not too late to start the run
	tooLate := false

	if deadline != nil {
		skew := lastMissed.Add(time.Duration(*deadline) * time.Second)

		log.Info("MissedSchedule", "skew", skew)

		tooLate = skew.Before(time.Now())
	}

	return tooLate
}
*/
