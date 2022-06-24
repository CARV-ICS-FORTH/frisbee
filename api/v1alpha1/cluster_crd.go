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

type PlacementSpec struct {
	// Collocate will place all the services within the same node.
	// +optional
	Collocate bool `json:"collocate"`

	// ConflictsWith points to another cluster whose services cannot be located with this one.
	// Used, for example, to place master nodes and slave nodes on different failures domains
	ConflictsWith []string `json:"conflictsWith,omitempty"`

	// Nodes will place all the services within the same specific node.
	// +optional
	Nodes []string `json:"nodes,omitempty"`
}

// ClusterSpec defines the desired state of Cluster.
type ClusterSpec struct {
	GenerateFromTemplate `json:",inline"`

	// Tolerate specifies the conditions under which the cluster will fail. If left undefined, the cluster
	// will fail immediately when a service has failed.
	// +optional
	Tolerate TolerateSpec `json:"tolerate,omitempty"`

	// Schedule defines the interval between the creation of services within the group. ExecutedActions creation is not
	// supported in collocated mode. Since Pods are intended to be disposable and replaceable, we cannot add a
	// container to a Pod once it has been created
	// +optional
	Schedule *SchedulerSpec `json:"schedule,omitempty"`

	// Placement defines rules for placing the containers across the available nodes.
	// +optional
	Placement *PlacementSpec `json:"placement,omitempty"`

	// Suspend flag tells the controller to suspend subsequent executions, it does
	// not apply to already started executions.  Defaults to false.
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
