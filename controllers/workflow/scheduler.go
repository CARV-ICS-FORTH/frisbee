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

package workflow

import (
	"time"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/utils/lifecycle"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// isJobInScheduledList take a job and checks if activeJobs has a job with the same
// name and namespace.
func isJobInScheduledList(name string, scheduledJobs map[string]metav1.Time) bool {
	_, ok := scheduledJobs[name]

	return ok
}

// GetNextLogicalJob returns a list of jobs that meet the logical and time constraints.
// That is, either the job has no dependencies, or the dependencies are met.
//
// It is possible for the logical dependencies to be met, but the timeout not yet expired.
// If at least one action exists, when the workflow is updated it will trigger another reconciliation cycle.
// However, if there are no actions, the workflow will stop the reconciliation cycle, and we will miss the
// next timeout. To handle this scenario, we have to requeue the request with the given duration.
// In this case, the given duration is the nearest expected timeout.
func GetNextLogicalJob(obj metav1.Object, all v1alpha1.ActionList, gs lifecycle.Classifier, scheduled map[string]metav1.Time) (v1alpha1.ActionList, time.Time) {
	var candidates v1alpha1.ActionList

	var nextCycle time.Time

	successOK := func(deps *v1alpha1.WaitSpec) bool {
		for _, dep := range deps.Success {
			if !gs.IsSuccessful(dep) {
				return false
			}
		}

		return true
	}

	runningOK := func(deps *v1alpha1.WaitSpec) bool {
		for _, dep := range deps.Running {
			if !gs.IsRunning(dep) {
				return false
			}
		}

		return true
	}

	timeOK := func(deps *v1alpha1.WaitSpec) bool {
		if dur := deps.After; dur != nil {
			cur := metav1.Now()
			deadline := obj.GetCreationTimestamp().Time.Add(dur.Duration)

			// the deadline has expired.
			if deadline.Before(cur.Time) {
				return true
			}

			// calculate time to the next shortest timeout
			if nextCycle.IsZero() {
				nextCycle = deadline
			} else if deadline.Before(nextCycle) {
				nextCycle = deadline
			}

			return false
		}

		return true
	}

	for _, action := range all {
		if gs.IsActive(action.Name) || isJobInScheduledList(action.Name, scheduled) {
			// Not starting action because it is already processed.

			// logrus.Warnf("Ignore action %s since it is already processed", action.Name)
			continue
		}

		if deps := action.DependsOn; deps != nil {
			if !successOK(deps) || !runningOK(deps) || !timeOK(deps) {
				// Not starting action because the dependencies are not met.

				// logrus.Warnf("Ignore action %s because dependency are not met", action.Name)
				continue
			}
		}

		candidates = append(candidates, action)
	}

	return candidates, nextCycle
}
