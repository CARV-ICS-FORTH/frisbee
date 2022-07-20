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
	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common/lifecycle"
	"github.com/pkg/errors"
)

// FiredState enforces user-driven decisions as to when the test has passed or has fail.
// if it has failed, it updates the workflow status and returns true to indicate that the status has been modified.
func FiredState(expr v1alpha1.ExprState, state lifecycle.ClassifierReader) (string, bool, error) {

	pass, err := expr.GoValuate(state)
	if err != nil {
		return "ExecutionError", false, errors.Wrapf(err, "dereference error")
	}

	if pass {
		return "StateOK", true, nil
	}

	return "InvalidTransition", false, nil
}
