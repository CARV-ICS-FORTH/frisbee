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
	"sort"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClassifierReader interface {
	v1alpha1.StateAggregationFunctions

	// IsDeletable returns true if a job is deletable: it is pending or running
	IsDeletable(job string) (client.Object, bool)

	GetPendingJobs() []client.Object
	GetRunningJobs() []client.Object
	GetSuccessfulJobs() []client.Object
	GetFailedJobs() []client.Object
}

type Classifier struct {
	pendingJobs    map[string]client.Object
	runningJobs    map[string]client.Object
	successfulJobs map[string]client.Object
	failedJobs     map[string]client.Object
}

func (in Classifier) IsZero() bool {
	return len(in.pendingJobs) == 0 &&
		len(in.runningJobs) == 0 &&
		len(in.successfulJobs) == 0 &&
		len(in.failedJobs) == 0
}

func (in *Classifier) Reset() {
	in.pendingJobs = make(map[string]client.Object)
	in.runningJobs = make(map[string]client.Object)
	in.successfulJobs = make(map[string]client.Object)
	in.failedJobs = make(map[string]client.Object)
}

func (in *Classifier) Classify(name string, obj client.Object) {
	if statusAware, getStatus := obj.(ReconcileStatusAware); getStatus {
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
	if statusAware, getStatus := obj.(ReconcileStatusAware); getStatus {
		status := statusAware.GetReconcileStatus()

		if status.Phase.Is(v1alpha1.PhaseFailed) {
			in.failedJobs[name] = obj
		}
	} else {
		ctrl.Log.Info("Object does not implement RecocileStatusAware interface.", "object", obj.GetName())
	}
}

func (in Classifier) IsPending(jobName string) bool {
	_, ok := in.pendingJobs[jobName]

	return ok
}

func (in Classifier) IsRunning(name string) bool {
	_, ok := in.runningJobs[name]

	return ok
}

func (in Classifier) IsSuccessful(name string) bool {
	_, ok := in.successfulJobs[name]

	return ok
}

func (in Classifier) IsFailed(name string) bool {
	_, ok := in.failedJobs[name]

	return ok
}

func (in Classifier) NumPendingJobs() int {
	return len(in.pendingJobs)
}

func (in Classifier) NumRunningJobs() int {
	return len(in.runningJobs)
}

func (in Classifier) NumSuccessfulJobs() int {
	return len(in.successfulJobs)
}

func (in Classifier) NumFailedJobs() int {
	return len(in.failedJobs)
}

func (in Classifier) ListPendingJobs() []string {
	list := make([]string, 0, len(in.pendingJobs))

	for jobName := range in.pendingJobs {
		list = append(list, jobName)
	}

	sort.Strings(list)

	return list
}

func (in Classifier) ListRunningJobs() []string {
	list := make([]string, 0, len(in.runningJobs))

	for jobName := range in.runningJobs {
		list = append(list, jobName)
	}

	sort.Strings(list)

	return list
}

func (in Classifier) ListSuccessfulJobs() []string {
	list := make([]string, 0, len(in.successfulJobs))

	for jobName := range in.successfulJobs {
		list = append(list, jobName)
	}

	sort.Strings(list)

	return list
}

func (in Classifier) ListFailedJobs() []string {
	list := make([]string, 0, len(in.failedJobs))

	for jobName := range in.failedJobs {
		list = append(list, jobName)
	}

	sort.Strings(list)

	return list
}

func (in Classifier) GetPendingJobs() []client.Object {
	list := make([]client.Object, 0, len(in.pendingJobs))

	for _, job := range in.pendingJobs {
		list = append(list, job)
	}

	return list
}

func (in Classifier) GetRunningJobs() []client.Object {
	list := make([]client.Object, 0, len(in.runningJobs))

	for _, job := range in.runningJobs {
		list = append(list, job)
	}

	return list
}

func (in Classifier) GetSuccessfulJobs() []client.Object {
	list := make([]client.Object, 0, len(in.successfulJobs))

	for _, job := range in.successfulJobs {
		list = append(list, job)
	}

	return list
}

func (in Classifier) GetFailedJobs() []client.Object {
	list := make([]client.Object, 0, len(in.failedJobs))

	for _, job := range in.failedJobs {
		list = append(list, job)
	}

	return list
}

// IsDeletable returns if a service can be deleted or not.
// Deletable are only pending or running services, which belong to the SUT (not on the system(
func (in Classifier) IsDeletable(jobName string) (client.Object, bool) {
	if job, exists := in.pendingJobs[jobName]; exists {
		return job, v1alpha1.GetComponentLabel(job) == v1alpha1.ComponentSUT
	}

	if job, exists := in.runningJobs[jobName]; exists {
		// A system service is not deletabled
		return job, v1alpha1.GetComponentLabel(job) == v1alpha1.ComponentSUT
	}

	return nil, false
}
