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

package cascade

import (
	"fmt"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/pkg/expressions"
	"github.com/carv-ics-forth/frisbee/pkg/lifecycle"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// updateLifecycle returns the update lifecycle of the cascade.
func (r *Controller) updateLifecycle(cr *v1alpha1.Cascade) bool {
	// Step 1. Skip any CR which are already completed, or uninitialized.
	if cr.Status.Phase.Is(v1alpha1.PhaseUninitialized, v1alpha1.PhaseSuccess, v1alpha1.PhaseFailed) {
		return false
	}

	// Step 2. Check if "SuspendWhen" conditions are met.
	if !cr.Spec.SuspendWhen.IsZero() {
		if meta.IsStatusConditionTrue(cr.Status.Conditions, v1alpha1.ConditionAllJobsAreScheduled.String()) {
			// The Until condition is already handled, and we are in the Running Phase.
			// From now on, the lifecycle depends on the progress of the already scheduled jobs.
			totalJobs := cr.Status.ScheduledJobs + 1
			return lifecycle.GroupedJobs(totalJobs, r.view, &cr.Status.Lifecycle, nil)
		}

		eval := expressions.Condition{Expr: cr.Spec.SuspendWhen}
		if eval.IsTrue(r.view, cr) {
			cr.Status.Lifecycle.Phase = v1alpha1.PhaseRunning
			cr.Status.Lifecycle.Reason = "UntilCondition"
			cr.Status.Lifecycle.Message = eval.Info

			meta.SetStatusCondition(&cr.Status.Lifecycle.Conditions, metav1.Condition{
				Type:    v1alpha1.ConditionAllJobsAreScheduled.String(),
				Status:  metav1.ConditionTrue,
				Reason:  "UntilCondition",
				Message: eval.Info,
			})

			// prevent the parent from spawning new jobs.
			suspend := true
			cr.Spec.Suspend = &suspend

			return true
		}

		// Event used in conjunction with "Until", instance act as a maximum bound.
		// If the maximum instances are reached before the Until conditions, we assume that
		// the experiment never converges, and it fails.
		maxJobs := cr.Spec.MaxInstances

		if maxJobs > 0 && (cr.Status.ScheduledJobs > maxJobs) {
			msg := fmt.Sprintf(`Resource [%s] has reached Max instances [%d] before Until conditions are met.
			Abort the experiment as it too flaky to accept. You can retry without defining instances.`,
				cr.GetName(), maxJobs)

			cr.Status.Lifecycle.Phase = v1alpha1.PhaseFailed
			cr.Status.Lifecycle.Reason = "MaxInstancesReached"
			cr.Status.Lifecycle.Message = msg

			meta.SetStatusCondition(&cr.Status.Lifecycle.Conditions, metav1.Condition{
				Type:    v1alpha1.ConditionJobUnexpectedTermination.String(),
				Status:  metav1.ConditionTrue,
				Reason:  "MaxInstancesReached",
				Message: msg,
			})

			return true
		}

		// A side effect of "Until" is that queued jobs will be reused,
		// until the conditions are met. In that sense, they resemble mostly a pool of jobs
		// rather than e queue.
		cr.Status.Lifecycle.Phase = v1alpha1.PhasePending
		cr.Status.Lifecycle.Reason = "SpawnUntilEvent"
		cr.Status.Lifecycle.Message = "Assertion is not yet satisfied."

		return true
	}

	// Step 4. Check if scheduling goes as expected.
	totalJobs := len(cr.Status.QueuedJobs)

	return lifecycle.GroupedJobs(totalJobs, r.view, &cr.Status.Lifecycle, nil)
}
