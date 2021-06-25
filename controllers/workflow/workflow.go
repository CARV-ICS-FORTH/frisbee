package workflow

import (
	"context"
	"time"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *Reconciler) schedule(ctx context.Context, w *v1alpha1.Workflow) {
	r.Logger.Info("Workflow Started", "name", w.GetName(), "time", time.Now())

	for _, action := range w.Spec.Actions {
		r.Logger.Info("Process Action", "type", action.ActionType, "name", action.Name, "depends", action.Depends)

		switch action.ActionType {
		case "Create":
			go r.createService(ctx, w, action)
		case "CreateGroup":
			go r.createServiceGroup(ctx, w, action)
		case "Wait": // Wait command will block the entire controller
			if err := r.wait(ctx, w, action.Wait); err != nil {
				common.Failed(ctx, w, err)
				return
			}
		default:
			common.Failed(ctx, w, errors.Errorf("unknown action %s", action.ActionType))

			return
		}
	}

	common.Success(ctx, w)
}

func (r *Reconciler) createService(ctx context.Context, w *v1alpha1.Workflow, action v1alpha1.Action) {
	if action.Depends != nil {
		if err := r.wait(ctx, w, action.Depends); err != nil {
			common.Failed(ctx, w, errors.Wrap(err, "unable to wait"))

			return
		}
	}

	service := v1alpha1.Service{}
	service.Namespace = w.GetNamespace()
	service.Name = action.Name
	action.CreateService.DeepCopyInto(&service.Spec)

	if err := common.SetOwner(w, &service); err != nil {
		common.Failed(ctx, w, errors.Wrap(err, "unable to set owner"))
		return
	}

	_, err := ctrl.CreateOrUpdate(ctx, r.Client, &service, func() error { return nil })
	if err != nil {
		common.Failed(ctx, w, errors.Wrapf(err, "cannot create service %s", service.GetName()))
	}
}

func (r *Reconciler) createServiceGroup(ctx context.Context, w *v1alpha1.Workflow, action v1alpha1.Action) {
	if action.Depends != nil {
		if err := r.wait(ctx, w, action.Depends); err != nil {
			common.Failed(ctx, w, errors.Wrapf(err, "unable to wait"))
			return
		}
	}

	group := v1alpha1.ServiceGroup{}
	group.Namespace = w.GetNamespace()
	group.Name = action.Name
	action.CreateServiceGroup.DeepCopyInto(&group.Spec)

	if err := common.SetOwner(w, &group); err != nil {
		common.Failed(ctx, w, errors.Wrap(err, "unable to set owner"))
		return
	}

	_, err := ctrl.CreateOrUpdate(ctx, r.Client, &group, func() error { return nil })
	if err != nil {
		common.Failed(ctx, w, errors.Wrapf(err, "cannot create servicegroup %s", group.GetName()))
		return
	}
}

func (r *Reconciler) wait(ctx context.Context, w *v1alpha1.Workflow, spec *v1alpha1.WaitSpec) error {
	if len(spec.Complete) > 0 {
		return common.WaitLifecycle(ctx, w.GetUID(), &v1alpha1.ServiceGroup{}, spec.Complete, v1alpha1.Complete)
	}

	if len(spec.Running) > 0 {
		return common.WaitLifecycle(ctx, w.GetUID(), &v1alpha1.ServiceGroup{}, spec.Running, v1alpha1.Running)
	}

	return nil
}
