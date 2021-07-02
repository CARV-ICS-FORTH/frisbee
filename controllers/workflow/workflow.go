package workflow

import (
	"context"
	"time"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/controllers/common/selector/service"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *Reconciler) scheduleActions(topCtx context.Context, obj *v1alpha1.Workflow) {
	ctx, cancel := context.WithCancel(topCtx)
	defer cancel()

	r.Logger.Info("Workflow Started", "name", obj.GetName(), "time", time.Now())

	var err error

	for _, action := range obj.Spec.Actions {
		r.Logger.Info("Process Action", "type", action.ActionType, "name", action.Name, "depends", action.Depends)

		switch action.ActionType {
		case "Wait": // Wait command will block the entire controller
			err = r.wait(ctx, obj, *action.Wait)

		case "ServiceGroup":
			err = r.createServiceGroup(ctx, obj, action)

		case "Stop":
			err = r.stop(ctx, obj, action)

		default:
			err = errors.Errorf("unknown action %s", action.ActionType)
		}

		if err != nil {
			common.Failed(ctx, obj, errors.Wrapf(err, "action %s failed", action.Name))

			return
		}
	}

	common.Success(ctx, obj)
}

func (r *Reconciler) wait(ctx context.Context, w *v1alpha1.Workflow, spec v1alpha1.WaitSpec) error {
	if len(spec.Complete) > 0 {
		common.WaitLifecycle(ctx, w.GetUID(), &v1alpha1.ServiceGroup{}, v1alpha1.Complete, spec.Complete...)
	}

	if len(spec.Running) > 0 {
		common.WaitLifecycle(ctx, w.GetUID(), &v1alpha1.ServiceGroup{}, v1alpha1.Running, spec.Running...)
	}

	if spec.Duration != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(spec.Duration.Duration):
			return nil
		}
	}

	return nil
}

func (r *Reconciler) createServiceGroup(ctx context.Context, obj *v1alpha1.Workflow, action v1alpha1.Action) error {
	group := v1alpha1.ServiceGroup{}
	action.ServiceGroup.DeepCopyInto(&group.Spec)

	if err := common.SetOwner(obj, &group, action.Name); err != nil {
		return errors.Wrapf(err, "ownership failed")
	}

	if action.Depends != nil {
		if err := r.wait(ctx, obj, *action.Depends); err != nil {
			return errors.Wrapf(err, "dependencies failed")
		}
	}

	_, err := ctrl.CreateOrUpdate(ctx, r.Client, &group, func() error { return nil })
	if err != nil {
		return errors.Wrapf(err, "create failed")
	}

	// TODO: Fix it with respect to threads
	// common.UpdateLifecycle(ctx, obj, &v1alpha1.ServiceGroup{}, group.GetName())

	return nil
}

func (r *Reconciler) stop(ctx context.Context, obj *v1alpha1.Workflow, action v1alpha1.Action) error {
	if !service.IsMacro(action.Stop.Macro) {
		return errors.Errorf("invalid macro %s", action.Stop.Macro)
	}

	// Resolve affected services
	services := service.Select(ctx, service.ParseMacro(action.Stop.Macro))
	if len(services) == 0 {
		return errors.Errorf("no services to stop")
	}

	if action.Depends != nil {
		if err := r.wait(ctx, obj, *action.Depends); err != nil {
			return errors.Wrapf(err, "dependencies failed")
		}
	}

	// Without Schedule
	if action.Stop.Schedule == nil {
		for i := 0; i < len(services); i++ {
			// Change service Phase to Chaos so to ignore the failure caused by the following deletion.
			_, _ = common.Chaos(ctx, &services[i])

			if err := r.Client.Delete(ctx, &services[i]); err != nil {
				return errors.Wrapf(err, "cannot delete service %s", services[i].GetName())
			}
		}

		return nil
	}

	// With Schedule
	if action.Stop.Schedule != nil {
		r.Logger.Info("Yield with Schedule", "services", services)

		for service := range common.YieldByTime(ctx, action.Stop.Schedule.Cron, services...) {
			if err := r.Client.Delete(ctx, service); err != nil {
				return errors.Wrapf(err, "cannot delete service %s", service.GetName())
			}
		}
		return nil
	}

	return nil
}
