package workflow

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
)

type Workflow struct {
	*v1alpha1.Workflow

	waitableActions map[string]lifecycle.InnerObject
}

func (r *Reconciler) scheduleActions(topCtx context.Context, obj *v1alpha1.Workflow) {
	ctx, cancel := context.WithCancel(topCtx)
	defer cancel()

	// keep an index of names and objects. This is used for wait to identify the type of object to wait for.
	w := Workflow{
		Workflow:        obj,
		waitableActions: make(map[string]lifecycle.InnerObject),
	}

	var err error

	for _, action := range obj.Spec.Actions {
		r.Logger.Info("Exec Action", "type", action.ActionType, "name", action.Name, "depends", action.Depends)

		switch action.ActionType {
		case "Wait": // Expect command will block the entire controller
			err = r.wait(ctx, w, *action.Wait)

		case "ServiceGroup":
			err = r.createServiceGroup(ctx, w, action)

		case "Stop":
			err = r.stop(ctx, w, action)

		case "Chaos":
			err = r.chaos(ctx, w, action)

		default:
			err = errors.Errorf("unknown action %s", action.ActionType)
		}

		if err != nil {
			_, _ = lifecycle.Failed(ctx, w.Workflow, errors.Wrapf(err, "action %s failed", action.Name))

			return
		}
	}

	_, _ = lifecycle.Success(ctx, w.Workflow, "all actions are complete")
}

func (r *Reconciler) wait(ctx context.Context, w Workflow, spec v1alpha1.WaitSpec) error {
	if len(spec.Success) > 0 {
		logrus.Warn("-> Wait success for ", spec.Success)

		// confirm that the referenced action have already happened. otherwise, it is possible to block forever.
		for _, waitFor := range spec.Success {
			_, ok := w.waitableActions[waitFor]
			if !ok {
				return errors.Errorf("action %s has not happened yet", spec.Success[0])
			}
		}

		// assume that all action to wait are of the same type
		kind := w.waitableActions[spec.Success[0]]

		err := lifecycle.New(ctx,
			lifecycle.NewWatchdog(kind, spec.Success...),
			lifecycle.WithFilter(lifecycle.FilterParent(w.GetUID())),
			lifecycle.WithLogger(r.Logger),
		).Expect(v1alpha1.PhaseSuccess)
		if err != nil {
			return errors.Wrapf(err, "wait error")
		}

		logrus.Warn("<- Wait success for ", spec.Success)
	}

	if len(spec.Running) > 0 {
		logrus.Warn("-> Wait running for ", spec.Running)

		// confirm that the referenced action have already happened. otherwise, it is possible to block forever.
		for _, waitFor := range spec.Running {
			_, ok := w.waitableActions[waitFor]
			if !ok {
				return errors.Errorf("action %s has not happened yet", spec.Success[0])
			}
		}

		// assume that all action to wait are of the same type
		kind := w.waitableActions[spec.Running[0]]

		err := lifecycle.New(ctx,
			lifecycle.NewWatchdog(kind, spec.Running...),
			lifecycle.WithFilter(lifecycle.FilterParent(w.GetUID())),
			lifecycle.WithLogger(r.Logger),
		).Expect(v1alpha1.PhaseRunning)
		if err != nil {
			return errors.Wrapf(err, "wait error")
		}

		logrus.Warn("<- Wait running for ", spec.Running)
	}

	if spec.Duration != nil {
		logrus.Warn("-> Wait duration for ", spec.Duration.Duration.String())

		select {
		case <-ctx.Done():
			return errors.Wrapf(ctx.Err(), "wait error")
		case <-time.After(spec.Duration.Duration):
		}

		logrus.Warn("<- Wait duration for ", spec.Duration.Duration.String())
	}

	return nil
}

func (r *Reconciler) createServiceGroup(ctx context.Context, w Workflow, action v1alpha1.Action) error {
	group := v1alpha1.ServiceGroup{}
	action.ServiceGroup.DeepCopyInto(&group.Spec)

	if err := common.SetOwner(w.Workflow, &group, action.Name); err != nil {
		return errors.Wrapf(err, "ownership failed")
	}

	if action.Depends != nil {
		if err := r.wait(ctx, w, *action.Depends); err != nil {
			return errors.Wrapf(err, "dependencies failed")
		}
	}

	if err := r.Client.Create(ctx, &group); err != nil {
		return errors.Wrapf(err, "create failed")
	}

	// TODO: Fix it with respect to threads
	// common.Update(ctx, w, &v1alpha1.ServiceGroup{}, group.GetName())

	w.waitableActions[action.Name] = &group

	return nil
}

func (r *Reconciler) stop(ctx context.Context, w Workflow, action v1alpha1.Action) error {
	// Resolve affected services
	services := service.Select(ctx, action.Stop.Selector)
	if len(services) == 0 {
		r.Logger.Info("no services to stop", "action", action.Name)

		return nil
	}

	if action.Depends != nil {
		if err := r.wait(ctx, w, *action.Depends); err != nil {
			return errors.Wrapf(err, "dependencies failed")
		}
	}

	// Without Schedule
	if action.Stop.Schedule == nil {
		for i := 0; i < len(services); i++ {
			err := lifecycle.Delete(ctx, r.Client, &services[i])
			if err != nil && !k8errors.IsNotFound(err) {
				return errors.Wrapf(err, "cannot delete service %s", services[i].GetName())
			}
		}

		return nil
	}

	// With Schedule
	if action.Stop.Schedule != nil {
		r.Logger.Info("Yield with Schedule", "services", services)

		for service := range common.YieldByTime(ctx, action.Stop.Schedule.Cron, services...) {
			err := lifecycle.Delete(ctx, r.Client, service)
			if err != nil && !k8errors.IsNotFound(err) {
				return errors.Wrapf(err, "cannot delete service %s", service.GetName())
			}
		}

		return nil
	}

	return nil
}

func (r *Reconciler) chaos(ctx context.Context, w Workflow, action v1alpha1.Action) error {
	chaos := v1alpha1.Chaos{}
	action.Chaos.DeepCopyInto(&chaos.Spec)

	if err := common.SetOwner(w.Workflow, &chaos, action.Name); err != nil {
		return errors.Wrapf(err, "ownership failed")
	}

	if action.Depends != nil {
		if err := r.wait(ctx, w, *action.Depends); err != nil {
			return errors.Wrapf(err, "dependencies failed")
		}
	}

	if err := r.Client.Create(ctx, &chaos); err != nil {
		return errors.Wrapf(err, "chaos injection failed")
	}

	w.waitableActions[action.Name] = &chaos

	return nil
}
