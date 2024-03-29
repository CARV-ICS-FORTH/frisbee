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
	GenerateObjectFromTemplate `json:",inline"`

	/*
		Preparation of Grouped Environment
	*/

	// TestData defines a volume that will be mounted across the Scenario's Services.
	// +optional
	TestData *TestdataVolume `json:"testData,omitempty"`

	// DefaultDistributionSpec pre-calculates a scoped distribution that can be accessed by other entities
	// using  "distribution.name : default". This default distribution allows us to describe complex relations
	// across features managed by different entities  (e.g, place the largest dataset on the largest node).
	// +optional
	DefaultDistributionSpec *DistributionSpec `json:"defaultDistribution,omitempty"`

	/*
		Job Scheduling
	*/

	// Resources defines how a set of resources will be distributed among the cluster's services.
	// +optional
	Resources *ResourceDistributionSpec `json:"resources,omitempty"`

	// Schedule defines the interval between the creation of services in the group.
	// +optional
	Schedule *TaskSchedulerSpec `json:"schedule,omitempty"`

	// Placement defines rules for placing the containers across the available nodes.
	// +optional
	Placement *PlacementSpec `json:"placement,omitempty"`

	/*
		Execution Flow
	*/

	// Suspend forces the Controller to stop scheduling any new jobs until it is resumed. Defaults to false.
	// +optional
	Suspend *bool `json:"suspend,omitempty"`

	// SuspendWhen automatically sets Suspend to True, when certain conditions are met.
	// +optional
	SuspendWhen *ConditionalExpr `json:"suspendWhen,omitempty"`

	// Tolerate forces the Controller to continue in spite of failed jobs.
	// +optional
	Tolerate *TolerateSpec `json:"tolerate,omitempty"`
}

// ClusterStatus defines the observed state of Cluster.
type ClusterStatus struct {
	Lifecycle `json:",inline"`

	// QueuedJobs is a list of jobs that the controller has to scheduled.
	// +optional
	QueuedJobs []ServiceSpec `json:"queuedJobs,omitempty"`

	// DefaultDistribution keeps the evaluated expression of GenerateObjectFromTemplate.DefaultDistributionSpec.
	// +optional
	DefaultDistribution []float64 `json:"defaultDistribution,omitempty"`

	// ExpectedTimeline is the result of evaluating a timeline distribution into specific points in time.
	// +optional
	ExpectedTimeline Timeline `json:"expectedTimeline,omitempty"`

	// ScheduledJobs points to the next QueuedJobs.
	ScheduledJobs int `json:"scheduledJobs,omitempty"`

	// LastScheduleTime provide information about  the last time a Job was successfully scheduled.
	LastScheduleTime metav1.Time `json:"lastScheduleTime,omitempty"`
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
