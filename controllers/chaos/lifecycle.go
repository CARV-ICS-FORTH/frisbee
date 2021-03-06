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

package chaos

import (
	"fmt"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

// ConditionType ...
type ConditionType string

const (
	// ConditionSelected indicates the chaos experiment had correctly selected the target pods
	// where to inject chaos actions.
	ConditionSelected ConditionType = "Selected"

	// ConditionAllInjected indicates the faults have been successfully injected to all target pods.
	ConditionAllInjected ConditionType = "AllInjected"

	// ConditionAllRecovered indicates the injected faults have been successfully restored from all target pods.
	ConditionAllRecovered ConditionType = "AllRecovered"

	// ConditionPaused  indicates the chaos experiment is in the "Paused" step.
	ConditionPaused ConditionType = "Paused"
)

type Condition struct {
	Type   ConditionType
	Status corev1.ConditionStatus
	Reason string
}

func (c Condition) True() bool {
	return c.Status == corev1.ConditionTrue
}

func (c Condition) False() bool {
	return c.Status == corev1.ConditionFalse
}

type DesiredPhase string

func (d DesiredPhase) Run() bool {
	return d == RunningPhase || d == ""
}

func (d DesiredPhase) Stop() bool {
	return d == StoppedPhase
}

const (
	// RunningPhase target is to make all selected targets (container or pod) into "Injected" phase.
	RunningPhase DesiredPhase = "Run"
	// StoppedPhase target  is to make all selected targets (container or pod) into "NotInjected" phase.
	StoppedPhase DesiredPhase = "Stop"
)

type ExperimentStatus struct {
	DesiredPhase `mapstructure:",omitempty"`
}

type v1alpha1ChaosStatus struct {
	// Conditions represents the current global condition of the chaos experiment.
	// The actual status of current chaos experiments can be inferred from these conditions.
	// +optional
	Conditions []Condition

	// Experiment records the last experiment state.
	Experiment ExperimentStatus
}

func (s v1alpha1ChaosStatus) Extract() (phase DesiredPhase, selected, allInjected, allRecovered, paused Condition) {
	phase = s.Experiment.DesiredPhase

	for i, condition := range s.Conditions {
		switch condition.Type {
		case ConditionSelected:
			selected = s.Conditions[i]

		case ConditionAllInjected:
			allInjected = s.Conditions[i]

		case ConditionAllRecovered:
			allRecovered = s.Conditions[i]

		case ConditionPaused:
			paused = s.Conditions[i]
		default:
			panic(errors.Errorf("unknown condition: %v", condition))
		}
	}

	return
}

func calculateLifecycle(cr *v1alpha1.Chaos, fault *GenericFault) v1alpha1.Lifecycle {
	// Skip any CR which are already completed, or uninitialized.
	if cr.Status.Phase.Is(v1alpha1.PhaseUninitialized, v1alpha1.PhaseSuccess, v1alpha1.PhaseFailed) {
		return cr.Status.Lifecycle
	}

	return convertLifecycle(fault)
}

/*
ConvertLifecycle infers the Frisbee Lifecycle from the of a Chaos-Mesh experiment.

In Chaos Mesh, the life cycle of a chaos experiment is divided into four steps, according to its running process:

	* Injecting: Chaos experiment is in the process of fault injection. Normally, this step lasts for a short time.
	If the "Injecting" step lasts a long time, it may be due to some exceptions in the chaos experiment.

	* Running: State the faults have been successfully injected into all target pods, the chaos experiment starts running.

	* Paused: when executing a paused process for a running chaos experiment, Chaos Mesh restores the injected
	faults from all target pods, which indicates the experiment is paused.

	* Finished: if the duration parameter of the experiment is configured, and when the experiment runs it up,
	Chaos Mesh restores the injected faults from all target pods, which indicates that the experiment is finished.
*/
func convertLifecycle(fault *GenericFault) v1alpha1.Lifecycle {
	var parsed v1alpha1ChaosStatus

	if err := mapstructure.Decode(fault.Object["status"], &parsed); err != nil {
		return v1alpha1.Lifecycle{
			Phase:   v1alpha1.PhaseFailed,
			Reason:  "Interoperability",
			Message: "unable to parse chaos message",
		}
	}

	if parsed.Conditions == nil {
		return v1alpha1.Lifecycle{
			Phase:   v1alpha1.PhasePending,
			Reason:  "ChaosStarted",
			Message: "Chaos experiment has just started.",
		}
	}

	/*
		There is a lot of duplication in our tests.
		For each test case only the condition, the expected outcome, and name of the test case change.
		Everything else is boilerplate.
		For this reason, we prefer table driven testing.
	*/
	phase, selected, allInjected, allRecovered, paused := parsed.Extract()

	tests := []struct {
		condition bool
		outcome   v1alpha1.Lifecycle
	}{
		{ // The experiment is paused. No currently supported
			condition: paused.True(),
			outcome: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseFailed,
				Reason:  "UnsupportedAction",
				Message: "chaos pausing is not yet supported",
			},
		},

		{ // Starting the experiment
			condition: selected.False() && allInjected.False() && allRecovered.False(),
			outcome: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhasePending,
				Reason:  "ChaosReStarting",
				Message: "Re-starting Chaos from clean slate",
			},
		},

		{ // Starting the experiment
			condition: phase.Run() && selected.False() && allInjected.True() && allRecovered.True(),
			outcome: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhasePending,
				Reason:  "ChaosSelectingTargets",
				Message: "Selecting the target pods",
			},
		},

		{ // Injecting faults into targets
			condition: phase.Run() && selected.True() && allInjected.False() && allRecovered.False(),
			outcome: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhasePending,
				Reason:  "ChaosInjecting",
				Message: "Chaos experiment is in the process of fault injection.",
			},
		},

		{
			// This condition happens when you delete the experiment during a network partition.
			// FIXME: Perhaps it could return a failure.
			condition: phase.Run() && selected.True() && allInjected.False() && allRecovered.True(),
			outcome: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhasePending,
				Reason:  "ChaosInjecting",
				Message: "Chaos experiment is in the process of fault injection.",
			},
		},

		{ // All Faults are injected to all targets.
			condition: phase.Run() && selected.True() && allInjected.True() && allRecovered.False(),
			outcome: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseRunning,
				Reason:  "ChaosRunning",
				Message: "The faults have been successfully injected into all target pods",
			},
		},

		{ // Stopping the experiment
			condition: phase.Stop() && selected.True() && allInjected.True() && allRecovered.False(),
			outcome: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseRunning,
				Reason:  "ChaosTearingDown",
				Message: "removing all the injected faults",
			},
		},

		{ // Faults are stopped but not yet recovered
			condition: phase.Stop() && selected.True() && allInjected.False() && allRecovered.False(),
			outcome: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseRunning,
				Reason:  "ChaosTearingDown",
				Message: "all faults are removed",
			},
		},

		{ // All faults are removed from all targets
			condition: phase.Stop() && selected.True() && allInjected.False() && allRecovered.True(),
			outcome: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseSuccess,
				Reason:  "ChaosFinished",
				Message: "all faults are recovered",
			},
		},

		{ // All faults are removed from all targets
			condition: phase.Stop() && selected.False() && allInjected.True() && allRecovered.True(),
			outcome: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseFailed,
				Reason:  "TargetNotFound",
				Message: fmt.Sprintf("%v", parsed),
			},
		},
	}

	for _, testcase := range tests {
		if testcase.condition {
			return testcase.outcome
		}
	}

	panic(errors.Errorf("unhandled lifecycle conditions. \nphase: %v, \n%v, \n%v, \n%v, \nraw: %v",
		phase, selected, allInjected, allRecovered, fault))
}
