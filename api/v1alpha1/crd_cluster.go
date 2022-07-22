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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Cluster is the Schema for the clusters API.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

// PlacementSpec defines rules for placing the containers across the available nodes.
type PlacementSpec struct {
	// Collocate will place all the Services of this Cluster within the same node.
	// +optional
	Collocate bool `json:"collocate"`

	// ConflictsWith points to another Cluster whose Services cannot be located with this one.
	// For example, this is needed for placing the master nodes on a different failure domain than the slave nodes.
	ConflictsWith []string `json:"conflictsWith,omitempty"`

	// Nodes will place all the Services of this Cluster within the specific set of nodes.
	// +optional
	Nodes []string `json:"nodes,omitempty"`
}

// ClusterSpec defines the desired state of Cluster.
type ClusterSpec struct {
	GenerateFromTemplate `json:",inline"`

	// Tolerate specifies the conditions under which the cluster will fail. If undefined, the cluster fails
	// immediately when a service has failed.
	// +optional
	Tolerate *TolerateSpec `json:"tolerate,omitempty"`

	// Schedule defines the interval between the creation of services in the group.
	// +optional
	Schedule *SchedulerSpec `json:"schedule,omitempty"`

	// Placement defines rules for placing the containers across the available nodes.
	// +optional
	Placement *PlacementSpec `json:"placement,omitempty"`

	// Suspend flag tells the controller to suspend subsequent executions, it does not apply to already started
	// executions.  Defaults to false.
	// +optional
	Suspend *bool `json:"suspend,omitempty"`
}

// ClusterStatus defines the observed state of Cluster.
type ClusterStatus struct {
	Lifecycle `json:",inline"`

	// QueuedJobs is a list of services scheduled for creation by the cluster.
	// +optional
	QueuedJobs []ServiceSpec `json:"queuedJobs,omitempty"`

	// ScheduledJobs points to the next QueuedJobs.
	ScheduledJobs int `json:"scheduledJobs,omitempty"`

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

// ClusterList contains a list of Cluster.
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
