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

package utils

import (
	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ExpandedSpecBody string

func NewScheme(caller metav1.Object, inputs *v1alpha1.Inputs, body []byte) (*v1alpha1.Scheme, error) {
	var scheme v1alpha1.Scheme

	scheme.Scenario = v1alpha1.GetScenario(caller)
	scheme.Namespace = caller.GetNamespace()
	scheme.Inputs = inputs
	scheme.Spec = body

	return &scheme, nil
}

// Evaluate parses a given scheme and returns the respective ServiceSpec.
func Evaluate(scheme *v1alpha1.Scheme) (ExpandedSpecBody, error) {
	if scheme == nil {
		return "", errors.Errorf("empty scheme")
	}

	out, err := v1alpha1.ExprState(scheme.Spec).Evaluate(scheme)
	if err != nil {
		return "", errors.Wrapf(err, "template execution error")
	}

	return ExpandedSpecBody(out), nil
}
