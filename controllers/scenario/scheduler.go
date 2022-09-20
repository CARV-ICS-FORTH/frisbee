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

package scenario

import (
	"time"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/pkg/structure"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NextJobs returns a list of jobs that meet the logical and time constraints.
// That is, either the job has no dependencies, or the dependencies are met.
//
// It is possible for the logical dependencies to be met, but the timeout not yet expired.
// If at least one action exists, when the workflow is updated it will trigger another reconciliation cycle.
// However, if there are no actions, the workflow will call the reconciliation cycle, and we will miss the
// next timeout. To handle this scenario, we have to requeue the request with the given duration.
// In this case, the given duration is the nearest expected timeout.
func (r *Controller) NextJobs(cr *v1alpha1.Scenario) (runNext []v1alpha1.Action, nextCycle time.Time, err error) {
	timeOK := func(deps *v1alpha1.WaitSpec) bool {
		if dur := deps.After; dur != nil {
			cur := metav1.Now()
			deadline := cr.GetCreationTimestamp().Add(dur.Duration)

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

	// check what actions are eligible for execution in this cycle.
	all := cr.Spec.Actions
	scheduled := cr.Status.ScheduledJobs

	for _, action := range all {
		// ignore scheduled jobs
		if structure.ContainsStrings(scheduled, action.Name) {
			continue
		}

		// a job is eligible for scheduling if there are no dependencies, or if defined dependencies are satisfied.
		deps := action.DependsOn
		if deps == nil {
			runNext = append(runNext, action)
		} else {
			// check a dependent "running" is not already terminated, as it will cause the scenario
			// to loop forever
			for _, dep := range deps.Running {
				if r.view.IsSuccessful(dep) || r.view.IsFailed(dep) {
					err := errors.Errorf("action '%s' has a Running dependency on completed job '%s'", action.Name, dep)

					return nil, time.Now(), err
				}
			}

			if r.view.IsSuccessful(deps.Success...) && r.view.IsRunning(deps.Running...) && timeOK(deps) {
				// conditions are met
				runNext = append(runNext, action)
			}
		}
	}

	return runNext, nextCycle, nil
}
