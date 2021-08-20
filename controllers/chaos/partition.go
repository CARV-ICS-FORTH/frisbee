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
		"targets", affectedPods.String(),
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

	if err := common.SetOwner(obj, &chaos); err != nil {
		return errors.Wrapf(err, "ownership error")
	}

	// occasionally Chaos-Mesh throws an internal timeout. in this case, just retry the operation.
	err := retry.OnError(common.DefaultBackoff, k8errors.IsInternalError, func() error {
		err := f.r.Create(ctx, &chaos)

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
		lifecycle.WithUpdateParent(obj),
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
	// (managed by Chaos-Mesh) it suffice to remove the internal object, and the external will be garbage collected.
	if err := lifecycle.Delete(ctx, f.r, obj); err != nil {
		return errors.Wrapf(err, "unable to revoke %s", obj.GetName())
	}

	return nil
}
