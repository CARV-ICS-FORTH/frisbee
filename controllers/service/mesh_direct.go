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
	logrus.Warnf("-> Bind service %s with port %s", obj.GetName(), port.GetName())
	defer logrus.Warnf("<- Bind service %s with port %s", obj.GetName(), port.GetName())

	// wait for port to become ready
	err := lifecycle.New(ctx,
		lifecycle.Watch(port, port.GetName()),
		lifecycle.WithLogger(r.Logger),
	).Until(v1alpha1.PhaseRunning, port)
	if err != nil {
		return lifecycle.Failed(ctx, obj, errors.Wrapf(err, "waiting for port %s failed", port.GetName()))
	}

	// convert status of the remote port to local annotations that will be used by the ENV.
	annotations := portStatusToAnnotations(port.GetName(), port.Spec.Protocol, port.GetProtocolStatus())
	if len(annotations) == 0 {
		panic("empty annotations")
	}

	if structure.Contains(obj.GetAnnotations(), annotations) {
		return lifecycle.Pending(ctx, obj, "wait for dataport to become ready")
	}

	obj.SetAnnotations(labels.Merge(obj.GetAnnotations(), annotations))

	// update is needed because the mesh operations may change labels and annotations
	return common.Update(ctx, obj)
}
