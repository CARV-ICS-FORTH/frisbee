package v1alpha1

// Mode represents the filter for selecting on of many.
type Mode string

const (
	// OneMode represents that the system will select one object randomly.
	OneMode Mode = "one"
	// AllMode represents that the system will select all objects  regardless of status (not ready or not running pods includes).
	// Use this label carefully.
	AllMode Mode = "all"
	// FixedMode represents that the system will select a specific number of running objects.
	FixedMode Mode = "fixed"
	// FixedPercentMode to specify a fixed % that can be inject chaos action.
	FixedPercentMode Mode = "fixed-percent"
	// RandomMaxPercentMode to specify a maximum % that can be inject chaos action.
	RandomMaxPercentMode Mode = "random-max-percent"
)

// TemplateSelectorSpec defines some selectors for chosing a template
type TemplateSelectorSpec struct {
	// Service selects the service template with the specified value.
	// +optional
	Service string `json:"service,omitempty"`
}

type TemplateSelector struct {
	// Family defines the target family of templates
	// +optional
	Family string `json:"family,omitempty"`

	// Selector contains the rules to select templates (services, failures) within the target family
	Selector TemplateSelectorSpec `json:"selector"`
}

// ServiceSelectorSpec defines the some selectors to select services.
// If the all selectors are empty, all services will be selected.
type ServiceSelectorSpec struct {
	// Services is a map of string keys and a set values that used to select services.
	// The key defines the namespace which services belong,
	// and the each values is a set of service names.
	// +optional
	Services map[string][]string `json:"services,omitempty"`

	// Map of string keys and values that can be used to select objects.
	// A selector based on labels.
	// +optional
	LabelSelectors map[string]string `json:"labelSelectors,omitempty"`

	// ServiceGroup defines the service group where services belong
	// +optional
	ServiceGroup string `json:"servicegroup,omitempty"`

	// Namespaces is a set of namespace to which objects belong.
	// +optional
	Namespaces []string `json:"namespaces,omitempty"`

	// For more options check
	// https://github.com/chaos-mesh/chaos-mesh/blob/31aef289b81a1d713b5a9976a257090da81ac29e/api/v1alpha1/selector.go#L94
}

type ServiceSelector struct {
	// Selector contains the rules to select target
	Selector ServiceSelectorSpec `json:"selector"`

	// Mode defines which of the selected services to use. If undefined, all() is used
	// Supported mode: one / all / fixed / fixed-percent / random-max-percent
	// +optional
	// +kubebuilder:validation:Enum=one;all;fixed;fixed-percent;random-max-percent
	Mode Mode `json:"mode"`

	// Value is required when the mode is set to `FixedPodMode` / `FixedPercentPodMod` / `RandomMaxPercentPodMod`.
	// If `FixedPodMode`, provide an integer of pods to do chaos action.
	// If `FixedPercentPodMod`, provide a number from 0-100 to specify the percent of pods the server can do chaos action.
	// IF `RandomMaxPercentPodMod`,  provide a number from 0-100 to specify the max percent of pods to do chaos action
	// +optional
	Value string `json:"value,omitempty"`
}
