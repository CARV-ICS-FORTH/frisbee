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
	"strings"
	"time"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/utils/lifecycle"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

// ValidateDAG validates the execution workflow.
// 1. Ensures that action names are qualified (since they are used as generators to jobs)
// 2. Ensures that there are no two actions with the same name.
// 3. Ensure that dependencies point to a valid action.
// 4. Ensure that macros point to a valid action
func ValidateDAG(list v1alpha1.ActionList) error {
	index := make(map[string]*v1alpha1.Action)

	for i, action := range list {
		if errs := validation.IsQualifiedName(action.Name); len(errs) != 0 {
			err := errors.New(strings.Join(errs, "; "))

			return errors.Wrapf(err, "invalid actioname %s", action.Name)
		}

		index[action.Name] = &list[i]
	}

	successOK := func(deps *v1alpha1.WaitSpec) bool {
		for _, dep := range deps.Success {
			_, ok := index[dep]
			if !ok {
				return false
			}
		}

		return true
	}

	runningOK := func(deps *v1alpha1.WaitSpec) bool {
		for _, dep := range deps.Running {
			_, ok := index[dep]
			if !ok {
				return false
			}
		}

		return true
	}

	for _, action := range list {
		if deps := action.DependsOn; deps != nil {
			if !successOK(deps) || !runningOK(deps) {
				return errors.Errorf("invalid dependency on action %s", action.Name)
			}
		}
	}

	// TODO:
	// 1) add validation for templateRef
	// 2) make validation as webhook so to validate the experiment before it begins.

	return nil
}

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
	gs lifecycle.Classifier,
	scheduled map[string]metav1.Time,
) (v1alpha1.ActionList, time.Time) {
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
