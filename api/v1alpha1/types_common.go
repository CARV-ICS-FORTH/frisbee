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

package v1alpha1

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/robfig/cron/v3"
)

const (
	// LabelManagedBy binds an object  back to a specific controller.
	LabelManagedBy = "frisbee-controller"

	// BelongsToWorkflow is a label that is passed from parent to the child, in order to identify all
	// the various objects that belong to a specific workflow.
	BelongsToWorkflow = "frisbee-workflow"
)

const (
	idListSeparator = " "
)

// SList is a service list
type SList []*Service

func (in SList) ToString() string {
	if len(in) == 0 {
		return ""
	}

	return strings.Join(in.GetNames(), idListSeparator)
}

func (in SList) GetNames() []string {
	if len(in) == 0 {
		return nil
	}

	names := make([]string, len(in))

	for i, service := range in {
		names[i] = service.GetName()
	}

	return names
}

func (in SList) GetNamespaces() []string {
	if len(in) == 0 {
		return nil
	}

	namespace := make([]string, len(in))

	for i, service := range in {
		namespace[i] = service.GetNamespace()
	}

	return namespace
}

func (in SList) GetNamespacedNames() []string {
	if len(in) == 0 {
		return nil
	}

	namespacedName := make([]string, len(in))

	for i, service := range in {
		namespacedName[i] = fmt.Sprintf("%s/%s", service.GetNamespace(), service.GetName())
	}

	return namespacedName
}

// ByNamespace return the services by the namespace they belong to.
func (in SList) ByNamespace() map[string][]string {
	if len(in) == 0 {
		return nil
	}

	all := make(map[string][]string)

	for _, s := range in {
		// get namespace
		sublist := all[s.GetNamespace()]

		// append service to the namespace
		sublist = append(sublist, s.GetName())

		// update namespace
		all[s.GetNamespace()] = sublist
	}

	return all
}

// Yield takes a list and return its elements one by one, with the frequency defined in the cronspec.
func (in SList) Yield(ctx context.Context, schedule *SchedulerSpec) <-chan *Service {

	switch {
	case len(in) == 0:
		return nil

	case schedule == nil:
		ret := make(chan *Service, len(in))

		for _, instance := range in {
			ret <- instance
		}

		close(ret)

		return ret

	case schedule != nil:
		ret := make(chan *Service)

		job := cron.New()
		cronspec := schedule.Cron
		stop := make(chan struct{})

		var last uint32

		_, err := job.AddFunc(cronspec, func() {
			defer atomic.AddUint32(&last, 1)

			v := atomic.LoadUint32(&last)

			switch {
			case v < uint32(len(in)):
				ret <- in[last]
			case v == uint32(len(in)):
				close(stop)
			case v > uint32(len(in)):
				return
			}
		})
		if err != nil {
			close(ret)

			return ret
		}

		job.Start()

		go func() {
			defer close(ret)

			select {
			case <-ctx.Done():
			case <-stop:
			}

			until := job.Stop()
			<-until.Done()
		}()

		return ret
	}

	return nil
}

// ActionList is a list of actions
type ActionList []Action

func (in ActionList) ToString() string {
	if len(in) == 0 {
		return ""
	}

	return strings.Join(in.GetNames(), idListSeparator)
}

func (in ActionList) GetNames() []string {
	if len(in) == 0 {
		return nil
	}

	names := make([]string, len(in))

	for i, action := range in {
		names[i] = action.Name
	}

	return names
}

// FromTemplate generates a spec by parameterizing the templateRef with the given inputs.
type FromTemplate struct {
	// TemplateRef refers to a service template. It conflicts with Service.
	TemplateRef string `json:"templateRef"`

	// Inputs is a list of inputs passed to the objects.
	// +optional
	Inputs map[string]string `json:"inputs,omitempty"`
}
