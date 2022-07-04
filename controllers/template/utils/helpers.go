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
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common/labelling"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var sprigFuncMap = sprig.TxtFuncMap() // a singleton for better performance

type ExpandedSpecBody string

type Scheme struct {
	// Scenario returns the name of the scenario that invokes the template.
	Scenario string `json:"scenario,omitempty"`

	// Returns the namespace where the scenario is running
	Namespace string `json:"namespace,omitempty"`

	// Inputs are dynamic fields that populate the spec.
	// +optional
	Inputs *v1alpha1.Inputs `json:"inputs,omitempty"`

	// Spec is the specification whose values will be replaced by parameters.
	Spec []byte `json:"spec"`
}

func NewScheme(caller metav1.Object, inputs *v1alpha1.Inputs, body []byte) (*Scheme, error) {
	var scheme Scheme

	scheme.Scenario = labelling.GetScenario(caller)
	scheme.Namespace = caller.GetNamespace()
	scheme.Inputs = inputs
	scheme.Spec = body

	return &scheme, nil
}

// Evaluate parses a given scheme and returns the respective ServiceSpec.
func Evaluate(scheme *Scheme) (ExpandedSpecBody, error) {
	if scheme == nil {
		return "", errors.Errorf("empty scheme")
	}

	t, err := template.New("").
		Funcs(sprigFuncMap).
		Option("missingkey=error").
		Parse(string(scheme.Spec))

	if err != nil {
		return "", errors.Wrapf(err, "parsing error")
	}

	var out strings.Builder

	// replace templated expression with actual values.
	if err := t.Execute(&out, scheme); err != nil {
		return "", errors.Wrapf(err, "template execution error")
	}

	return ExpandedSpecBody(out.String()), nil
}
