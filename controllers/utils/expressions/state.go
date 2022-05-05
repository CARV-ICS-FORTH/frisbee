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

package expressions

import (
	"strings"
	"text/template"

	"github.com/Knetic/govaluate"
	"github.com/Masterminds/sprig/v3"
	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/utils/lifecycle"
	"github.com/pkg/errors"
)

var sprigFuncMap = sprig.TxtFuncMap() // a singleton for better performance

// FiredState enforces user-driven decisions as to when the test has passed or has fail.
// if it has failed, it updates the workflow status and returns true to indicate that the status has been modified.
func FiredState(expr v1alpha1.ExprState, state lifecycle.ClassifierReader) (string, bool, error) {
	if expr == "" || state.IsZero() {
		return "", false, nil
	}

	t, err := template.New("").Funcs(sprigFuncMap).Option("missingkey=error").Parse(string(expr))
	if err != nil {
		return "ParsingError", false, errors.Wrapf(err, "dereference error")
	}

	var out strings.Builder

	if err := t.Execute(&out, state); err != nil {
		return "AssertionExecutionError", false, errors.Wrapf(err, "execution error")
	}

	pass, err := shouldExecute(out.String())
	if err != nil {
		return "AssertionExecutionError", false, errors.Wrapf(err, "assertion dereference error")
	}

	if pass {
		return "AssertionValidationOK", true, nil
	}

	return "AssertionValidationError", false, nil
}

// Taken from Argo-TestPlan.
// shouldExecute evaluates an already substituted expression to decide whether a step should execute.
func shouldExecute(expr string) (bool, error) {
	if expr == "" {
		return true, nil
	}

	expression, err := govaluate.NewEvaluableExpression(expr)
	if err != nil {
		if strings.Contains(err.Error(), "Invalid token") {
			return false, errors.Wrapf(err, "Invalid 'expr' expression '%s': %v "+
				"(hint: try wrapping the affected expression in quotes (\"))", expr, err)
		}

		return false, errors.Wrapf(err, "Invalid 'expr' expression '%s': %v", expr, err)
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
		return false, errors.Wrapf(err, "Failed to parse 'expr' expression '%s': %v", expr, err)
	}

	result, err := expression.Evaluate(nil)
	if err != nil {
		return false, errors.Wrapf(err, "Failed to FiredState 'expr' expresion '%s': %v", expr, err)
	}

	boolRes, ok := result.(bool)
	if !ok {
		return false, errors.Errorf("QueuedJobs boolean evaluation for '%s'. Got %v", expr, result)
	}

	return boolRes, nil
}
