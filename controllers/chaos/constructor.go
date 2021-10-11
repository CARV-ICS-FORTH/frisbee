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
	"github.com/fnikolai/frisbee/controllers/service/helpers"
	"github.com/fnikolai/frisbee/controllers/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Fault = unstructured.Unstructured

type chaoHandler interface {
	GetFault(r *Controller) *Fault

	Inject(ctx context.Context, r *Controller) error
}

func dispatch(chaos *v1alpha1.Chaos) chaoHandler {
	switch chaos.Spec.Type {
	case v1alpha1.FaultPartition:
		return &partitionHandler{cr: chaos}

	case v1alpha1.FaultKill:
		return &killHandler{cr: chaos}
	default:
		panic("should never happen")
	}
}

func AsPartition(fault *Fault) {
	fault.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "chaos-mesh.org",
		Version: "v1alpha1",
		Kind:    "NetworkChaos",
	})
}

func AsKill(fault *Fault) {
	fault.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "chaos-mesh.org",
		Version: "v1alpha1",
		Kind:    "PodChaos",
	})
}

/*
	Network Partition Handler
*/

type partitionHandler struct {
	cr *v1alpha1.Chaos
}

func (h partitionHandler) GetFault(r *Controller) *Fault {
	var fault Fault

	AsPartition(&fault)

	utils.SetOwner(r, h.cr, &fault)
	fault.SetName(h.cr.GetName())

	return &fault
}

func (h partitionHandler) Inject(ctx context.Context, r *Controller) error {
	spec := h.cr.Spec.Partition

	var fault Fault

	affectedPods := helpers.Select(ctx, r, h.cr.GetNamespace(), &spec.Selector)

	{ // spec
		fault.SetUnstructuredContent(map[string]interface{}{
			"spec": map[string]interface{}{
				"action": "partition",
				"mode":   "all",
				"selector": map[string]interface{}{
					"namespaces": []string{h.cr.GetNamespace()},
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

	AsPartition(&fault)

	utils.SetOwner(r, h.cr, &fault)
	fault.SetName(h.cr.GetName())

	if err := utils.Create(ctx, r, &fault); err != nil {
		return errors.Wrapf(err, "cannot inject fault")
	}

	return nil
}

/*
	Service Killer
*/

type killHandler struct {
	cr *v1alpha1.Chaos
}

func (h *killHandler) GetFault(r *Controller) *Fault {
	var fault Fault

	AsKill(&fault)

	utils.SetOwner(r, h.cr, &fault)
	fault.SetName(h.cr.GetName())

	return &fault
}

func (h killHandler) Inject(ctx context.Context, r *Controller) error {
	spec := h.cr.Spec.Kill

	var fault Fault

	affectedPods := helpers.Select(ctx, r, h.cr.GetNamespace(), &spec.Selector)

	{ // spec
		fault.SetUnstructuredContent(map[string]interface{}{
			"spec": map[string]interface{}{
				"action": "pod-kill",
				"mode":   "all",
				"selector": map[string]interface{}{
					"pods": affectedPods.ByNamespace(),
				},
			},
		})

		logrus.Warn("KILL ", affectedPods.ToString())
	}

	AsKill(&fault)

	utils.SetOwner(r, h.cr, &fault)
	fault.SetName(h.cr.GetName())

	if err := utils.Create(ctx, r, &fault); err != nil {
		return errors.Wrapf(err, "cannot inject fault")
	}

	return nil
}
