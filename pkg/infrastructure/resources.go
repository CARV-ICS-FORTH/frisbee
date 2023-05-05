/*
Copyright 2021-2023 ICS-FORTH.

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

package infrastructure

import (
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func RequestIsWithinLimits(ask corev1.ResourceList, allocatable corev1.ResourceList) error {
	var merr *multierror.Error

	if cmp := allocatable.Cpu().Cmp(*ask.Cpu()); cmp < 0 {
		merr = multierror.Append(merr,
			errors.Errorf("CPU: ask[%s] allocatable[%s]", ask.Cpu().String(), allocatable.Cpu().String()),
		)
	}

	if cmp := allocatable.Memory().Cmp(*ask.Memory()); cmp < 0 {
		merr = multierror.Append(merr,
			errors.Errorf("Memory: ask[%s] allocatable[%s]", ask.Memory().String(), allocatable.Memory().String()),
		)
	}

	if cmp := allocatable.Pods().Cmp(*ask.Pods()); cmp < 0 {
		merr = multierror.Append(merr,
			errors.Errorf("Pods: ask[%s] allocatable[%s]", ask.Pods().String(), allocatable.Pods().String()),
		)
	}

	if cmp := allocatable.Storage().Cmp(*ask.Storage()); cmp < 0 {
		merr = multierror.Append(merr,
			errors.Errorf("Storage: ask[%s] allocatable[%s]", ask.Storage().String(), allocatable.Storage().String()),
		)
	}

	if cmp := allocatable.StorageEphemeral().Cmp(*ask.StorageEphemeral()); cmp < 0 {
		merr = multierror.Append(merr,
			errors.Errorf("StorageEphemeral: ask[%s] allocatable[%s]", ask.StorageEphemeral().String(), allocatable.StorageEphemeral().String()),
		)
	}

	return merr.ErrorOrNil()
}

func TotalAllocatableResources(nodeList ...corev1.Node) corev1.ResourceList {
	var (
		cpu       resource.Quantity
		memory    resource.Quantity
		pods      resource.Quantity
		storage   resource.Quantity
		ephemeral resource.Quantity
	)

	for _, node := range nodeList {
		cpu.Add(*node.Status.Allocatable.Cpu())
		memory.Add(*node.Status.Allocatable.Memory())
		pods.Add(*node.Status.Allocatable.Pods())
		storage.Add(*node.Status.Allocatable.Storage())
		ephemeral.Add(*node.Status.Allocatable.StorageEphemeral())
	}

	return corev1.ResourceList{
		corev1.ResourceCPU:              cpu,
		corev1.ResourceMemory:           memory,
		corev1.ResourcePods:             pods,
		corev1.ResourceStorage:          storage,
		corev1.ResourceEphemeralStorage: ephemeral,
	}
}
