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

package v1alpha1

// TaskSchedulerSpec determines the conditions for creating new tasks of a Job.
// The scheduler will schedule up to spec.GenerateObjectFromTemplate.Instances or spec.GenerateObjectFromTemplate.Until.
type TaskSchedulerSpec struct {
	// Sequential schedules a new task once the previous task is complete.
	Sequential *bool `json:"sequential,omitempty"`

	// StartingDeadlineSeconds is an optional deadline in seconds for starting the job if it misses scheduled
	// time for any reason. if we miss this deadline, we'll just wait till the next scheduled time
	//
	// +optional
	StartingDeadlineSeconds *int64 `json:"startingDeadlineSeconds,omitempty"`

	// Cron defines a cron job rule.
	//
	// Some rule examples:
	// "0 30 * * * *" means to "Every hour on the half hour"
	// "@hourly"      means to "Every hour"
	// "@every 1h30m" means to "Every hour thirty"
	//
	// More rule info: https://godoc.org/github.com/robfig/cron
	//
	// +optional
	Cron *string `json:"cron,omitempty"`

	// Timeline schedules new tasks deterministically, based on predefined times that honors the underlying distribution.
	// Multiple tasks may run concurrently.
	// +optional
	Timeline *TimelineDistributionSpec `json:"timeline,omitempty"`

	// Event schedules new tasks in a non-deterministic manner, based on system-driven events.
	// Multiple tasks may run concurrently.
	// +optional
	Event *ConditionalExpr `json:"event,omitempty"`
}

// DefaultStartingDeadlineSeconds hints to abort the experiment if the schedule is skewed more than 1 minuted.
var DefaultStartingDeadlineSeconds = int64(60)
