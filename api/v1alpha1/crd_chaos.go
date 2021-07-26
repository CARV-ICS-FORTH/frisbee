package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&Chaos{}, &ChaosList{})
}

type FaultType string

const (
	FaultPartition = FaultType("partition")
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type Chaos struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the behavior of the object
	Spec ChaosSpec `json:"spec"`

	// Most recently observed status of the object
	Status ChaosStatus `json:"status,omitempty"`
}

type ChaosSpec struct {
	// Type indicate the type of the injected fault
	// +kubebuilder:validation:Enum=partition;
	Type FaultType `json:"type"`

	*EmbedFaultType `json:",inline"`
}

type EmbedFaultType struct {
	// +optional
	Partition *PartitionSpec `json:"partition,omitempty"`
}

type PartitionSpec struct {
	Selector ServiceSelector `json:"selector"`

	// +optional
	Duration *metav1.Duration `json:"duration,omitempty"`
}

type ChaosStatus struct {
	Lifecycle `json:",inline"`
}

func (s *Chaos) GetLifecycle() Lifecycle {
	return s.Status.Lifecycle
}

func (s *Chaos) SetLifecycle(lifecycle Lifecycle) {
	s.Status.Lifecycle = lifecycle
}

// +kubebuilder:object:root=true
type ChaosList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Chaos `json:"items"`
}
