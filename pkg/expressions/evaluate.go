/*
Copyright 2022-2023 ICS-FORTH.

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
	"fmt"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/pkg/lifecycle"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Condition struct {
	Expr *v1alpha1.ConditionalExpr
	Info string
}

func (c Condition) IsTrue(state lifecycle.ClassifierReader, job metav1.Object) bool {
	// Check for state expressions
	if c.Expr.HasStateExpr() {
		pass, err := c.Expr.State.GoValuate(state)
		if err != nil {
			c.Info = fmt.Sprintf("Err: '%s'. DebugInfo: '%s'", err, state.ListAll())

			return false
		}

		c.Info = fmt.Sprintf("State '%s' is %t", c.Expr.State, pass)

		return pass
	}

	if c.Expr.HasMetricsExpr() {
		_, info, fired := AlertIsFired(job)

		c.Info = fmt.Sprintf("Alert '%s' is %s", c.Expr.Metrics, info)

		// non-fired mean that the condition is still true.
		// fired means that the condition is violated, and should return false
		return !fired
	}

	return false
}

func (c Condition) GetInfo() string {
	return c.Info
}
