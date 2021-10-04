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

package workflow

import (
	"time"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetNextLogicalJob returns a list of jobs that meet the logical and time constraints.
// That is, either the job has no dependencies, or the dependencies are met.
//
// It is possible for the logical dependencies to be met, but the timeout not yet expired.
// If at least one action exists, when the workflow is updated it will trigger another reconciliation cycle.
// However, if there are no actions, the workflow will stop the reconciliation cycle, and we will miss the
// next timeout. To handle this scenario, we have to requeue the request with the given duration.
// In this case, the given duration is the nearest expected timeout.
func GetNextLogicalJob(
	obj metav1.Object,
	all v1alpha1.ActionList,
	gs utils.LifecycleClassifier,
	nextCycle *time.Duration,
) v1alpha1.ActionList {
	var candidates v1alpha1.ActionList

	successOK := func(deps *v1alpha1.WaitSpec) bool {
		// validate Success dependencies
		for _, dep := range deps.Success {
			if !gs.IsSuccessful(dep) {
				return false
			}
		}

		return true
	}

	runningOK := func(deps *v1alpha1.WaitSpec) bool {
		// validate Success dependencies
		for _, dep := range deps.Running {
			if !gs.IsRunning(dep) {
				return false
			}
		}

		return true
	}

	timeOK := func(deps *v1alpha1.WaitSpec) bool {
		if dur := deps.Duration; dur != nil {
			earliestTime := obj.GetCreationTimestamp().Time
			deadline := earliestTime.Add(dur.Duration)

			if metav1.Now().After(deadline) {
				return true
			}

			// calculate time to the next shortest timeout
			timeToNextTimeout := time.Until(deadline)

			if nextCycle == nil {
				*nextCycle = timeToNextTimeout
			}

			if timeToNextTimeout < *nextCycle {
				*nextCycle = timeToNextTimeout
			}

			return false
		}

		return true
	}

	for _, action := range all {
		if deps := action.DependsOn; deps != nil {
			if !successOK(deps) || !runningOK(deps) || !timeOK(deps) {
				continue
			}

			candidates = append(candidates, action)
		} else {
			candidates = append(candidates, action)
		}
	}

	return candidates
}
