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

package v1alpha1

import (
	"reflect"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Template is the Schema for the templates API
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Template struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TemplateSpec   `json:"spec,omitempty"`
	Status TemplateStatus `json:"status,omitempty"`
}

func ParameterValue(v interface{}) *apiextensionsv1.JSON {
	raw, _ := json.Marshal(v)

	return &apiextensionsv1.JSON{Raw: raw}
}

type Parameters map[string]*apiextensionsv1.JSON

func (in Parameters) Unmarshal() (map[string]interface{}, error) {
	decoded := map[string]interface{}{}

	for key, value := range in {
		var eValue interface{}

		if err := json.Unmarshal(value.Raw, &eValue); err != nil {
			return nil, errors.Wrapf(err, "cannot unmarshal parameters")
		}

		decoded[key] = eValue
	}

	return decoded, nil
}

type TemplateInputs struct {
	// Parameters are user-set values that are dynamically evaluated
	// +optional
	Parameters Parameters `json:"parameters,omitempty"`

	// Namespace returns the namespace from which the template is called from.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Scenario returns the scenario from which the template is called from.
	// +optional
	Scenario string `json:"scenario,omitempty"`
}

// TemplateSpec defines the desired state of Template.
type TemplateSpec struct {
	// Inputs are dynamic fields that populate the spec.
	// +optional
	Inputs *TemplateInputs `json:"inputs,omitempty"`

	// EmbedSpecs point to the Frisbee specs that can be templated.
	*EmbedSpecs `json:",inline"`
}

type EmbedSpecs struct {
	// +optional
	Service *ServiceSpec `json:"service,omitempty"`

	// +optional
	Chaos *ChaosSpec `json:"chaos,omitempty"`
}

// TemplateStatus defines the observed state of Template.
type TemplateStatus struct {
	Lifecycle `json:",inline"`
}

func (in *Template) GetReconcileStatus() Lifecycle {
	return in.Status.Lifecycle
}

func (in *Template) SetReconcileStatus(lifecycle Lifecycle) {
	in.Status.Lifecycle = lifecycle
}

// +kubebuilder:object:root=true

// TemplateList contains a list of Template.
type TemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Template `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Template{}, &TemplateList{})
}

/*

	Smart Guy for building objects from a template.

*/

// +kubebuilder:object:generate=false

type UserInputs map[string]*apiextensionsv1.JSON

func (u UserInputs) Unmarshal() (map[string]interface{}, error) {
	decoded := map[string]interface{}{}

	for key, value := range u {
		var eValue interface{}

		if err := json.Unmarshal(value.Raw, &eValue); err != nil {
			return nil, errors.Wrapf(err, "cannot unmarshal parameters")
		}

		decoded[key] = eValue
	}

	return decoded, nil
}

// GenerateObjectFromTemplate generates a spec by parameterizing the templateRef with the given inputs.
type GenerateObjectFromTemplate struct {
	// TemplateRef refers to a  template (e.g, iperf-server).
	TemplateRef string `json:"templateRef"`

	// Until defines the conditions under which the CR will stop spawning new jobs.
	// If used in conjunction with inputs, it will loop over inputs until the conditions are met.
	// +optional
	Until *ConditionalExpr `json:"until,omitempty"`

	// MaxInstances dictate the number of objects to be created for the CR.
	// If no inputs are defined, then all instances will be initiated using the default parameters of the template.
	// Event used in conjunction with Until, MaxInstances as a max bound.
	// +optional
	MaxInstances int `json:"instances"`

	// UserParameters is a map of parameters passed to the objects.
	// Event used in conjunction with instances, if the number of instances is larger that the number of inputs,
	// then inputs are recursively iteration.
	// +optional
	Inputs []UserInputs `json:"inputs,omitempty"`
}

// Prepare automatically fills missing values from the template, according to the following rules:
// * Without inputs and without instances, there is 1 instance with default values.
// * Without instances, the number of instances is inferred by the number of inputs.
func (in *GenerateObjectFromTemplate) Prepare(allowMultipleInputs bool) error {
	switch {
	case in.TemplateRef == "":
		return errors.New("empty templateRef")

	case len(in.Inputs) > 1 && !allowMultipleInputs: // object violation
		return errors.Errorf("Allowed inputs '%t' but got '%d'", allowMultipleInputs, len(in.Inputs))

	case len(in.Inputs) == 0 && in.MaxInstances == 0: // use default parameters for all instances
		in.MaxInstances = 1

		return nil

	case len(in.Inputs) > in.MaxInstances: // every instance has its own parameters.
		in.MaxInstances = len(in.Inputs)

		return nil

	case in.MaxInstances > 0: // all instances have the same parameters.
		return nil

	default:
		logrus.Warn(
			"TemplateRef:", in.TemplateRef,
			" MaxInstances:", in.MaxInstances,
			" AllowMultipleInputs:", allowMultipleInputs,
			" Inputs:", in.Inputs,
		)

		panic("unhandled case")
	}
}

func (in *GenerateObjectFromTemplate) GetInputs(inputIndex uint) UserInputs {
	switch len(in.Inputs) {
	case 0:
		// no inputs
		return nil
	case 1:
		copied := UserInputs{}

		for key, value := range in.Inputs[0] {
			copied[key] = value
		}

		return copied

	default:
		// safety is assumed by IterateInputs
		return in.Inputs[inputIndex]
	}
}

func (in *GenerateObjectFromTemplate) IterateInputs(callBack func(nextInputSet uint) error) error {
	if len(in.Inputs) == 0 {
		for i := 0; i < in.MaxInstances; i++ {
			if err := callBack(0); err != nil {
				return err
			}
		}
	} else {
		for i := 0; i < in.MaxInstances; i++ {
			// recursively iterate the input.
			if err := callBack(uint(i % len(in.Inputs))); err != nil {
				return err
			}
		}
	}

	return nil
}

func (in *GenerateObjectFromTemplate) Generate(spec interface{}, userInputsSet uint, tSpec TemplateSpec, templateBody []byte) error {
	evaluationParams := struct {
		Inputs struct {
			Parameters map[string]interface{} `json:"parameters"`
			Namespace  string                 `json:"namespace"`
			Scenario   string                 `json:"scenario"`
		} `json:"inputs"`
	}{}

	// Step 1. Expose Scope Information
	evaluationParams.Inputs.Namespace = tSpec.Inputs.Namespace
	evaluationParams.Inputs.Scenario = tSpec.Inputs.Scenario

	// Step 2: Initialize using the default templat evalues
	templateParams, err := tSpec.Inputs.Parameters.Unmarshal()
	if err != nil {
		return errors.Wrapf(err, "cannot unmarshal template parameters")
	}

	evaluationParams.Inputs.Parameters = templateParams

	// Step 3: Overwrite default parameters with user arguments
	if in.Inputs != nil {
		if tSpec.Inputs == nil || tSpec.Inputs.Parameters == nil {
			return errors.New("template is not parameterizable")
		}

		userParams, err := in.Inputs[userInputsSet].Unmarshal()
		if err != nil {
			return errors.Wrapf(err, "cannot unmarshal user parameters")
		}

		for key, value := range userParams {
			expected, exists := templateParams[key]
			if !exists {
				return errors.Errorf("parameter '%s' does not exist", key)
			}

			if reflect.TypeOf(expected) != reflect.TypeOf(value) {
				return errors.Errorf("mismatched types. expected '%s' but got '%s'",
					reflect.TypeOf(expected), reflect.TypeOf(value))
			}

			evaluationParams.Inputs.Parameters[key] = value
		}
	}

	// Step 4: Evaluate the template and decode it to the caller's type.
	expandedTemplateBody, err := ExprState(templateBody).Evaluate(evaluationParams)
	if err != nil {
		return errors.Wrapf(err, "template execution error")
	}

	return json.Unmarshal([]byte(expandedTemplateBody), spec)
}
