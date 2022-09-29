/*
Copyright 2022 ICS-FORTH.

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

func ValidateScheduler(instances int, sch *SchedulerSpec) error {
	if instances < 1 {
		return errors.Errorf("scheduling requires at least one instance")
	}

	// Cron and Timeline can be active at the same time.
	// However, both Cron and Timeline can be used in conjuction with Events.
	if sch.Cron != nil && sch.Timeline != nil {
		return errors.Errorf("cron and timeline distribution cannot be activated in paralle")
	}

	// cron
	if cronspec := sch.Cron; cronspec != nil {
		if _, err := cron.ParseStandard(*cronspec); err != nil {
			return errors.Wrapf(err, "invalid schedule %q", *cronspec)
		}
	}

	// event
	if conditions := sch.Event; conditions != nil {
		if err := ValidateExpr(conditions); err != nil {
			return errors.Wrapf(err, "conditions error")
		}
	}

	// timeline
	if timeline := sch.Timeline; timeline != nil {
		if err := ValidateDistribution(timeline.DistributionSpec); err != nil {
			return errors.Wrapf(err, "conditions error")
		}
	}

	return nil
}

func ValidateDistribution(dist *DistributionSpec) error {
	switch dist.Distribution {
	case "constant":
		return nil

	case "uniform":
		return nil

	case "zipfian":
		return nil

	case "histogram":
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
