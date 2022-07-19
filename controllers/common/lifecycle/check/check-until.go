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
	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common/expressions"
	"github.com/carv-ics-forth/frisbee/controllers/common/lifecycle"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func UntilConditionIsMet(spec *v1alpha1.ConditionalExpr, state lifecycle.ClassifierReader, job metav1.Object, lf *v1alpha1.Lifecycle) bool {
	if spec.HasMetricsExpr() {
		_, info, fired := expressions.AlertIsFired(job)
		if fired {
			*lf = v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseRunning,
				Reason:  "MetricsEventFired",
				Message: info,
			}

			meta.SetStatusCondition(&lf.Conditions, metav1.Condition{
				Type:    v1alpha1.ConditionAllJobsAreScheduled.String(),
				Status:  metav1.ConditionTrue,
				Reason:  "MetricsEventFired",
				Message: info,
			})

			return true
		}
	}

	if spec.HasStateExpr() {
		info, fired, err := expressions.FiredState(spec.State, state)
		if err != nil {
			*lf = v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseFailed,
				Reason:  "StateQueryError",
				Message: err.Error(),
			}

			meta.SetStatusCondition(&lf.Conditions, metav1.Condition{
				Type:    v1alpha1.ConditionJobUnexpectedTermination.String(),
				Status:  metav1.ConditionTrue,
				Reason:  "StateQueryError",
				Message: err.Error(),
			})

			return true
		}

		if fired {
			*lf = v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseRunning,
				Reason:  "StateEventFired",
				Message: info,
			}

			meta.SetStatusCondition(&lf.Conditions, metav1.Condition{
				Type:    v1alpha1.ConditionAllJobsAreScheduled.String(),
				Status:  metav1.ConditionTrue,
				Reason:  "StateEventFired",
				Message: info,
			})

			return true
		}
	}

	return false
}
