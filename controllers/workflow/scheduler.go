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
	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/utils"
)

// GetNextLogicalJob returns a list of jobs that meet the logical constraints.
// That is, either the job has no dependencies, or the dependencies are met.
//
// If the reconciliation cycle is fast enough, it is possible for the next cycle not to account for
// components that are scheduled but not yet created. To handle this race condition, we add a local state
// to keep track of what components are already scheduled.
func GetNextLogicalJob(all v1alpha1.ActionList, gs utils.LifecycleClassifier, scheduled map[string]bool) v1alpha1.ActionList {
	var filtered v1alpha1.ActionList

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

	for _, action := range all {
		if deps := action.DependsOn; deps != nil {
			if successOK(deps) && runningOK(deps) {
				_, exists := scheduled[action.Name]
				if !exists {
					filtered = append(filtered, action)
				}
			}
		} else {
			_, exists := scheduled[action.Name]
			if !exists {
				filtered = append(filtered, action)
			}
		}
	}

	for _, action := range filtered {
		scheduled[action.Name] = true
	}

	return filtered
}
