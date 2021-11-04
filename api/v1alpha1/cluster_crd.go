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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TolerateSpec specifies the system's ability to continue operating despite failures or malfunctions.
// If tolerate is enable, a cluster will be "alive" even if some services have failed.
// Such failures are likely to happen as part of a Chaos experiment.
type TolerateSpec struct {
	// FailedServices indicate the number of services that may fail before the cluster fails itself.
	FailedServices int `json:"failedServices"`
}

func (in TolerateSpec) String() string {
	if in.FailedServices == 0 {
		return "None"
	}

	return fmt.Sprintf("FailedServices:%d", in.FailedServices)
}

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {
	FromTemplate `json:",inline"`

	// Schedule defines the interval between the creation of services within the group. Executed creation is not
	// supported in collocated mode. Since Pods are intended to be disposable and replaceable, we cannot add a
	// container to a Pod once it has been created
	// +optional
	Schedule *SchedulerSpec `json:"schedule,omitempty"`

	// Tolerate specifies the conditions under which the cluster will fail. If left undefined, the cluster
	// will fail immediately when a service has failed.
	// +optional
	Tolerate TolerateSpec `json:"tolerate,omitempty"`

	// Domain specifies the location where Service will be placed. For this to work,
	// the nodes included in the domain must have the label domain:{{domain-name}}.
	// for the moment simply match domain to a specific node. this will change in the future
	// +optional
	Domain string `json:"domain,omitempty"`

	// Suspend flag tells the controller to suspend subsequent executions, it does
	// not apply to already started executions.  Defaults to false.
	// +optional
	Suspend *bool `json:"suspend,omitempty"`
}

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	Lifecycle `json:",inline"`

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Expected is a list of services scheduled for creation by the cluster.
	// +optional
	Expected []ServiceSpec `json:"expected,omitempty"`

	// LastScheduleJob points to the next Expect Job
	LastScheduleJob int `json:"lastScheduleJob,omitempty"`

	// LastScheduleTime provide information about  the last time a Service was successfully scheduled.
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`
}

func (in *Cluster) GetReconcileStatus() Lifecycle {
	return in.Status.Lifecycle
}

func (in *Cluster) SetReconcileStatus(lifecycle Lifecycle) {
	in.Status.Lifecycle = lifecycle
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Cluster is the Schema for the clusters API
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
