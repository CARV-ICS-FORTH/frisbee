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
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

type Inputs struct {
	// Parameters are user-set values that are dynamically evaluated
	Parameters map[string]string `json:"parameters,omitempty"`
}

// TemplateSpec defines the desired state of Template
type TemplateSpec struct {
	// Inputs are dynamic fields that populate the spec.
	// +optional
	Inputs *Inputs `json:"inputs,omitempty"`

	// EmbedSpecs point to the Frisbee specs that can be templated.
	*EmbedSpecs `json:",inline"`
}

type EmbedSpecs struct {
	// +optional
	Service *ServiceSpec `json:"service,omitempty"`

	// +optional
	Chaos *ChaosSpec `json:"chaos,omitempty"`
}

// TemplateStatus defines the observed state of Template
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

// TemplateList contains a list of Template
type TemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Template `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Template{}, &TemplateList{})
}

// GenerateFromTemplate generates a spec by parameterizing the templateRef with the given inputs.
type GenerateFromTemplate struct {
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

	// Inputs are list of inputs passed to the objects.
	// Event used in conjunction with instances, if the number of instances is larger that the number of inputs,
	// then inputs are recursively iteration.
	// +optional
	Inputs []map[string]string `json:"inputs,omitempty"`
}

// Prepare automatically fills missing values from the template, according to the following rules:
// * Without inputs and without instances, there is 1 instance with default values.
// * Without instances, the number of instances is inferred by the number of inputs.
func (in *GenerateFromTemplate) Prepare(allowMultipleInputs bool) error {
	switch {
	case in.TemplateRef == "":
		return errors.New("empty templateRef")

	case len(in.Inputs) == 0: // use default parameters for all instances
		if in.MaxInstances == 0 {
			in.MaxInstances = 1
		}

		return nil

	case !allowMultipleInputs && len(in.Inputs) > 1: // object violation
		return errors.Errorf("Allowed inputs '%t' but got '%d'", allowMultipleInputs, len(in.Inputs))

	case len(in.Inputs) >= in.MaxInstances: // every instance has its own parameters.
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

func (in *GenerateFromTemplate) GetInput(i int) map[string]string {
	switch len(in.Inputs) {
	case 0:
		// no inputs
		return nil
	case 1:
		copied := make(map[string]string)

		for key, elem := range in.Inputs[0] {
			copied[key] = elem
		}

		return copied

	default:
		// safety is assumed by IterateInputs
		return in.Inputs[i]
	}
}

func (in *GenerateFromTemplate) IterateInputs(cb func(in map[string]string) error) error {
	if len(in.Inputs) == 0 {
		for i := 0; i < in.MaxInstances; i++ {
			if err := cb(nil); err != nil {
				return err
			}
		}
	} else {
		for i := 0; i < in.MaxInstances; i++ {
			// recursively iterate the input.
			if err := cb(in.GetInput(i % len(in.Inputs))); err != nil {
				return err
			}
		}
	}

	return nil
}

type Scheme struct {
	// Scenario returns the name of the scenario that invokes the template.
	Scenario string `json:"scenario,omitempty"`

	// Returns the namespace where the scenario is running
	Namespace string `json:"namespace,omitempty"`

	// Inputs are dynamic fields that populate the spec.
	// +optional
	Inputs *Inputs `json:"inputs,omitempty"`

	// Spec is the specification whose values will be replaced by parameters.
	Spec []byte `json:"spec"`
}
