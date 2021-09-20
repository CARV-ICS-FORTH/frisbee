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

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Phase is the current status of an object
type Phase string

// These are the valid statuses of services. The following lifecycles are valid:
// PhaseUninitialized -> PhaseFailed
// PhaseUninitialized -> PhaseRunning* -> Completed
// PhaseUninitialized -> PhaseRunning* -> PhaseFailed
// PhaseUninitialized -> PhaseChaos* -> Completed
// PhaseUninitialized -> PhaseRunning* -> PhaseChaos -> Completed
// The asterix (*) Indicate that the same phase may appear recursively.
const (
	// PhaseUninitialized means that the service has been accepted by the system.
	PhaseUninitialized = Phase("")

	// PhasePending means that the service been accepted by the Kubernetes cluster, but one of the dependent
	// conditions is not met yet. This includes the time waiting for logical dependencies (e.g, Run, Success),
	// Ports discovery and rewiring, and placement of Pods.
	PhasePending = Phase("Pending")

	// PhaseRunning means that the service has been bound to a node and all the containers have been started.
	// At least one container is still running or is in the process of being restarted.
	PhaseRunning = Phase("Running")

	// PhaseSuccess means that all containers in the pod have voluntarily terminated
	// with a container exit code of 0, and the system is not going to restart any of these containers.
	PhaseSuccess = Phase("Success")

	// PhaseFailed means that all containers in the pod have terminated, and at least one container has
	// terminated in a failure (exited with a non-zero exit code or was stopped by the system).
	PhaseFailed = Phase("Failed")

	// PhaseChaos indicates a managed abnormal condition such STOP or KILL. In this phase, the controller ignores
	// any subsequent failures and let the system under evaluation to progress as it can.
	PhaseChaos = Phase("Chaos")
)

type Lifecycle struct {
	// Is it needed ?
	Kind string `json:"kind,omitempty"`

	// Is it needed ?
	Name string `json:"name,omitempty"`

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

func (in *Lifecycle) String() string {
	if in.Phase == PhaseFailed {
		if in.Reason == "" {
			in.Reason = "check the logs"
		}

		return fmt.Sprintf("object:%s, Name:%s, phase:%s reason:%s",
			in.Kind,
			in.Name,
			in.Phase,
			in.Reason)
	}

	return fmt.Sprintf("object:%s, Name:%s, phase:%s ",
		in.Kind,
		in.Name,
		in.Phase)
}
