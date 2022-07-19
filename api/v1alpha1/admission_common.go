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

func ValidateTolerate(tol *TolerateSpec) error {
	return nil
}

func ValidateExpr(expr *ConditionalExpr) error {
	if expr.IsZero() {
		return nil
	}

	if expr.HasStateExpr() {
		if _, err := expr.State.Parse(); err != nil {
			return errors.Wrapf(err, "Invalid state expr")
		}
	}

	if expr.HasMetricsExpr() {
		if _, err := expr.Metrics.Parse(); err != nil {
			return errors.Wrapf(err, "Invalid metrics expr")
		}
	}

	return nil
}

func ValidateScheduler(sch *SchedulerSpec) error {

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

	return nil
}
