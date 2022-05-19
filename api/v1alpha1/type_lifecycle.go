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

const (
	// ComponentSys is a component that belongs to Frisbee. Such components can be excluded from Chaos events.
	ComponentSys = "SYS"

	// ComponentSUT is a component that belongs to the system under testing
	ComponentSUT = "SUT"
)

// https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
const (
	// LabelCreatedBy points to the controller/user who created this resource
	LabelCreatedBy = "plan.frisbee.io/created-by"

	// LabelPartOfPlan points to the name of a higher level application this one is part of.
	LabelPartOfPlan = "plan.frisbee.io/part-of"

	// LabelComponent describes the role of the component within the architecture.
	// It can be SUT (for system under service) or SYS (if it's a frisbee component like Grafana).
	LabelComponent = "plan.frisbee.io/component"
)

// Phase is a simple, high-level summary of where the Object is in its lifecycle.
type Phase string

// These are the valid statuses of services. The following lifecycles are valid:
// PhaseUninitialized -> PhaseFailed
// PhaseUninitialized -> PhaseRunning* -> Completed
// PhaseUninitialized -> PhaseRunning* -> PhaseFailed
// PhaseUninitialized -> PhaseChaos* -> Completed
// PhaseUninitialized -> PhaseRunning* -> PhaseChaos -> Completed
// The asterix (*) Indicate that the same phase may appear recursively.
const (
	// PhaseUninitialized means that request is not yet accepted by the controller.
	PhaseUninitialized = Phase("")

	// PhasePending means that the CR has been accepted by the Kubernetes cluster, but one of the child
	// jobs has not been created. This includes the time waiting for logical dependencies, Ports discovery,
	// data rewiring, and placement of Pods.
	PhasePending = Phase("Pending")

	// PhaseRunning means that all the child jobs of a CR have been created, and at least one job
	// is still running.
	PhaseRunning = Phase("Running")

	// PhaseSuccess means that all jobs in a CR have voluntarily exited, and the system is not going
	// to restart any of these Jobs.
	PhaseSuccess = Phase("Success")

	// PhaseFailed means that at least one job of the CR has terminated in a failure (exited with a
	// non-zero exit code or was stopped by the system).
	PhaseFailed = Phase("Failed")
)

func (p Phase) Is(refs ...Phase) bool {
	for _, ref := range refs {
		if p == ref {
			return true
		}
	}

	return false
}

type Lifecycle struct {
	// Phase is a simple, high-level summary of where the Object is in its lifecycle.
	// The conditions array, the reason and message fields, and the individual container status
	// arrays contain more detail about the pod's status.
	Phase Phase `json:"phase,omitempty"`

	// Reason is A brief CamelCase message indicating details about why the service is in this Phase.
	// e.g. 'Evicted'
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message provides more details for understanding the Reason.
	Message string `json:"message,omitempty"`

	// Conditions describe sequences of events that warrant the present Phase.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (in *Lifecycle) String() string {
	if in.Phase == PhaseFailed {
		if in.Reason == "" {
			in.Reason = "check the logs"
		}

		return fmt.Sprintf("phase:%s reason:%s", in.Phase, in.Reason)
	}

	return fmt.Sprintf("phase:%s ", in.Phase)
}

// ConditionType is a valid value for WorkflowCondition.Type
type ConditionType string

func (t ConditionType) String() string {
	return string(t)
}

// These are valid conditions of pod.
const (
	// ConditionCRInitialized indicates whether the workflow has been initialized
	ConditionCRInitialized = ConditionType("initialized")

	ConditionJobFailed = ConditionType("hasFailedJobs")

	// ConditionAllJobsScheduled indicates whether all actions in the workflow have been executed.
	ConditionAllJobsScheduled = ConditionType("AllJobsAreScheduled")

	// ConditionAllJobsCompleted indicates whether all actions in the workflow have been completed.
	ConditionAllJobsCompleted = ConditionType("complete")

	// ConditionTerminated indicates the user-defined conditions are met.
	ConditionTerminated = ConditionType("terminated")
)
