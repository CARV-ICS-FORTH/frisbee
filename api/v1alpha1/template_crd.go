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

type Inputs struct {
	// Parameters define dynamically valued fields. The values are given by higher level entities, such as the workflow.
	Parameters map[string]string `json:"parameters,omitempty"`
}

// TemplateSpec defines the desired state of Template
type TemplateSpec struct {
	// Inputs are dynamic fields that populate the spec.
	// +optional
	Inputs *Inputs `json:"inputs,omitempty"`

	*EmbedSpecs `json:",inline"`
}

type EmbedSpecs struct {
	// +optional
	Service *ServiceSpec `json:"service,omitempty"`
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
// +kubebuilder:subresource:status

// Template is the Schema for the templates API
type Template struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TemplateSpec   `json:"spec,omitempty"`
	Status TemplateStatus `json:"status,omitempty"`
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

	// Instances dictate the number of objects to be created for the service. If Env is specified, the values
	// with be identical across the spawned instances. For instances with different parameters, use Inputs.
	// +optional
	Instances int `json:"instances"`

	// Inputs are list of inputs passed to the objects. When used in conjunction with Instances, there can be
	// only one input and all the instances will run with identical parameters. If Instances is defined and there are
	// more than one inputs, the request will be rejected.
	// +optional
	Inputs []map[string]string `json:"inputs,omitempty"`
}

func (t *GenerateFromTemplate) Validate(allowMultipleInputs bool) error {
	switch {
	case t.TemplateRef == "":
		return errors.New("empty templateRef")

	case len(t.Inputs) == 0 && t.Instances == 0: // use default parameters for all instances
		t.Instances = 1

		return nil

	case len(t.Inputs) == 0 && t.Instances > 0: // use default parameters for all instances
		return nil

	case !allowMultipleInputs && len(t.Inputs) > 1: // object violation
		return errors.Errorf("Allowed inputs [%t] but got [%d]", allowMultipleInputs, len(t.Inputs))

	case len(t.Inputs) >= t.Instances: // every instance has its own parameters.
		t.Instances = len(t.Inputs)

		return nil

	case t.Instances > len(t.Inputs) && len(t.Inputs) > 1:
		return errors.New("Max one input when multiple instances are defined")

	case len(t.Inputs) == 1 && t.Instances > 0: // all instances have the same parameters.
		return nil

	default:
		logrus.Warn(
			"TemplateRef:", t.TemplateRef,
			" Instances:", t.Instances,
			" AllowMultipleInputs:", allowMultipleInputs,
			" Inputs:", t.Inputs,
		)

		panic("unhandled case")
	}
}

func (t *GenerateFromTemplate) GetInput(i int) map[string]string {
	switch len(t.Inputs) {
	case 0:
		// no inputs
		return nil
	case 1:
		copied := make(map[string]string)

		for key, elem := range t.Inputs[0] {
			copied[key] = elem
		}

		return copied

	default:
		return t.Inputs[i]
	}
}

func (t *GenerateFromTemplate) Iterate(cb func(in map[string]string) error) error {
	for i := 0; i < t.Instances; i++ {
		if err := cb(t.GetInput(i)); err != nil {
			return err
		}
	}

	return nil
}
