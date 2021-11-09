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

package workflow

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/Knetic/govaluate"
	"github.com/Masterminds/sprig/v3"
	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/utils/lifecycle"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// assert enforces user-driven decisions as to when the test has passed or has fail.
// if it has failed, it updates the workflow status and returns true to indicate that the status has been modified.
func assert(exp string, w *v1alpha1.Workflow, gs *lifecycle.Classifier) bool {
	accessor := BuiltinObjects{
		Workflow: w,
		Runtime:  gs,
	}

	// when condition is fulfilled, and we must Evaluate the assertion
	pass, err := Evaluate(exp, accessor)

	if err != nil {
		w.Status.Lifecycle = v1alpha1.Lifecycle{
			Phase:   v1alpha1.PhaseFailed,
			Reason:  "AssertionError",
			Message: errors.Wrapf(err, "assertion dereference error").Error(),
		}

		meta.SetStatusCondition(&w.Status.Conditions, metav1.Condition{
			Type:    v1alpha1.WorkflowAssertion.String(),
			Status:  metav1.ConditionTrue,
			Reason:  "AssertionError",
			Message: errors.Wrapf(err, "assertion dereference error").Error(),
		})

		return false
	}

	if !pass {
		w.Status.Lifecycle = v1alpha1.Lifecycle{
			Phase:   v1alpha1.PhaseFailed,
			Reason:  "AssertionFailed",
			Message: fmt.Sprintf("Assertion has failed. [%s]", exp),
		}

		meta.SetStatusCondition(&w.Status.Conditions, metav1.Condition{
			Type:    v1alpha1.WorkflowAssertion.String(),
			Status:  metav1.ConditionTrue,
			Reason:  "AssertionFailed",
			Message: fmt.Sprintf("Assertion has failed. [%s]", exp),
		})

		return false
	}

	return true
}

var sprigFuncMap = sprig.TxtFuncMap() // a singleton for better performance

// BuiltinObjects are passed into a template from the template engine.
type BuiltinObjects struct {
	Workflow *v1alpha1.Workflow
	Runtime  *lifecycle.Classifier
}

func Evaluate(expr string, objects BuiltinObjects) (bool, error) {
	if expr == "" {
		return true, nil
	}

	if objects == (BuiltinObjects{}) {
		return false, errors.New("invalid state objects")
	}

	// dereference expands the templated expression
	t := template.Must(
		template.New("").
			Funcs(sprigFuncMap).
			Option("missingkey=error").
			Parse(expr),
	)

	var out strings.Builder

	if err := t.Execute(&out, objects); err != nil {
		return false, errors.Wrapf(err, "execution error")
	}

	return shouldExecute(out.String())
}

// Taken from Argo-Workflow.
// shouldExecute evaluates an already substituted expression to decide whether a step should execute
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
		return false, errors.Wrapf(err, "Failed to Evaluate 'expr' expresion '%s': %v", expr, err)
	}

	boolRes, ok := result.(bool)
	if !ok {
		return false, errors.Errorf("Expected boolean evaluation for '%s'. Got %v", expr, result)
	}

	return boolRes, nil
}
