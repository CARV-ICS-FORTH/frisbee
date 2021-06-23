package workflow

import (
	"context"
	"time"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *Reconciler) run(ctx context.Context, obj *v1alpha1.Workflow) (ctrl.Result, error) {
	r.Logger.Info("Workflow Started", "name", obj.GetName(), "time", time.Now())

	var err error

	for _, action := range obj.Spec.Actions {
		if action.Depends != nil {
			r.Logger.Info("Dependency", "cond", action.Depends)

			if err := r.wait(ctx, obj, action.Depends); err != nil {
				return common.Failed(ctx, obj, errors.Wrapf(err, "unable to wait"))
			}
		}

		r.Logger.Info("Process Action", "type", action.ActionType, "name", action.Name)

		// FIXME: should it run within a goroutine ?

		switch action.ActionType {
		case "Create":
			err = r.createService(ctx, obj, &action)
		case "CreateGroup":
			err = r.createServiceGroup(ctx, obj, &action)
		case "Wait":
			err = r.wait(ctx, obj, action.Wait)
		default:
			return common.Failed(ctx, obj, errors.New("unknown action"))
		}

		if err != nil {
			return common.Failed(ctx, obj, err)
		}
	}

	r.Logger.Info("Workflow Completed", "name", obj.GetName(), "time", time.Now())

	logrus.Warn("-- DONE --")

	return common.DoNotRequeue()
}

func (r *Reconciler) createService(ctx context.Context, w *v1alpha1.Workflow, action *v1alpha1.Action) error {
	service := v1alpha1.Service{}
	service.Namespace = w.GetNamespace()
	service.Name = action.Name
	action.CreateService.DeepCopyInto(&service.Spec)

	if err := common.SetOwner(w, &service); err != nil {
		return errors.Wrap(err, "unable to set owner")
	}

	_, err := ctrl.CreateOrUpdate(ctx, r.Client, &service, func() error { return nil })

	return errors.Wrapf(err, "cannot create service %s", service.GetName())
}

func (r *Reconciler) createServiceGroup(ctx context.Context, w *v1alpha1.Workflow, action *v1alpha1.Action) error {
	group := v1alpha1.ServiceGroup{}
	group.Namespace = w.GetNamespace()
	group.Name = action.Name
	action.CreateServiceGroup.DeepCopyInto(&group.Spec)

	if err := common.SetOwner(w, &group); err != nil {
		return errors.Wrap(err, "unable to set owner")
	}

	_, err := ctrl.CreateOrUpdate(ctx, r.Client, &group, func() error { return nil })

	return errors.Wrapf(err, "cannot create servicegroup %s", group.GetName())
}

func (r *Reconciler) wait(ctx context.Context, w *v1alpha1.Workflow, spec *v1alpha1.WaitSpec) error {

	if len(spec.Ready) > 0 {
		// wait for groups
		return common.WaitForPhase(ctx, w.GetUID(), &v1alpha1.ServiceGroup{}, spec.Ready, v1alpha1.Running)
	}

	if len(spec.Complete) > 0 {
		// wait for groups
		return common.WaitForPhase(ctx, w.GetUID(), &v1alpha1.ServiceGroup{}, spec.Complete, v1alpha1.Succeed)
	}

	return nil
}
