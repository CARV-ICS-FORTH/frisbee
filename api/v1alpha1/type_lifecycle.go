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
	// PhaseUninitialized means that request is not yet accepted by the controller.
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
)

func (p Phase) toInt() int {
	switch p {
	case PhaseUninitialized:
		return 0
	case PhasePending:
		return 1
	case PhaseRunning:
		return 2
	case PhaseSuccess:
		return 3
	case PhaseFailed:
		return 4
	default:
		panic("invalid phase")
	}
}

// IsValid return true if the given phase precedes the reference phase.
func (p Phase) IsValid(ref Phase) bool {
	return p.toInt() < ref.toInt()
}

func (p Phase) Equals(ref Phase) bool {
	return p == ref
}

type Lifecycle struct {
	Phase Phase `json:"phase,omitempty"`

	// A brief CamelCase message indicating details about why the service is in this Phase.
	// e.g. 'Evicted'
	// +optional
	Reason string `json:"reason,omitempty"`
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
