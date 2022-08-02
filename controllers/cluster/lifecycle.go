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
	"github.com/carv-ics-forth/frisbee/controllers/common/expressions"
	"github.com/carv-ics-forth/frisbee/controllers/common/lifecycle"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// calculateLifecycle returns the update lifecycle of the cluster.
func (r *Controller) calculateLifecycle(cr *v1alpha1.Cluster) {
	gs := r.view

	// Step 1. Skip any CR which are already completed, or uninitialized.
	if cr.Status.Lifecycle.Phase.Is(v1alpha1.PhaseUninitialized, v1alpha1.PhaseSuccess, v1alpha1.PhaseFailed) {
		return
	}

	// Step 3. Check if "Until" conditions are met.
	if !cr.Spec.Until.IsZero() {
		eval := expressions.Condition{Expr: cr.Spec.Until}

		if eval.IsTrue(gs, cr) {

			cr.Status.Lifecycle.Phase = v1alpha1.PhaseRunning
			cr.Status.Lifecycle.Reason = "UntilCondition"
			cr.Status.Lifecycle.Message = eval.Info

			meta.SetStatusCondition(&cr.Status.Lifecycle.Conditions, metav1.Condition{
				Type:    v1alpha1.ConditionAllJobsAreScheduled.String(),
				Status:  metav1.ConditionTrue,
				Reason:  "UntilCondition",
				Message: eval.Info,
			})

			suspend := true
			cr.Spec.Suspend = &suspend

			return
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

			return
		}

		// A side effect of "Until" is that queued jobs will be reused,
		// until the conditions are met. In that sense, they resemble mostly a pool of jobs
		// rather than e queue.
		cr.Status.Lifecycle.Phase = v1alpha1.PhasePending
		cr.Status.Lifecycle.Reason = "SpawnUntilEvent"
		cr.Status.Lifecycle.Message = "Assertion is not yet satisfied."

		return
	}

	// Step 4. Check if scheduling goes as expected.
	queuedJobs := len(cr.Status.QueuedJobs)

	if lifecycle.GroupedJobs(queuedJobs, gs, &cr.Status.Lifecycle, cr.Spec.Tolerate) {
		return
	}

	panic(errors.Errorf(`unhandled lifecycle conditions.
		current: %v,
		total: %d,
		pendingJobs: %s,
		runningJobs: %s,
		successfulJobs: %s,
		failedJobs: %s
	`, cr.Status.Lifecycle, queuedJobs, gs.ListPendingJobs(), gs.ListRunningJobs(), gs.ListSuccessfulJobs(), gs.ListFailedJobs()))
}
