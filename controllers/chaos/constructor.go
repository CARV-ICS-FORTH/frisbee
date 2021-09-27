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
	"context"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/utils"
	"github.com/fnikolai/frisbee/controllers/utils/selector/service"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func AsPartition(fault *Fault) {
	fault.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "chaos-mesh.org",
		Version: "v1alpha1",
		Kind:    "NetworkChaos",
	})
}

type partition struct {
	spec *v1alpha1.PartitionSpec
}

func (f partition) GetName() string {
	return "partition"
}

func (f *partition) GetFault() *Fault {
	var fault Fault

	AsPartition(&fault)

	return &fault
}

func (f partition) ConstructJob(ctx context.Context, obj *v1alpha1.Chaos) Fault {
	spec := obj.Spec.Partition

	var fault Fault

	{ // spec
		affectedPods := service.Select(ctx, &spec.Selector)

		fault.SetUnstructuredContent(map[string]interface{}{
			"spec": map[string]interface{}{
				"action": "partition",
				"mode":   "all",
				"selector": map[string]interface{}{
					"namespaces": []string{obj.GetNamespace()},
				},
				"direction": "both",
				"target": map[string]interface{}{
					"mode": "all",
					"selector": map[string]interface{}{
						"pods": affectedPods.ByNamespace(),
					},
				},
				"duration": spec.Duration,
			},
		})
	}

	{ // metadata
		// Reverse the spec and metadata order as to avoid overwrites.
		AsPartition(&fault)

		utils.SetOwner(obj, &fault)
		fault.SetName(obj.GetName())
	}

	return fault
}
