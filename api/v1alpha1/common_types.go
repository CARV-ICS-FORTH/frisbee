package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Phase is the current status of an object
type Phase string

const (
	Uninitialized Phase = ""
	Running       Phase = "Running"
	Failed        Phase = "Failed"
	Succeed       Phase = "Succeed"
)

type EtherStatus struct {
	// +kubebuilder:validation:Enum=Running;Failed;Succeed
	Phase Phase `json:"phase,omitempty"`

	// A brief CamelCase message indicating details about why the service is in this Phase.
	// e.g. 'Evicted'
	// +optional
	Reason string `json:"reason,omitempty"`

	// RFC 3339 date and time at which the object was acknowledged by the Kubelet.
	// This is before the Kubelet pulled the container image(s) for the pod.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// Most recently observed status of the object
	// +optional
	EndTime *metav1.Time `json:"endTime,omitempty"`
}
