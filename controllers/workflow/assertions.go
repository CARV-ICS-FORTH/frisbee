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
	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/telemetry/grafana"
	"github.com/carv-ics-forth/frisbee/controllers/utils"
	"github.com/carv-ics-forth/frisbee/controllers/utils/lifecycle"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// JobHasSLA indicate that a job has SLA assertion. Used to deregister the alert once the job has finished.
	// Used as [JobHasSLA]: [alertID]
	JobHasSLA = "frisbee.io/sla"

	// SLAViolationFired indicate that a Grafana alert has been fired.
	// Used as [SLAViolationFired]: [alertID]
	SLAViolationFired = "sla.frisbee.io/fire"

	// SLAViolationInfo include information about the fired Grafana Alert.
	// Used as [SlaViolationINfo]: [string]
	SLAViolationInfo = "sla.frisbee.io/info"
)

func InsertSLAAlert(action v1alpha1.Action, w *v1alpha1.Workflow, job metav1.Object) error {
	if action.Assert == nil || action.Assert.SLA == "" {
		return nil
	}

	// create an alert
	alert, err := grafana.NewAlert(action.Assert.SLA)
	if err != nil {
		return errors.Wrapf(err, "invalid SLA")
	}

	alert.Name = action.Name
	alert.Message = fmt.Sprintf("The SLA of action [%s] has failed", action.Name)

	// push the alert to grafana
	alertID, err := grafana.DefaultClient.SetAlert(alert)
	if err != nil {
		return errors.Wrapf(err, "SLA injection error")
	}

	// use annotations to know which jobs have alert in Grafana.
	// we use this information to remove alerts when the jobs are complete.
	utils.MergeAnnotation(job, map[string]string{JobHasSLA: fmt.Sprint(alertID)})

	if w.Status.ExpectedAlerts == nil {
		w.Status.ExpectedAlerts = make(map[string]bool)
	}

	w.Status.ExpectedAlerts[fmt.Sprint(alertID)] = true

	return nil
}

func AssertSLA(w *v1alpha1.Workflow) (string, bool) {
	annotations := w.GetAnnotations()

	alertID, exists := annotations[SLAViolationFired]
	if !exists {
		return "", false
	}

	info := annotations[SLAViolationInfo]

	enabled, ok := w.Status.ExpectedAlerts[alertID]
	if !ok {
		// logrus.Warn("Unhandled alert has been fired", info)

		return "", false
	}

	if !enabled {
		logrus.Warn("A disabled alert has been fired", info)

		return "", false
	}

	return info, true
}

var sprigFuncMap = sprig.TxtFuncMap() // a singleton for better performance

type BuiltinState struct {
	Runtime *lifecycle.Classifier
}

// AssertState enforces user-driven decisions as to when the test has passed or has fail.
// if it has failed, it updates the workflow status and returns true to indicate that the status has been modified.
func AssertState(expr string, state *BuiltinState) error {
	if expr == "" {
		return nil
	}

	if state == nil || state.Runtime == nil {
		return errors.New("invalid state state")
	}

	// dereference expands the templated expression
	t := template.Must(
		template.New("").
			Funcs(sprigFuncMap).
			Option("missingkey=error").
			Parse(expr),
	)

	var out strings.Builder

	if err := t.Execute(&out, state); err != nil {
		return errors.Wrapf(err, "execution error")
	}

	pass, err := shouldExecute(out.String())
	if err != nil {
		return errors.Wrapf(err, "assertion dereference error")
	}

	if !pass {
		return errors.Errorf("Assert has failed. [%s]", expr)
	}

	return nil
}

// Taken from Argo-Workflow.
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
		return false, errors.Wrapf(err, "Failed to AssertState 'expr' expresion '%s': %v", expr, err)
	}

	boolRes, ok := result.(bool)
	if !ok {
		return false, errors.Errorf("Expected boolean evaluation for '%s'. Got %v", expr, result)
	}

	return boolRes, nil
}
