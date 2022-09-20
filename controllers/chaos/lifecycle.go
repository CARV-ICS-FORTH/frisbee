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
	"github.com/carv-ics-forth/frisbee/pkg/lifecycle"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *Controller) updateLifecycle(chaos *v1alpha1.Chaos) bool {
	// Skip any CR which are already completed, or uninitialized.
	if chaos.Status.Phase.Is(v1alpha1.PhaseUninitialized, v1alpha1.PhaseSuccess, v1alpha1.PhaseFailed) {
		return false
	}

	return lifecycle.SingleJob(r.view, &chaos.Status.Lifecycle)
}

// ConditionType ...
type ConditionType string

const (
	// ConditionSelected indicates the chaos experiment had correctly selected the target pods
	// where to runJob chaos actions.
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
	// Conditions represents the current global expression of the chaos experiment.
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
			panic(errors.Errorf("unknown expression: %v", condition))
		}
	}

	return
}

func convertChaosLifecycle(obj client.Object) v1alpha1.Lifecycle {
	var parsed v1alpha1ChaosStatus

	if err := mapstructure.Decode(obj.(*GenericFault).Object["status"], &parsed); err != nil {
		return v1alpha1.Lifecycle{
			Phase:   v1alpha1.PhaseFailed,
			Reason:  "Interoperability",
			Message: "cannot parse chaos message",
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
		For each test case only the expression, the expected lifecycle, and name of the test case change.
		Everything else is boilerplate.
		For this reason, we prefer table driven testing.
	*/
	phase, selected, allInjected, allRecovered, paused := parsed.Extract()

	tests := []struct {
		expression bool
		lifecycle  v1alpha1.Lifecycle
	}{
		{ // The experiment is paused. No currently supported
			expression: paused.True(),
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseFailed,
				Reason:  "UnsupportedAction",
				Message: "chaos pausing is not yet supported",
			},
		},

		{ // Starting the experiment
			expression: selected.False() && allInjected.False() && allRecovered.False(),
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhasePending,
				Reason:  "ChaosReStarting",
				Message: "Re-starting Chaos from clean slate",
			},
		},

		{ // Starting the experiment
			expression: phase.Run() && selected.False() && allInjected.True() && allRecovered.True(),
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhasePending,
				Reason:  "ChaosSelectingTargets",
				Message: "Selecting the target pods",
			},
		},

		{ // Injecting faults into targets
			expression: phase.Run() && selected.True() && allInjected.False() && allRecovered.False(),
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhasePending,
				Reason:  "ChaosInjecting",
				Message: "Chaos experiment is in the process of fault injection.",
			},
		},

		{
			// This expression happens when you delete the experiment during a network partition.
			// FIXME: Perhaps it could return a failure.
			expression: phase.Run() && selected.True() && allInjected.False() && allRecovered.True(),
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhasePending,
				Reason:  "ChaosInjecting",
				Message: "Chaos experiment is in the process of fault injection.",
			},
		},

		{ // All Faults are injected to all targets.
			expression: phase.Run() && selected.True() && allInjected.True() && allRecovered.False(),
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseRunning,
				Reason:  "ChaosRunning",
				Message: "The faults have been successfully injected into all target pods",
			},
		},

		{ // Stopping the experiment
			expression: phase.Stop() && selected.True() && allInjected.True() && allRecovered.False(),
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseRunning,
				Reason:  "ChaosTearingDown",
				Message: "removing all the injected faults",
			},
		},

		{ // Faults are stopped but not yet recovered
			expression: phase.Stop() && selected.True() && allInjected.False() && allRecovered.False(),
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseRunning,
				Reason:  "ChaosTearingDown",
				Message: "all faults are removed",
			},
		},

		{ // All faults are removed from all targets
			expression: phase.Stop() && selected.True() && allInjected.False() && allRecovered.True(),
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseSuccess,
				Reason:  "ChaosFinished",
				Message: "all faults are recovered",
			},
		},

		{ // All faults are removed from all targets
			expression: phase.Stop() && selected.False() && allInjected.True() && allRecovered.True(),
			lifecycle: v1alpha1.Lifecycle{
				Phase:   v1alpha1.PhaseFailed,
				Reason:  "TargetNotFound",
				Message: fmt.Sprintf("%v", parsed),
			},
		},
	}

	for _, testcase := range tests {
		if testcase.expression {
			return testcase.lifecycle
		}
	}

	panic(errors.Errorf("unhandled lifecycle conditions. \nphase: %v, \n%v, \n%v, \n%v, \nraw: %v",
		phase, selected, allInjected, allRecovered, obj))
}
