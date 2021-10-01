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

package chaos

import (
	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

type v1alpha1ChaosStatus struct {
	// Conditions represents the current global condition of the chaos
	// +optional
	Conditions []Condition

	// Experiment records the last experiment state.
	Experiment ExperimentStatus
}

type ConditionType string

const (
	ConditionSelected     ConditionType = "Selected"
	ConditionAllInjected  ConditionType = "AllInjected"
	ConditionAllRecovered ConditionType = "AllRecovered"
	ConditionPaused       ConditionType = "Paused"
)

type Condition struct {
	Type   ConditionType
	Status corev1.ConditionStatus
	Reason string
}

type DesiredPhase string

const (
	// RunningPhase target is to make all selected targets (container or pod) into "Injected" phase
	RunningPhase DesiredPhase = "Run"
	// StoppedPhase target  is to make all selected targets (container or pod) into "NotInjected" phase
	StoppedPhase DesiredPhase = "Stop"
)

type ExperimentStatus struct {
	DesiredPhase `mapstructure:",omitempty"`
}

func CalculateLifecycle(fault *Fault) v1alpha1.Lifecycle {
	var parsed v1alpha1ChaosStatus

	if err := mapstructure.Decode(fault.Object["status"], &parsed); err != nil {
		panic(errors.Wrap(err, "unable to parse chaos message"))
	}

	switch phase := parsed.Experiment.DesiredPhase; phase {

	case RunningPhase:
		return v1alpha1.Lifecycle{
			Phase:  v1alpha1.PhaseRunning,
			Reason: "Fault injected",
		}

	case StoppedPhase:
		return v1alpha1.Lifecycle{
			Phase:  v1alpha1.PhaseSuccess,
			Reason: "Fault recovered or paused",
		}

	default:
		panic("this should never happen")
	}
}
