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

import (
	"regexp"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/pkg/errors"
)

// ConditionalExpr is a source of information about whether the state of the workflow after a given time is correct or not.
// This is needed because some scenarios may run in infinite-horizons.
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

func (c *ConditionalExpr) IsZero() bool {
	return c == nil || !c.HasStateExpr() || !c.HasMetricsExpr()
}

func (c *ConditionalExpr) HasMetricsExpr() bool {
	return c != nil && c.Metrics != ""
}

func (c *ConditionalExpr) HasStateExpr() bool {
	return c != nil && c.State != ""
}

/*
	Validate State Expressions
*/

var sprigFuncMap = sprig.TxtFuncMap() // a singleton for better performance

type ExprState string

func (query ExprState) Parse() (*template.Template, error) {
	t, err := template.New("").Funcs(sprigFuncMap).Option("missingkey=error").Parse(string(query))
	if err != nil {
		return nil, errors.Wrapf(err, "parsing error")
	}

	return t, nil
}

/*
	Validate Metrics Expressions
*/

type ExprMetrics string

func (query ExprMetrics) Parse() ([]string, error) {
	matches := ExprMetricsValidator.FindStringSubmatch(string(query))

	if len(matches) == 0 {
		return nil, errors.Errorf(`erroneous query '%s'. 
		Examples:
			- 'avg() of query(wpFnYRwGk/2/bitrate, 15m, now) is below(14)'
			- 'avg() of query(wpFnYRwGk/2/bitrate, 15m, now) is below(0.4)'
			- 'avg() of query(wpFnYRwGk/2/bitrate, 15m, now) is novalue()'
			- 'avg() of query(wpFnYRwGk/2/bitrate, 15m, now) is withinrange(4, 88)'
			- 'avg() of query(wpFnYRwGk/2/bitrate, 15m, now) is withinrange(4, 88) for (1m)'
			- 'avg() of query(wpFnYRwGk/2/bitrate, 15m, now) is withinrange(4, 88) for (1m) every(1m)'

		Prepare your expressions at: https://regex101.com/r/sIspYb/1/`, query)
	}

	return matches, nil
}

// ExprMetricsValidator expressions evaluated with https://regex101.com/r/ZB8rPs/1
var ExprMetricsValidator = regexp.MustCompile(`(?m)^(?P<reducer>\w+)\(\)\s+of\s+query\((?P<dashboardUID>\w+)\/(?P<panelID>\d+)\/(?P<metric>\w+),\s+(?P<from>\w+),\s+(?P<to>\w+)\)\s+is\s+(?P<evaluator>\w+)\((?P<params>\d*[\.,\s]*\d*)\)\s*(for\s+\((?P<for>\w+)\))*\s*(every\((?P<every>\w+)\))*\s*$`)
