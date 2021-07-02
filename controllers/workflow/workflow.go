package workflow

import (
	"context"
	"time"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/controllers/common/selector"
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
		common.WaitLifecycle(ctx, w.GetUID(), &v1alpha1.ServiceGroup{}, spec.Complete, v1alpha1.Complete)
	}

	if len(spec.Running) > 0 {
		common.WaitLifecycle(ctx, w.GetUID(), &v1alpha1.ServiceGroup{}, spec.Running, v1alpha1.Running)
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
		return errors.Wrapf(err, "ownership failed %s", group.GetName())
	}

	if action.Depends != nil {
		if err := r.wait(ctx, obj, *action.Depends); err != nil {
			return errors.Wrapf(err, "dependencies failed. action %s ", action.Name)
		}
	}

	_, err := ctrl.CreateOrUpdate(ctx, r.Client, &group, func() error { return nil })
	if err != nil {
		return errors.Wrapf(err, "unable to create group %s", group.GetName())
	}

	// TODO: fix it with respect to other threads
	// common.UpdateLifecycle(ctx, obj, &v1alpha1.ServiceGroup{}, group.GetName())

	return nil
}

func (r *Reconciler) stop(ctx context.Context, obj *v1alpha1.Workflow, action v1alpha1.Action) error {
	// Resolve affected services
	var services []v1alpha1.Service
	if action.Stop.Macro != "" {
		services = service.Select(ctx, selector.ParseMacro(action.Stop.Macro))
	}

	if len(services) == 0 {
		return errors.Errorf("no service to stop. macro %s", action.Stop.Macro)
	}

	if action.Depends != nil {
		if err := r.wait(ctx, obj, *action.Depends); err != nil {
			return errors.Wrapf(err, "dependencies failed. action %s ", action.Name)
		}
	}

	// Without Schedule
	if action.Stop.Schedule == nil {
		for i := 0; i < len(services); i++ {
			// Change service Phase to Chaos so to ignore the failure caused by the following deletion.
			_, _ = common.Chaos(ctx, &services[i])

			if err := r.Client.Delete(ctx, &services[i]); err != nil {
				return errors.Wrapf(err, "unable to delete service %s", services[i].GetName())
			}
		}

		return nil
	}

	// With Schedule
	if action.Stop.Schedule != nil {
		r.Logger.Info("Yield with Schedule", "services", services)

		for service := range common.YieldByTime(ctx, action.Stop.Schedule.Cron, services...) {
			if err := r.Client.Delete(ctx, service); err != nil {
				return errors.Wrapf(err, "unable to delete service %s", service.GetName())
			}
		}

		r.Logger.Info("Scheduled operations done ")

		return nil
	}

	return nil
}
