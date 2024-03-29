/*
Copyright 2021-2023 ICS-FORTH.

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

// Mode represents the filter for selecting on of many.
type Mode string

const (
	// OneMode represents that the system will select one object randomly.
	OneMode Mode = "one"
	// AllMode represents that the system will select all objects  regardless of status
	// (not ready or not running pods includes).
	// Use this label carefully.
	AllMode Mode = "all"
	// FixedMode represents that the system will select a specific number of running objects.
	FixedMode Mode = "fixed"
	// FixedPercentMode to specify a fixed % of a cluster.
	FixedPercentMode Mode = "fixed-percent"
	// RandomMaxPercentMode to specify a maximum % of a cluster.
	RandomMaxPercentMode Mode = "random-max-percent"
)

func Convert(mode string) Mode {
	switch mode {
	case "one":
		return OneMode
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

// MatchBy defines the selectors for services.
// If the all selectors are empty, all services will be selected.
type MatchBy struct {
	// ByName is a map of string keys and a set values that used to select services.
	// The key defines the namespace which services belong, and the values is a set of service names.
	// +optional
	ByName map[string][]string `json:"byName,omitempty"`

	// Map of string keys and values that can be used to select objects.
	// A selector based on labels.
	// +optional
	// Labels map[string]string `json:"labels,omitempty"`

	// ByCluster defines the service group where services belong.
	// +optional
	ByCluster map[string]string `json:"byCluster,omitempty"`

	// Namespaces is a set of namespace to which objects belong.
	// +optional
	// Namespaces []string `json:"namespaces,omitempty"`
}

type ServiceSelector struct {
	// Match contains the rules to select target
	// +optional
	Match MatchBy `json:"match,omitempty"`

	// Mode defines which of the selected services to use. If undefined, all() is used
	// Supported mode: one / all / fixed / fixed-percent / random-max-percent
	// +optional
	Mode Mode `json:"mode"`

	// Value is required when the mode is set to `FixedPodMode` / `FixedPercentPodMod` / `RandomMaxPercentPodMod`.
	// If `FixedPodMode`, provide an integer of pods to do chaos action.
	// If `FixedPercentPodMod`, provide a number from 0-100 to specify the percent of pods the server can do chaos action.
	// IF `RandomMaxPercentPodMod`,  provide a number from 0-100 to specify the max percent of pods to do chaos action
	// +optional
	// +kubebuilder:validation:Enum=one;all;fixed;fixed-percent;random-max-percent
	Value string `json:"value,omitempty"`

	// Macro abstract selector parameters into a structured string (e.g, .cluster.master.all). Every parsed field is
	// represents an inner structure of the selector.
	// In case of invalid macro, the selector will return empty results.
	// Macro conflicts with any other parameter.
	// +optional
	Macro *string `json:"macro,omitempty"`
}
