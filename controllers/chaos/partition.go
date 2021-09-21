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
	"time"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/controllers/common/lifecycle"
	"github.com/fnikolai/frisbee/controllers/common/selector/service"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/util/retry"
)

type partition struct {
	r *Reconciler
}

func (f *partition) generate(ctx context.Context, obj *v1alpha1.Chaos) unstructured.Unstructured {
	affectedPods := service.Select(ctx, &obj.Spec.Partition.Selector)

	f.r.Logger.Info("Inject network partition",
		"name", obj.GetName(),
		"targets", affectedPods.ToString(),
	)

	return unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "chaos-mesh.org/v1alpha1",
			"kind":       "NetworkChaos",
			"metadata": map[string]interface{}{
				"name":      obj.GetName(),
				"namespace": obj.GetNamespace(),
			},
			"spec": map[string]interface{}{
				"action": "partition",
				"mode":   "all",
				"selector": map[string]interface{}{
					"namespaces": []string{obj.GetNamespace()},
				},
				"target": map[string]interface{}{
					"mode": "all",
					"selector": map[string]interface{}{
						"pods": affectedPods.ByNamespace(),
					},
				},
			},
		},
	}
}

func (f *partition) Inject(ctx context.Context, obj *v1alpha1.Chaos) error {
	chaos := f.generate(ctx, obj)

	common.SetOwner(obj, &chaos)

	// occasionally Chaos-Mesh throws an internal timeout. in this case, just retry the operation.
	err := retry.OnError(common.DefaultBackoff, k8errors.IsInternalError, func() error {
		err := f.r.GetClient().Create(ctx, &chaos)

		return errors.Wrapf(err, "create error")
	})
	if err != nil {
		return errors.Wrapf(err, "injection failed")
	}

	// fixme: it may need FilterByName()

	err = lifecycle.New(
		lifecycle.WatchExternal(&chaos, AccessChaosStatus, chaos.GetName()),
		lifecycle.WithFilters(lifecycle.FilterByParent(obj.GetUID())),
		lifecycle.WithAnnotator(&lifecycle.RangeAnnotation{}),
		lifecycle.WithUpdateParentStatus(obj.DeepCopy()),
	).Run(ctx)

	return errors.Wrapf(err, "lifecycle failed")
}

func (f *partition) WaitForDuration(ctx context.Context, obj *v1alpha1.Chaos) error {
	if duration := obj.Spec.Partition.Duration; duration != nil {
		injectionTime := obj.GetCreationTimestamp()
		revokeTime := injectionTime.Add(duration.Duration)

		// if chaos injection + duration is elapsed, return immediately
		if time.Now().After(revokeTime) {
			return nil
		}

		// otherwise, wait for the event to happen in the future
		select {
		case <-ctx.Done():
			return errors.Wrapf(ctx.Err(), "waiting error")
		case <-time.After(time.Until(revokeTime)):
			return nil
		}
	}

	logrus.Warn("Partition without duration")

	return nil
}

func (f *partition) Revoke(ctx context.Context, obj *v1alpha1.Chaos) error {
	// because the internal Chaos object (managed by Chaos controller) owns the external Chaos implementation
	// (managed by Chaos-Mesh) it suffices to remove the internal object, and the external will be garbage collected.
	if err := lifecycle.Delete(ctx, f.r.GetClient(), obj); err != nil {
		return errors.Wrapf(err, "unable to revoke %s", obj.GetName())
	}

	return nil
}
