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

package lifecycle

import (
	"fmt"
	"sort"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClassifierReader interface {
	v1alpha1.StateAggregationFunctions

	// ListAll returns a printable form of the names of classified objects
	ListAll() string

	// NumAll returns a printable form of the cardinality of classified objects
	NumAll() string

	// Count returns the number of all registered entities.
	Count() int

	// SystemState returns the state of the SYS services. If all services are running, it should return (false, nil).
	SystemState() (abort bool, err error)

	GetPendingJobs(jobName ...string) []client.Object
	GetRunningJobs(jobName ...string) []client.Object
	GetSuccessfulJobs(jobName ...string) []client.Object
	GetFailedJobs(jobName ...string) []client.Object
}

var _ ClassifierReader = (*Classifier)(nil)

// Classifier splits jobs into Pending, Running, Successful, and Failed.
// To relief the garbage collector, we use a embeddable structure that we reset at every reconciliation cycle.
type Classifier struct {
	pendingJobs     map[string]client.Object
	runningJobs     map[string]client.Object
	successfulJobs  map[string]client.Object
	failedJobs      map[string]client.Object
	terminatingJobs map[string]client.Object
	systemJobs      map[string]client.Object
}

func (in *Classifier) Reset() {
	in.pendingJobs = make(map[string]client.Object)
	in.runningJobs = make(map[string]client.Object)
	in.successfulJobs = make(map[string]client.Object)
	in.failedJobs = make(map[string]client.Object)
	in.terminatingJobs = make(map[string]client.Object)
	in.systemJobs = make(map[string]client.Object)
}

type Convertor func(object client.Object) v1alpha1.Lifecycle

// ClassifyExternal classifies the object based on the custom lifecycle.
func (in *Classifier) ClassifyExternal(name string, obj client.Object, conv Convertor) {
	status := conv(obj)

	if !obj.GetDeletionTimestamp().IsZero() {
		in.terminatingJobs[name] = obj

		return
	}

	switch status.Phase {
	case v1alpha1.PhaseUninitialized:
		// Ignore uninitialized/unscheduled jobs

	case v1alpha1.PhasePending:
		in.pendingJobs[name] = obj

	case v1alpha1.PhaseSuccess:
		in.successfulJobs[name] = obj

	case v1alpha1.PhaseFailed:
		in.failedJobs[name] = obj

	case v1alpha1.PhaseRunning:
		in.runningJobs[name] = obj

	default:
		panic("unhandled lifecycle condition")
	}
}

// Classify the object based on the  standard Frisbee lifecycle.
func (in *Classifier) Classify(name string, obj client.Object) {
	if !obj.GetDeletionTimestamp().IsZero() {
		in.terminatingJobs[name] = obj

		return
	}

	if statusAware, getStatus := obj.(v1alpha1.ReconcileStatusAware); getStatus {
		status := statusAware.GetReconcileStatus()

		// == Handle System resources. ==
		// Resources of this type have the following rules:
		// 1) Are ignored by Pending(), Running(), and Successful() calls, as well as from Count().
		// 2) If they have Failed, they are returned Failed().
		// 3) If they are Running, they are returned by the SystemOK().
		if v1alpha1.GetComponentLabel(obj) == v1alpha1.ComponentSys {
			if status.Phase.Is(v1alpha1.PhaseFailed) {
				in.failedJobs[name] = obj
			} else {
				in.systemJobs[name] = obj
			}

			return
		}

		// Handle SUT resources
		switch status.Phase {
		case v1alpha1.PhaseUninitialized:
			// Ignore uninitialized/unscheduled jobs

		case v1alpha1.PhasePending:
			in.pendingJobs[name] = obj

		case v1alpha1.PhaseSuccess:
			in.successfulJobs[name] = obj

		case v1alpha1.PhaseFailed:
			in.failedJobs[name] = obj

		case v1alpha1.PhaseRunning:
			in.runningJobs[name] = obj

		default:
			panic("unhandled lifecycle condition")
		}
	} else {
		ctrl.Log.Info("Object does not implement RecocileStatusAware interface.", "object", obj.GetName())
	}
}

func (in *Classifier) SystemState() (abort bool, err error) {
	for _, job := range in.systemJobs {
		phase := job.(v1alpha1.ReconcileStatusAware).GetReconcileStatus().Phase

		switch phase {
		case v1alpha1.PhaseRunning:
			continue
		case v1alpha1.PhaseFailed:
			return true, errors.Errorf("System Job '%s' has failed", job.GetName())
		case v1alpha1.PhaseSuccess:
			return true, errors.Errorf("System Job '%s' has terminated", job.GetName())
		case v1alpha1.PhasePending:
			return false, errors.Errorf("System Job '%s' is still pending", job.GetName())
		case v1alpha1.PhaseUninitialized:
			return false, errors.Errorf("System Job '%s' is not yet initialized", job.GetName())
		}
	}

	// All system jobs are running
	return false, nil
}

func (in *Classifier) Count() int {
	return len(in.pendingJobs) +
		len(in.runningJobs) +
		len(in.successfulJobs) +
		len(in.failedJobs)
}

func (in *Classifier) IsPending(job ...string) bool {
	for _, name := range job {
		_, ok := in.pendingJobs[name]
		if !ok {
			return false
		}
	}

	return true
}

func (in *Classifier) IsRunning(job ...string) bool {
	for _, name := range job {
		_, ok := in.runningJobs[name]
		if !ok {
			return false
		}
	}

	return true
}

func (in *Classifier) IsSuccessful(job ...string) bool {
	for _, name := range job {
		_, ok := in.successfulJobs[name]
		if !ok {
			return false
		}
	}

	return true
}

func (in *Classifier) IsFailed(job ...string) bool {
	for _, name := range job {
		_, ok := in.failedJobs[name]
		if !ok {
			return false
		}
	}

	return true
}

func (in *Classifier) IsTerminating(job ...string) bool {
	for _, name := range job {
		_, ok := in.terminatingJobs[name]
		if !ok {
			return false
		}
	}

	return true
}

func (in *Classifier) NumPendingJobs() int {
	return len(in.pendingJobs)
}

func (in *Classifier) NumRunningJobs() int {
	return len(in.runningJobs)
}

func (in *Classifier) NumSuccessfulJobs() int {
	return len(in.successfulJobs)
}

func (in Classifier) NumFailedJobs() int {
	return len(in.failedJobs)
}

func (in Classifier) NumTerminatingJobs() int {
	return len(in.terminatingJobs)
}

func (in *Classifier) NumAll() string {
	return fmt.Sprint(
		"\n * Pending:", in.NumPendingJobs(),
		"\n * Running:", in.NumRunningJobs(),
		"\n * Success:", in.NumSuccessfulJobs(),
		"\n * Failed:", in.NumFailedJobs(),
		"\n * Failed:", in.NumTerminatingJobs(),
		"\n",
	)
}

func (in *Classifier) ListPendingJobs() []string {
	list := make([]string, 0, len(in.pendingJobs))

	for jobName := range in.pendingJobs {
		list = append(list, jobName)
	}

	sort.Strings(list)

	return list
}

func (in *Classifier) ListRunningJobs() []string {
	list := make([]string, 0, len(in.runningJobs))

	for jobName := range in.runningJobs {
		list = append(list, jobName)
	}

	sort.Strings(list)

	return list
}

func (in *Classifier) ListSuccessfulJobs() []string {
	list := make([]string, 0, len(in.successfulJobs))

	for jobName := range in.successfulJobs {
		list = append(list, jobName)
	}

	sort.Strings(list)

	return list
}

func (in *Classifier) ListFailedJobs() []string {
	list := make([]string, 0, len(in.failedJobs))

	for jobName := range in.failedJobs {
		list = append(list, jobName)
	}

	sort.Strings(list)

	return list
}

func (in *Classifier) ListTerminatingJobs() []string {
	list := make([]string, 0, len(in.terminatingJobs))

	for jobName := range in.terminatingJobs {
		list = append(list, jobName)
	}

	sort.Strings(list)

	return list
}

func (in *Classifier) ListAll() string {
	return fmt.Sprint(
		"\n * Pending:", in.ListPendingJobs(),
		"\n * Running:", in.ListRunningJobs(),
		"\n * Success:", in.ListSuccessfulJobs(),
		"\n * Failed:", in.ListFailedJobs(),
		"\n * Terminating:", in.ListTerminatingJobs(),
		"\n",
	)
}

func (in *Classifier) GetPendingJobs(jobNames ...string) []client.Object {
	list := make([]client.Object, 0, len(in.pendingJobs))

	if len(jobNames) == 0 {
		// if no job names are defined, return everything
		for _, job := range in.pendingJobs {
			list = append(list, job)
		}
	} else {
		// otherwise, iterate the list
		for _, job := range jobNames {
			j, exists := in.pendingJobs[job]
			if exists {
				list = append(list, j)
			}
		}
	}

	return list
}

func (in *Classifier) GetRunningJobs(jobNames ...string) []client.Object {
	list := make([]client.Object, 0, len(in.runningJobs))

	if len(jobNames) == 0 {
		// if no job names are defined, return everything
		for _, job := range in.runningJobs {
			list = append(list, job)
		}
	} else {
		// otherwise, iterate the list
		for _, job := range jobNames {
			j, exists := in.runningJobs[job]
			if exists {
				list = append(list, j)
			}
		}
	}

	return list
}

func (in *Classifier) GetSuccessfulJobs(jobNames ...string) []client.Object {
	list := make([]client.Object, 0, len(in.successfulJobs))

	if len(jobNames) == 0 {
		// if no job names are defined, return everything
		for _, job := range in.successfulJobs {
			list = append(list, job)
		}
	} else {
		// otherwise, iterate the list
		for _, job := range jobNames {
			j, exists := in.successfulJobs[job]
			if exists {
				list = append(list, j)
			}
		}
	}

	return list
}

func (in *Classifier) GetFailedJobs(jobNames ...string) []client.Object {
	list := make([]client.Object, 0, len(in.failedJobs))

	if len(jobNames) == 0 {
		// if no job names are defined, return everything
		for _, job := range in.failedJobs {
			list = append(list, job)
		}
	} else {
		// otherwise, iterate the list
		for _, job := range jobNames {
			j, exists := in.failedJobs[job]
			if exists {
				list = append(list, j)
			}
		}
	}

	return list
}

func (in *Classifier) GetTerminatingJobs(jobNames ...string) []client.Object {
	list := make([]client.Object, 0, len(in.failedJobs))

	if len(jobNames) == 0 {
		// if no job names are defined, return everything
		for _, job := range in.terminatingJobs {
			list = append(list, job)
		}
	} else {
		// otherwise, iterate the list
		for _, job := range jobNames {
			j, exists := in.terminatingJobs[job]
			if exists {
				list = append(list, j)
			}
		}
	}

	return list
}
