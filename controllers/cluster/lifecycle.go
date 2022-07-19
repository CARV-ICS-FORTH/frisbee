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

package cluster

import (
	"fmt"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common/lifecycle"
	"github.com/carv-ics-forth/frisbee/controllers/common/lifecycle/check"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// calculateLifecycle returns the update lifecycle of the cluster.
func calculateLifecycle(cluster *v1alpha1.Cluster, gs lifecycle.ClassifierReader) v1alpha1.Lifecycle {
	cycle := cluster.Status.Lifecycle

	// Step 1. Skip any CR which are already completed, or uninitialized.
	if cycle.Phase.Is(v1alpha1.PhaseUninitialized, v1alpha1.PhaseSuccess, v1alpha1.PhaseFailed) {
		return cycle
	}

	// Step 2. Check if failures violate cluster's toleration.
	if tolerate := cluster.Spec.Tolerate; tolerate != nil {
		if check.TolerationExceeded(tolerate, gs, &cycle) {
			return cycle
		}
	}

	// Step 3. Check if "Until" conditions are met.
	if until := cluster.Spec.Until; until != nil {
		if check.UntilConditionIsMet(until, gs, cluster, &cycle) {
			suspend := true
			cluster.Spec.Suspend = &suspend

			return cycle
		}

		// Event used in conjunction with "Until", instance act as a maximum bound.
		// If the maximum instances are reached before the Until conditions, we assume that
		// the experiment never converges, and it fails.
		if cluster.Spec.MaxInstances > 0 && (cluster.Status.ScheduledJobs > cluster.Spec.MaxInstances) {
			msg := fmt.Sprintf(`Cluster [%s] has reached Max instances [%d] before Until conditions are met.
			Abort the experiment as it too flaky to accept. You can retry without defining instances.`,
				cluster.GetName(), cluster.Spec.MaxInstances)

			cycle = v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseFailed,
				Reason:  "MaxInstancesReached",
				Message: msg,
			}

			meta.SetStatusCondition(&cycle.Conditions, metav1.Condition{
				Type:    v1alpha1.ConditionJobUnexpectedTermination.String(),
				Status:  metav1.ConditionTrue,
				Reason:  "MaxInstancesReached",
				Message: msg,
			})

			return cycle
		}

		// A side effect of "Until" is that queued jobs will be reused,
		// until the conditions are met. In that sense, they resemble mostly a pool of jobs
		// rather than e queue.
		cycle = v1alpha1.Lifecycle{
			Phase:   v1alpha1.PhasePending,
			Reason:  "SpawnUntilEvent",
			Message: "Assertion is not yet satisfied.",
		}

		return cycle
	}

	// Step 4. Check if scheduling goes as expected.
	queuedJobs := len(cluster.Status.QueuedJobs)

	if check.ScheduledJobs(queuedJobs, gs, &cycle) {
		return cycle
	}

	panic(errors.Errorf(`unhandled lifecycle conditions.
		current: %v,
		total: %d,
		pendingJobs: %s,
		runningJobs: %s,
		successfulJobs: %s,
		failedJobs: %s
	`, cycle, queuedJobs, gs.PendingJobsList(), gs.RunningJobsList(), gs.SuccessfulJobsList(), gs.FailedJobsList()))
}
