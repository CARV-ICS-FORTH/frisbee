/*
Copyright 2022 ICS-FORTH.

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

package check

import (
	"fmt"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common/lifecycle"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TolerationExceeded compares the TolerateSpec against the state, and if the toleration are exceeded,
// it modifies the given lifecycle and returns true. Otherwise, the lifecycle remains the same and returns false.
func TolerationExceeded(spec *v1alpha1.TolerateSpec, state lifecycle.ClassifierReader, lf *v1alpha1.Lifecycle) bool {
	if spec == nil {
		panic(errors.Errorf("empty spec"))
	}

	if state.FailedJobsNum() > spec.FailedJobs {
		*lf = v1alpha1.Lifecycle{
			Phase:  v1alpha1.PhaseFailed,
			Reason: "TolerateFailuresExceeded",
			Message: fmt.Sprintf("tolerate: %s. failed jobs: %s",
				spec.String(), state.FailedJobsList()),
		}

		meta.SetStatusCondition(&lf.Conditions, metav1.Condition{
			Type:    v1alpha1.ConditionJobUnexpectedTermination.String(),
			Status:  metav1.ConditionTrue,
			Reason:  "JobHasFailed",
			Message: fmt.Sprintf("failed jobs: %s", state.FailedJobsList()),
		})

		return true
	}

	return false
}
