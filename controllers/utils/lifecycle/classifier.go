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

package lifecycle

import (
	"sort"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Classifier struct {
	// activeJobs involve pending + running
	activeJobs     map[string]client.Object
	runningJobs    map[string]client.Object
	successfulJobs map[string]client.Object
	failedJobs     map[string]client.Object
}

func (in *Classifier) Reset() {
	in.activeJobs = make(map[string]client.Object)
	in.runningJobs = make(map[string]client.Object)
	in.successfulJobs = make(map[string]client.Object)
	in.failedJobs = make(map[string]client.Object)
}

func (in *Classifier) Classify(name string, obj client.Object) {
	if statusAware, getStatus := obj.(ReconcileStatusAware); getStatus {
		status := statusAware.GetReconcileStatus()

		switch status.Phase {
		case v1alpha1.PhaseUninitialized, v1alpha1.PhasePending:
			in.activeJobs[name] = obj

		case v1alpha1.PhaseSuccess:
			in.successfulJobs[name] = obj

		case v1alpha1.PhaseFailed:
			in.failedJobs[name] = obj

		case v1alpha1.PhaseRunning:
			in.runningJobs[name] = obj
			in.activeJobs[name] = obj

		default:
			panic("unhandled lifecycle condition")
		}
	} else {
		logrus.Warn("Not RecocileStatusAware, not setting status for obj:", obj.GetName())
	}
}

func (in *Classifier) IsActive(jobName string) bool {
	_, ok := in.activeJobs[jobName]

	return ok
}

func (in *Classifier) IsRunning(name string) bool {
	_, ok := in.runningJobs[name]

	return ok
}

func (in *Classifier) IsSuccessful(name string) bool {
	_, ok := in.successfulJobs[name]

	return ok
}

func (in *Classifier) IsFailed(name string) bool {
	_, ok := in.failedJobs[name]

	return ok
}

func (in *Classifier) NumActiveJobs() int {
	return len(in.activeJobs)
}

func (in *Classifier) NumRunningJobs() int {
	return len(in.runningJobs)
}

func (in *Classifier) NumSuccessfulJobs() int {
	return len(in.successfulJobs)
}

func (in *Classifier) NumFailedJobs() int {
	return len(in.failedJobs)
}

func (in *Classifier) ActiveList() []string {
	list := make([]string, 0, len(in.activeJobs))

	for jobName := range in.activeJobs {
		list = append(list, jobName)
	}

	sort.Strings(list)

	return list
}

func (in *Classifier) RunningList() []string {
	list := make([]string, 0, len(in.runningJobs))

	for jobName := range in.runningJobs {
		list = append(list, jobName)
	}

	sort.Strings(list)

	return list
}

func (in *Classifier) SuccessfulList() []string {
	list := make([]string, 0, len(in.successfulJobs))

	for jobName := range in.successfulJobs {
		list = append(list, jobName)
	}

	sort.Strings(list)

	return list
}

func (in *Classifier) FailedList() []string {
	list := make([]string, 0, len(in.failedJobs))

	for jobName := range in.failedJobs {
		list = append(list, jobName)
	}

	sort.Strings(list)

	return list
}

func (in *Classifier) ActiveJobs() []client.Object {
	list := make([]client.Object, 0, len(in.activeJobs))

	for _, job := range in.activeJobs {
		list = append(list, job)
	}

	return list
}

func (in *Classifier) RunningJobs() []client.Object {
	list := make([]client.Object, 0, len(in.runningJobs))

	for _, job := range in.runningJobs {
		list = append(list, job)
	}

	return list
}

func (in *Classifier) SuccessfulJobs() []client.Object {
	list := make([]client.Object, 0, len(in.successfulJobs))

	for _, job := range in.successfulJobs {
		list = append(list, job)
	}

	return list
}

func (in *Classifier) FailedJobs() []client.Object {
	list := make([]client.Object, 0, len(in.failedJobs))

	for _, job := range in.failedJobs {
		list = append(list, job)
	}

	return list
}

func WaitUntil(src <-chan *v1alpha1.Lifecycle, phase v1alpha1.Phase) error {
	for lf := range src {
		if lf.Phase.Equals(v1alpha1.PhaseRunning) {
			break
		}

		if lf.Phase.IsValid(v1alpha1.PhaseRunning) {
			continue
		}

		return errors.Errorf("expected %s but got %s", phase, lf.Phase)
	}

	return nil
}
