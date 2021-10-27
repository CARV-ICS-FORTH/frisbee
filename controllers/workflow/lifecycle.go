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
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type test struct {
	expression bool
	lifecycle  v1alpha1.Lifecycle
	condition  metav1.Condition
}

func (r *Controller) calculateLifecycle(w *v1alpha1.Workflow) v1alpha1.WorkflowStatus {
	status := w.Status
	gs := r.state // global state

	// Skip any CR which are already completed, or uninitialized.
	if status.Phase == v1alpha1.PhaseUninitialized ||
		status.Phase == v1alpha1.PhaseSuccess ||
		status.Phase == v1alpha1.PhaseFailed {
		return status
	}

	// we are only interested in the number of jobs in each category.
	expectedJobs := len(w.Spec.Actions)

	userTests := r.useOracle(w, gs)

	autotests := []test{
		{ // A job has failed during execution.
			expression: gs.NumFailedJobs() > 0,
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseFailed,
				Reason:  "JobHasFailed",
				Message: fmt.Sprintf("failed jobs: %s", gs.FailedList()),
			},
			condition: metav1.Condition{
				Type:    v1alpha1.ConditionJobFailed.String(),
				Status:  metav1.ConditionTrue,
				Reason:  "JobHasFailed",
				Message: fmt.Sprintf("failed jobs: %s", gs.FailedList()),
			},
		},
		{ // All jobs are created, and completed successfully
			expression: gs.NumSuccessfulJobs() == expectedJobs,
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseSuccess,
				Reason:  "AllJobsCompleted",
				Message: fmt.Sprintf("successful jobs: %s", gs.SuccessfulList()),
			},
			condition: metav1.Condition{
				Type:    v1alpha1.ConditionAllJobsDone.String(),
				Status:  metav1.ConditionTrue,
				Reason:  "AllJobsCompleted",
				Message: fmt.Sprintf("successful jobs: %s", gs.SuccessfulList()),
			},
		},
		{ // All jobs are created, and at least one is still running
			expression: gs.NumRunningJobs()+gs.NumSuccessfulJobs() == expectedJobs,
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseRunning,
				Reason:  "JobIsRunning",
				Message: fmt.Sprintf("running jobs: %s", gs.RunningList()),
			},
			condition: metav1.Condition{
				Type:    v1alpha1.ConditionAllJobs.String(),
				Status:  metav1.ConditionTrue,
				Reason:  "AllJobsRunning",
				Message: fmt.Sprintf("running jobs: %s", gs.RunningList()),
			},
		},
		{ // Not all Jobs are yet created
			expression: status.Phase == v1alpha1.PhasePending,
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhasePending,
				Reason:  "JobIsPending",
				Message: "at least one jobs has not yet created",
			},
		},
	}

	allTests := append(userTests, autotests...)

	for _, testcase := range allTests {
		if testcase.expression {
			status.Lifecycle = testcase.lifecycle

			if testcase.condition != (metav1.Condition{}) {
				meta.SetStatusCondition(&status.Conditions, testcase.condition)
			}

			return status
		}
	}

	logrus.Warn("Workflow Debug info \n",
		" current ", status.Lifecycle.Phase,
		" total actions: ", len(w.Spec.Actions),
		" activeJobs: ", gs.ActiveList(),
		" successfulJobs: ", gs.SuccessfulList(),
		" failedJobs: ", gs.FailedList(),
		" cur status: ", status,
	)

	panic("unhandled lifecycle conditions")
}

// useOracle enforces user-driven decisions as to when the test has passed or has fail.
// the return arguments are: lifecycle, apply, error.
func (r *Controller) useOracle(w *v1alpha1.Workflow, gs lifecycle.Classifier) []test {
	oracle := w.Spec.WithTestOracle

	if oracle == nil {
		return nil
	}

	var testlist []test

	if exp := oracle.Pass; exp != nil {
		expanded, err := dereference(*exp, gs)
		if err != nil {
			panic(errors.Wrapf(err, "dereference error"))
		}

		ok, _ := shouldExecute(expanded)
		if ok {
			testlist = append(testlist, test{
				expression: true,
				lifecycle: v1alpha1.Lifecycle{
					Phase:   v1alpha1.PhaseSuccess,
					Reason:  "OraclePass",
					Message: "The oracle decided that the test has passed",
				},
				condition: metav1.Condition{
					Type:    v1alpha1.WorkflowOracle.String(),
					Status:  metav1.ConditionTrue,
					Reason:  "TestPass",
					Message: fmt.Sprintf("expr:%s", *exp),
				},
			})
		}
	}

	if exp := oracle.Fail; exp != nil {
		expanded, err := dereference(*exp, gs)
		if err != nil {
			panic(errors.Wrapf(err, "dereference error"))
		}

		ok, _ := shouldExecute(expanded)
		if ok {
			testlist = append(testlist, test{
				expression: true,
				lifecycle: v1alpha1.Lifecycle{
					Phase:   v1alpha1.PhaseSuccess,
					Reason:  "TestFail",
					Message: "The oracle decided that the test has failed",
				},
				condition: metav1.Condition{
					Type:    v1alpha1.WorkflowOracle.String(),
					Status:  metav1.ConditionFalse,
					Reason:  "OracleFailed",
					Message: fmt.Sprintf("expr:%s", *exp),
				},
			})
		}
	}

	return testlist
}

func ValidateOracle(w *v1alpha1.Workflow, gs lifecycle.Classifier) error {
	oracle := w.Spec.WithTestOracle

	if oracle == nil {
		return nil
	}

	if exp := oracle.Pass; exp != nil {
		expanded, err := dereference(*exp, gs)
		if err != nil {
			return errors.Wrapf(err, "dereference error")
		}

		_, err = shouldExecute(expanded)
		if err != nil {
			return err
		}
	}

	if exp := oracle.Fail; exp != nil {
		expanded, err := dereference(*exp, gs)
		if err != nil {
			return errors.Wrapf(err, "dereference error")
		}

		_, err = shouldExecute(expanded)
		if err != nil {
			return err
		}
	}

	return nil
}

var sprigFuncMap = sprig.TxtFuncMap() // a singleton for better performance

// deference gives access to the gs from the template.
func dereference(oracle string, gs lifecycle.Classifier) (string, error) {
	t := template.Must(
		template.New("").
			Funcs(sprigFuncMap).
			Option("missingkey=error").
			Parse(oracle),
	)

	var out strings.Builder

	if err := t.Execute(&out, &gs); err != nil {
		return "", errors.Wrapf(err, "execution error")
	}

	return out.String(), nil
}

// Taken from Argo-Workflow.
// shouldExecute evaluates a already substituted when expression to decide whether or not a step should execute
func shouldExecute(when string) (bool, error) {
	if when == "" {
		return true, nil
	}

	expression, err := govaluate.NewEvaluableExpression(when)
	if err != nil {
		if strings.Contains(err.Error(), "Invalid token") {
			return false, errors.Wrapf(err, "Invalid 'when' expression '%s': %v "+
				"(hint: try wrapping the affected expression in quotes (\"))", when, err)
		}

		return false, errors.Wrapf(err, "Invalid 'when' expression '%s': %v", when, err)
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
		return false, errors.Wrapf(err, "Failed to parse 'when' expression '%s': %v", when, err)
	}

	result, err := expression.Evaluate(nil)
	if err != nil {
		return false, errors.Wrapf(err, "Failed to evaluate 'when' expresion '%s': %v", when, err)
	}

	boolRes, ok := result.(bool)
	if !ok {
		return false, errors.Errorf("Expected boolean evaluation for '%s'. Got %v", when, result)
	}

	return boolRes, nil
}
