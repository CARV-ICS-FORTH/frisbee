package service

import (
	"context"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/pkg/structure"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *Reconciler) direct(ctx context.Context, obj *v1alpha1.Service, port *v1alpha1.DataPort) (ctrl.Result, error) {
	logrus.Warn("-> Wait for lifecycle of ", port.GetName())
	logrus.Warn("<- Wait for lifecycle of ", port.GetName())

	// if the port is not ready, just wait for it.
	if port.Status.Lifecycle.Phase != v1alpha1.PhaseRunning {
		err := common.GetLifecycle(ctx,
			common.Watch(port, port.GetName()),
		).Expect(v1alpha1.PhaseRunning)
		if err != nil {
			return common.Failed(ctx, obj, errors.Wrapf(err, "waiting for port %s failed", port.GetName()))
		}
	}

	annotations := portStatusToAnnotations(port.GetName(), port.Spec.Protocol, port.Status.Direct)
	// abusively use the labels to evaluate annotations
	if structure.Contains(obj.GetAnnotations(), annotations) {
		return common.Pending(ctx, obj)
	}

	obj.SetAnnotations(labels.Merge(obj.GetAnnotations(), annotations))

	// update is needed because the mesh operations may change labels and annotations
	return common.Update(ctx, obj)
}
