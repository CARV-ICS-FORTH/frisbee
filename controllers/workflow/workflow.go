package workflow

import (
	"context"
	"time"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/controllers/common/selector/service"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (r *Reconciler) scheduleActions(topCtx context.Context, obj *v1alpha1.Workflow) {
	ctx, cancel := context.WithCancel(topCtx)
	defer cancel()

	// r.Logger.Info("Workflow Started", "name", obj.GetName(), "time", time.Now())

	var err error

	for _, action := range obj.Spec.Actions {
		r.Logger.Info("Exec Action", "type", action.ActionType, "name", action.Name, "depends", action.Depends)

		switch action.ActionType {
		case "Wait": // Expect command will block the entire controller
			err = r.wait(ctx, obj, *action.Wait)

		case "ServiceGroup":
			err = r.createServiceGroup(ctx, obj, action)

		case "Stop":
			err = r.stop(ctx, obj, action)

		case "Chaos":
			err = r.chaos(ctx, obj, action)

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
	if len(spec.Success) > 0 {
		logrus.Warn("-> Wait success for ", spec.Success)

		err := common.GetLifecycle(ctx,
			common.Watch(&v1alpha1.ServiceGroup{}, spec.Success...),
			common.WithFilter(common.FilterParent(w.GetUID())),
		).Expect(v1alpha1.PhaseSuccess)
		if err != nil {
			return errors.Wrapf(err, "wait error")
		}

		logrus.Warn("<- Wait success for ", spec.Success)
	}

	if len(spec.Running) > 0 {
		logrus.Warn("-> Wait running for ", spec.Running)

		err := common.GetLifecycle(ctx,
			common.Watch(&v1alpha1.ServiceGroup{}, spec.Running...),
			common.WithFilter(common.FilterParent(w.GetUID())),
		).Expect(v1alpha1.PhaseRunning)
		if err != nil {
			return errors.Wrapf(err, "wait error")
		}

		logrus.Warn("<- Wait running for ", spec.Running)
	}

	if spec.Duration != nil {
		logrus.Warn("-> Wait duration for ", spec.Duration)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(spec.Duration.Duration):
		}

		logrus.Warn("<- Wait duration for ", spec.Duration)
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

	if err := r.Client.Create(ctx, &group); err != nil {
		return errors.Wrapf(err, "create failed")
	}

	// TODO: Fix it with respect to threads
	// common.UpdateLifecycle(ctx, obj, &v1alpha1.ServiceGroup{}, group.GetName())

	return nil
}

func (r *Reconciler) stop(ctx context.Context, obj *v1alpha1.Workflow, action v1alpha1.Action) error {
	if action.Depends != nil {
		if err := r.wait(ctx, obj, *action.Depends); err != nil {
			return errors.Wrapf(err, "dependencies failed")
		}
	}

	// Resolve affected services
	services := service.Select(ctx, action.Stop.Selector)
	if len(services) == 0 {
		r.Logger.Info("no services to stop", "action", action.Name)

		return nil
	}

	// Without Schedule
	if action.Stop.Schedule == nil {
		for i := 0; i < len(services); i++ {
			// Change service Phase to PhaseChaos so to ignore the failure caused by the following deletion.
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

func (r *Reconciler) chaos(ctx context.Context, obj *v1alpha1.Workflow, action v1alpha1.Action) error {
	chaos := v1alpha1.Chaos{}
	action.Chaos.DeepCopyInto(&chaos.Spec)

	if err := common.SetOwner(obj, &chaos, action.Name); err != nil {
		return errors.Wrapf(err, "ownership failed")
	}

	if action.Depends != nil {
		if err := r.wait(ctx, obj, *action.Depends); err != nil {
			return errors.Wrapf(err, "dependencies failed")
		}
	}

	if err := r.Client.Create(ctx, &chaos); err != nil {
		return errors.Wrapf(err, "chaos injection failed")
	}

	return nil
}
