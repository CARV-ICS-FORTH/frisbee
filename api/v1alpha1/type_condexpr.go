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

package v1alpha1

type ExprMetrics string

type ExprState string

// ConditionalExpr is a source of information about whether the state of the workflow after a given time is correct or not.
// This is needed because some test plans may run in infinite-horizons.
type ConditionalExpr struct {
	// Metrics set a Grafana alert that will be triggered once the condition is met.
	// Parsing:
	// Grafana URL: http://grafana/d/A2EjFbsMk/ycsb-services?editPanel=86
	// metrics: A2EjFbsMk/86/Average (Panel/Dashboard/Metric)
	//
	// +optional
	// +nullable
	Metrics ExprMetrics `json:"metrics,omitempty"`

	// State describe the runtime condition that should be met after the action has been executed
	// Shall be defined using .Lifecycle() methods. The methods account only jobs that are managed by the object.
	// +optional
	// +nullable
	State ExprState `json:"state,omitempty"`
}

func (c *ConditionalExpr) HasMetricsExpr() bool {
	return c != nil && c.Metrics != ""
}

func (c *ConditionalExpr) HasStateExpr() bool {
	return c != nil && c.State != ""
}

func (c *ConditionalExpr) IsZero() bool {
	return c == nil || !c.HasStateExpr() || !c.HasMetricsExpr()
}
