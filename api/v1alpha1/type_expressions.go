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
	"reflect"
	"regexp"
	"strings"
	"text/template"

	"github.com/Knetic/govaluate"
	"github.com/Masterminds/sprig/v3"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/json"
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

func (in *ConditionalExpr) IsZero() bool {
	return in == nil || *in == (ConditionalExpr{})
}

func (in *ConditionalExpr) HasMetricsExpr() bool {
	return in != nil && in.Metrics != ""
}

func (in *ConditionalExpr) HasStateExpr() bool {
	return in != nil && in.State != ""
}

/*
	Validate State Expressions
*/

// +kubebuilder:object:generate=false

func structToLowercase(in interface{}) map[string]interface{} {
	v := reflect.ValueOf(in)
	if v.Kind() != reflect.Struct {
		return nil
	}

	vType := v.Type()

	result := make(map[string]interface{}, v.NumField())

	for i := 0; i < v.NumField(); i++ {
		name := vType.Field(i).Name
		result[strings.ToLower(name)] = v.Field(i).Interface()
	}

	return result
}

func lower(f interface{}) interface{} {
	switch f := f.(type) {
	case []interface{}:
		for i := range f {
			f[i] = lower(f[i])
		}
		return f
	case map[string]interface{}:
		lf := make(map[string]interface{}, len(f))
		for k, v := range f {
			lf[strings.ToLower(k)] = lower(v)
		}
		return lf
	default:
		return f
	}
}

var sprigFuncMap = sprig.TxtFuncMap() // a singleton for better performance

type ExprState string

// Evaluate will evaluate the expression using the golang's templates enriched with the spring func map.
func (expr ExprState) Evaluate(state interface{}) (string, error) {
	if expr == "" || state == nil {
		return "", nil
	}

	// Parse the expression
	t, err := template.New("").Funcs(sprigFuncMap).Option("missingkey=error").Parse(string(expr))
	if err != nil {
		return "", errors.Wrapf(err, "parsing error")
	}

	// Access the state fields and substitute the output.
	var out strings.Builder

	// pretty retarded way to support lower-case macros e.g, {{.inputs.parameters.}}
	// The StateAggregationFunctions is an exception as need the param to be in the form {{.NumSuccessfulJobs}}.
	if _, ok := state.(StateAggregationFunctions); !ok {
		var lowercase map[string]interface{}

		tmp, err := json.Marshal(state)
		if err != nil {
			return "", errors.Wrapf(err, "unable to create lowercase version (Marshal)")
		}

		if err := json.Unmarshal(tmp, &lowercase); err != nil {
			return "", errors.Wrapf(err, "unable to create lowercase version (Unmarshal)")
		}

		state = lowercase
	}

	if err := t.Execute(&out, state); err != nil {
		return "", errors.Wrapf(err, "malformed inputs. Available: %v", state)
	}

	return out.String(), nil
}

// GoValuate wraps the Evaluate function to the GoValuate expressions.
func (expr ExprState) GoValuate(state interface{}) (bool, error) {
	if expr == "" {
		return true, nil
	}

	out, err := expr.Evaluate(state)
	if err != nil {
		return false, errors.Wrapf(err, "dereference error")
	}

	expression, err := govaluate.NewEvaluableExpression(out)
	if err != nil {
		return false, errors.Wrapf(err, "invalid  expression '%s'", expr)
	}

	// The following loop converts govaluate variables (which we don't use), into strings. This
	// allows us to have expressions like: "foo != bar" without requiring foo and bar to be quoted.
	tokens := expression.Tokens()
	for i, tok := range tokens {
		switch tok.Kind {
		case govaluate.VARIABLE:
			tok.Kind = govaluate.STRING
		default:
			continue
		}

		tokens[i] = tok
	}

	expression, err = govaluate.NewEvaluableExpressionFromTokens(tokens)
	if err != nil {
		return false, errors.Wrapf(err, "failed to parse expression '%s'", expr)
	}

	result, err := expression.Evaluate(nil)
	if err != nil {
		return false, errors.Wrapf(err, "failed to evaluate expresion '%s'", expr)
	}

	boolRes, ok := result.(bool)
	if !ok {
		return false, errors.Errorf("expected boolean evaluation for '%s'. Got %v", expr, result)
	}

	return boolRes, nil
}

/*
	Validate Metrics Expressions
*/

// +kubebuilder:object:generate=false

// ExprMetricsValidator expressions evaluated with https://regex101.com/r/8JrgyI/1
var ExprMetricsValidator = regexp.MustCompile(`(?m)^(?P<reducer>\w+)\(\)\s+of\s+query\((?P<dashboardUID>\w+)\/(?P<panelID>\d+)\/(?P<metric>.+),\s+(?P<from>\w+),\s+(?P<to>\w+)\)\s+is\s+(?P<evaluator>\w+)\((?P<params>-*\d*[\.,\s]*\d*)\)\s*(for\s+\((?P<for>\w+)\))*\s*(every\((?P<every>\w+)\))*\s*$`)

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
			- 'avg() of query(summary/152/tx-avg, 1m, now) is below(5000)'
			- 'avg() of query(summary/152/tx-avg, 1m, now) is below(-5000)'

		Prepare your expressions at: https://regex101.com/r/8JrgyI/1`, query)
	}

	return matches, nil
}
