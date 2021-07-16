package service

import (
	"context"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *Reconciler) direct(ctx context.Context, obj *v1alpha1.Service, port *v1alpha1.DataPort) (ctrl.Result, error) {
	// if the port is running, extract the necessary information and trigger start the pod (move obj to Pending).
	if port.Status.Lifecycle.Phase == v1alpha1.PhaseRunning {
		annotations := portStatusToAnnotations(port.GetName(), port.Spec.Protocol, port.Status.Direct)

		// abusively use the labels to evaluate annotations
		if labels.Equals(annotations, obj.GetAnnotations()) {
			return common.Pending(ctx, obj)
		}

		// update is needed because the mesh operations may change labels and annotations
		obj.SetAnnotations(annotations)
		return common.Update(ctx, obj)
	}

	logrus.Warn("Wait for lifecycle of ", port.GetName())

	// if the port is not ready, just wait for it.
	if err := common.GetLifecycle(ctx, "", port, port.GetName()).Expect(v1alpha1.PhaseRunning); err != nil {
		return common.Failed(ctx, obj, errors.Wrapf(err, "waiting for port %s failed", port.GetName()))
	}

	// cause reconcile to get the new port information.
	return common.Requeue()
}

