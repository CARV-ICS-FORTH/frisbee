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
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClassifierReader interface {
	IsZero() bool
	IsPending(jobName string) bool
	IsRunning(name string) bool
	IsSuccessful(name string) bool
	IsFailed(name string) bool

	// IsDeletable returns true if a job is deletable: it is pending or running
	IsDeletable(jobName string) (client.Object, bool)

	PendingJobs() []client.Object
	RunningJobs() []client.Object
	SuccessfulJobs() []client.Object
	FailedJobs() []client.Object

	PendingJobsNum() int
	RunningJobsNum() int
	SuccessfulJobsNum() int
	FailedJobsNum() int

	PendingJobsList() []string
	RunningJobsList() []string
	SuccessfulJobsList() []string
	FailedJobsList() []string
}

type Classifier struct {
	// pendingJobs involve pending + running
	pendingJobs    map[string]client.Object
	runningJobs    map[string]client.Object
	successfulJobs map[string]client.Object
	failedJobs     map[string]client.Object
}

func (in Classifier) IsZero() bool {
	return in.pendingJobs == nil && in.runningJobs == nil && in.successfulJobs == nil && in.failedJobs == nil
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
		case v1alpha1.PhaseUninitialized, v1alpha1.PhasePending:
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
		logrus.Warn("Not RecocileStatusAware, not setting status for obj:", obj.GetName())
	}
}

// Exclude registers a system service.
// Services classified by this function are not accounted, unless they have failed.
func (in *Classifier) Exclude(name string, obj client.Object) {
	if statusAware, getStatus := obj.(ReconcileStatusAware); getStatus {
		status := statusAware.GetReconcileStatus()

		if status.Phase.Is(v1alpha1.PhaseFailed) {
			in.failedJobs[name] = obj
		}
	} else {
		logrus.Warn("Not RecocileStatusAware, not setting status for obj:", obj.GetName())
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

func (in Classifier) PendingJobsNum() int {
	return len(in.pendingJobs)
}

func (in Classifier) RunningJobsNum() int {
	return len(in.runningJobs)
}

func (in Classifier) SuccessfulJobsNum() int {
	return len(in.successfulJobs)
}

func (in Classifier) FailedJobsNum() int {
	return len(in.failedJobs)
}

func (in Classifier) PendingJobsList() []string {
	list := make([]string, 0, len(in.pendingJobs))

	for jobName := range in.pendingJobs {
		list = append(list, jobName)
	}

	sort.Strings(list)

	return list
}

func (in Classifier) RunningJobsList() []string {
	list := make([]string, 0, len(in.runningJobs))

	for jobName := range in.runningJobs {
		list = append(list, jobName)
	}

	sort.Strings(list)

	return list
}

func (in Classifier) SuccessfulJobsList() []string {
	list := make([]string, 0, len(in.successfulJobs))

	for jobName := range in.successfulJobs {
		list = append(list, jobName)
	}

	sort.Strings(list)

	return list
}

func (in Classifier) FailedJobsList() []string {
	list := make([]string, 0, len(in.failedJobs))

	for jobName := range in.failedJobs {
		list = append(list, jobName)
	}

	sort.Strings(list)

	return list
}

func (in Classifier) PendingJobs() []client.Object {
	list := make([]client.Object, 0, len(in.pendingJobs))

	for _, job := range in.pendingJobs {
		list = append(list, job)
	}

	return list
}

func (in Classifier) RunningJobs() []client.Object {
	list := make([]client.Object, 0, len(in.runningJobs))

	for _, job := range in.runningJobs {
		list = append(list, job)
	}

	return list
}

func (in Classifier) SuccessfulJobs() []client.Object {
	list := make([]client.Object, 0, len(in.successfulJobs))

	for _, job := range in.successfulJobs {
		list = append(list, job)
	}

	return list
}

func (in Classifier) FailedJobs() []client.Object {
	list := make([]client.Object, 0, len(in.failedJobs))

	for _, job := range in.failedJobs {
		list = append(list, job)
	}

	return list
}

func (in Classifier) IsDeletable(jobName string) (client.Object, bool) {
	if job, exists := in.pendingJobs[jobName]; exists {
		return job, true
	}

	if job, exists := in.runningJobs[jobName]; exists {
		return job, true
	}

	return nil, false
}
