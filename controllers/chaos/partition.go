package chaos

import (
	"context"
	"fmt"
	"time"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/controllers/common/lifecycle"
	"github.com/fnikolai/frisbee/controllers/common/selector/service"
	"github.com/pkg/errors"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
)

type partition struct {
	r *Reconciler
}

func (f *partition) generate(ctx context.Context, obj *v1alpha1.Chaos) unstructured.Unstructured {
	affectedPods := service.Select(ctx, &obj.Spec.Partition.Selector)

	f.r.Logger.Info("Inject network partition", "targets", affectedPods.String())

	return unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "chaos-mesh.org/v1alpha1",
			"kind":       "NetworkChaos",
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

func (f *partition) Inject(ctx context.Context, obj *v1alpha1.Chaos) (ctrl.Result, error) {
	chaos := f.generate(ctx, obj)

	if err := common.SetOwner(obj, &chaos, fmt.Sprintf("%s.%d", obj.GetName(), time.Now().UnixNano())); err != nil {
		return lifecycle.Failed(ctx, obj, errors.Wrapf(err, "ownership error"))
	}

	// occasionally Chaos-Mesh throws an internal timeout. in this case, just retry the operation.
	err := retry.OnError(common.DefaultBackoff, k8errors.IsInternalError, func() error {
		return f.r.Create(ctx, &chaos)
	})
	if err != nil {
		return lifecycle.Failed(ctx, obj, errors.Wrapf(err, "injection failed"))
	}

	err = lifecycle.WatchObject(ctx,
		lifecycle.WatchExternal(&chaos, convertStatus, chaos.GetName()),
		lifecycle.WithFilter(lifecycle.FilterParent(obj.GetUID())),
	).Expect(v1alpha1.PhaseRunning)

	if err != nil {
		return lifecycle.Failed(ctx, obj, errors.Wrapf(err, "chaos error"))
	}

	AnnotateChaos(obj)

	f.r.Logger.Info("Chaos was successfully injected", "name", obj.GetName(), "faulttype", obj.Spec.Type)

	return lifecycle.Running(ctx, obj)
}
