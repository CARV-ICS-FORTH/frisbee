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
	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sort"
)

type ClassifierReader interface {
	v1alpha1.StateAggregationFunctions

	// IsDeletable returns true if a job is deletable: it is pending or running
	IsDeletable(job string) (client.Object, bool)

	// ListAll returns a printable form of the names of classified objects
	ListAll() string

	// NumAll returns a printable form of the cardinality of classified objects
	NumAll() string

	// Count returns the number of all registered entities.
	Count() int

	GetPendingJobs() []client.Object
	GetRunningJobs() []client.Object
	GetSuccessfulJobs() []client.Object
	GetFailedJobs() []client.Object
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
}

func (in *Classifier) Reset() {
	in.pendingJobs = make(map[string]client.Object)
	in.runningJobs = make(map[string]client.Object)
	in.successfulJobs = make(map[string]client.Object)
	in.failedJobs = make(map[string]client.Object)
	in.terminatingJobs = make(map[string]client.Object)
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

// Exclude registers a system service.
// Services classified by this function are not accounted in the lifecycle, unless they have failed.
func (in *Classifier) Exclude(name string, obj client.Object) {
	if statusAware, getStatus := obj.(v1alpha1.ReconcileStatusAware); getStatus {
		status := statusAware.GetReconcileStatus()

		if status.Phase.Is(v1alpha1.PhaseFailed) {
			in.failedJobs[name] = obj
		}
	} else {
		ctrl.Log.Info("Object does not implement RecocileStatusAware interface.", "object", obj.GetName())
	}
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

func (in *Classifier) GetPendingJobs() []client.Object {
	list := make([]client.Object, 0, len(in.pendingJobs))

	for _, job := range in.pendingJobs {
		list = append(list, job)
	}

	return list
}

func (in *Classifier) GetRunningJobs() []client.Object {
	list := make([]client.Object, 0, len(in.runningJobs))

	for _, job := range in.runningJobs {
		list = append(list, job)
	}

	return list
}

func (in *Classifier) GetSuccessfulJobs() []client.Object {
	list := make([]client.Object, 0, len(in.successfulJobs))

	for _, job := range in.successfulJobs {
		list = append(list, job)
	}

	return list
}

func (in *Classifier) GetFailedJobs() []client.Object {
	list := make([]client.Object, 0, len(in.failedJobs))

	for _, job := range in.failedJobs {
		list = append(list, job)
	}

	return list
}

func (in *Classifier) GetTerminatingJobs() []client.Object {
	list := make([]client.Object, 0, len(in.failedJobs))

	for _, job := range in.terminatingJobs {
		list = append(list, job)
	}

	return list
}

// IsDeletable returns if a service can be deleted or not.
// Deletable are only pending or running services, which belong to the SUT (not on the system(
func (in *Classifier) IsDeletable(jobName string) (client.Object, bool) {
	if job, exists := in.pendingJobs[jobName]; exists {
		return job, v1alpha1.GetComponentLabel(job) == v1alpha1.ComponentSUT
	}

	if job, exists := in.runningJobs[jobName]; exists {
		// A system service is not deletabled
		return job, v1alpha1.GetComponentLabel(job) == v1alpha1.ComponentSUT
	}

	return nil, false
}
