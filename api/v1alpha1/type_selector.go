// Licensed to FORTH/ICS under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. FORTH/ICS licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package v1alpha1

// Mode represents the filter for selecting on of many.
type Mode string

const (
	// AnyMode represents that the system will select one object randomly.
	AnyMode Mode = "any"
	// AllMode represents that the system will select all objects  regardless of status
	// (not ready or not running pods includes).
	// Use this label carefully.
	AllMode Mode = "all"
	// FixedMode represents that the system will select a specific number of running objects.
	FixedMode Mode = "fixed"
	// FixedPercentMode to specify a fixed % that can be injected chaos action.
	FixedPercentMode Mode = "fixed-percent"
	// RandomMaxPercentMode to specify a maximum % that can be injected chaos action.
	RandomMaxPercentMode Mode = "random-max-percent"
)

func Convert(mode string) Mode {
	switch mode {
	case "any":
		return AnyMode
	case "all":
		return AllMode
	case "fixed":
		return FixedMode
	case "fixed-percent":
		return FixedPercentMode
	case "random-max-percent":
		return RandomMaxPercentMode
	default:
		panic("invalid mode")
	}
}

// +kubebuilder:validation:Enum=one;all;fixed;fixed-percent;random-max-percent

// TemplateSelectorSpec defines some selectors for chosing a template
type TemplateSelectorSpec struct {
	// Reference selects the service template with the specified value.
	// +optional
	Reference string `json:"entry,omitempty"`
}

type TemplateSelector struct {
	Namespace string `json:"namespace"`

	// Family defines the target family of templates
	// +optional
	Family string `json:"family,omitempty"`

	// Selector contains the rules to select templates (services, failures) within the target family
	Selector TemplateSelectorSpec `json:"selector"`
}

// MatchServiceSpec defines the selectors for services.
// If the all selectors are empty, all services will be selected.
type MatchServiceSpec struct {
	// ServiceNames is a map of string keys and a set values that used to select services.
	// The key defines the namespace which services belong, and the values is a set of service names.
	// +optional
	// ServiceNames map[string][]string `json:"names,omitempty"`

	// Map of string keys and values that can be used to select objects.
	// A selector based on labels.
	// +optional
	// Labels map[string]string `json:"labels,omitempty"`

	// ServiceGroup defines the service group where services belong. A ServiceGroup may refer either to
	// a DistributedGroup or a CollocatedGroup. Therefore, we need to search both of them.
	// +optional
	ServiceGroup map[string]string `json:"servicegroup,omitempty"`

	// Namespaces is a set of namespace to which objects belong.
	// +optional
	// Namespaces []string `json:"namespaces,omitempty"`
}

type ServiceSelector struct {
	// Match contains the rules to select target
	// +optional
	Match MatchServiceSpec `json:"match,omitempty"`

	// Mode defines which of the selected services to use. If undefined, all() is used
	// Supported mode: one / all / fixed / fixed-percent / random-max-percent
	// +optional
	Mode Mode `json:"mode"`

	// Value is required when the mode is set to `FixedPodMode` / `FixedPercentPodMod` / `RandomMaxPercentPodMod`.
	// If `FixedPodMode`, provide an integer of pods to do chaos action.
	// If `FixedPercentPodMod`, provide a number from 0-100 to specify the percent of pods the server can do chaos action.
	// IF `RandomMaxPercentPodMod`,  provide a number from 0-100 to specify the max percent of pods to do chaos action
	// +optional
	Value string `json:"value,omitempty"`

	// Macro abstract selector parameters into a structured string (e.g, .groupservice.master.all). Every parsed field is
	// represents an inner structure of the selector.
	// In case of invalid macro, the selector will return empty results.
	// Macro conflicts with any other parameter.
	// +optional
	Macro *string `json:"macro,omitempty"`
}
