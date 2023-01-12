/*
Copyright 2022-2023 ICS-FORTH.

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

package v1alpha1

import (
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
)

func ValidateTolerate(_ *TolerateSpec) error {
	return nil
}

func ValidateExpr(expr *ConditionalExpr) error {
	if expr.IsZero() {
		return nil
	}

	if expr.HasStateExpr() {
		if _, err := expr.State.GoValuate(DefaultClassifier{}); err != nil {
			return errors.Wrapf(err, "wrong state expr")
		}
	}

	if expr.HasMetricsExpr() {
		if _, err := expr.Metrics.Parse(); err != nil {
			return errors.Wrapf(err, "wrong metrics expr")
		}
	}

	return nil
}

func ValidateTaskScheduler(sch *TaskSchedulerSpec) error {
	var merr *multierror.Error

	var enabledPolicies uint8

	// sequential
	if sch.Sequential != nil && *sch.Sequential {
		enabledPolicies++
	}

	// cron
	if cronspec := sch.Cron; cronspec != nil {
		enabledPolicies++

		if _, err := cron.ParseStandard(*cronspec); err != nil {
			merr = multierror.Append(merr, errors.Wrapf(err, "CronError"))
		}
	}

	// event
	if conditions := sch.Event; conditions != nil {
		enabledPolicies++

		if err := ValidateExpr(conditions); err != nil {
			merr = multierror.Append(merr, errors.Wrapf(err, "EventError"))
		}
	}

	// timeline
	if timeline := sch.Timeline; timeline != nil {
		enabledPolicies++

		if err := ValidateDistribution(timeline.DistributionSpec); err != nil {
			merr = multierror.Append(merr, errors.Wrapf(err, "TimelineError"))
		}
	}

	// check for conflicts
	if enabledPolicies != 1 {
		merr = multierror.Append(merr, errors.Errorf("Expected 1 scheduling policy but got %d", enabledPolicies))
	}

	return merr.ErrorOrNil()
}

func ValidateDistribution(dist *DistributionSpec) error {
	switch dist.Name {
	case DistributionConstant:
		return nil

	case DistributionUniform:
		return nil

	case DistributionZipfian:
		return nil

	case DistributionHistogram:
		return nil

	case DistributionDefault:
		return nil
	}
	/* TODO: continue with the other distributions */

	return nil
}

// ValidatePlacement validates the placement policy. However, because it may involve references to other
// services, the validation requires a list of the defined actions.
func ValidatePlacement(policy *PlacementSpec, callIndex map[string]*Action) error {
	// Validate the name of the references nodes.
	if policy.Nodes != nil {
		// TODO: add logic
	}

	// Validate the presence of the references actions.
	if policy.ConflictsWith != nil {
		for _, ref := range policy.ConflictsWith {
			action, exists := callIndex[ref]
			if !exists {
				return errors.Errorf("referenced action '%s' does not exist. ", ref)
			}

			if action.ActionType != ActionCluster && action.ActionType != ActionService {
				return errors.Errorf("referenced action '%s' is type '%s'. Expected: '%s|%s'",
					ref, action.ActionType, ActionCluster, ActionService)
			}
		}
	}

	return nil
}
