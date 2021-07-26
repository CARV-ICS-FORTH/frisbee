package service

import (
	"context"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/controllers/common/lifecycle"
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
		err := lifecycle.New(ctx,
			lifecycle.NewWatchdog(port, port.GetName()),
			lifecycle.WithLogger(r.Logger),
		).Expect(v1alpha1.PhaseRunning)
		if err != nil {
			return lifecycle.Failed(ctx, obj, errors.Wrapf(err, "waiting for port %s failed", port.GetName()))
		}
	}

	annotations := portStatusToAnnotations(port.GetName(), port.Spec.Protocol, port.Status.Direct)
	// abusively use the labels to evaluate annotations
	if structure.Contains(obj.GetAnnotations(), annotations) {
		return lifecycle.Pending(ctx, obj, "wait for dataport to become ready")
	}

	obj.SetAnnotations(labels.Merge(obj.GetAnnotations(), annotations))

	// update is needed because the mesh operations may change labels and annotations
	return common.Update(ctx, obj)
}
