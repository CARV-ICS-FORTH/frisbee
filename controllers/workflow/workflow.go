package workflow

import (
	"context"
	"time"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/controllers/common/selector"
	"github.com/fnikolai/frisbee/controllers/common/selector/service"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *Reconciler) schedule(ctx context.Context, w *v1alpha1.Workflow) (ctrl.Result, error) {
	r.Logger.Info("Workflow Started", "name", w.GetName(), "time", time.Now())

	if action := w.Spec.Actions[len(w.Spec.Actions)-1]; action.ActionType != "Wait" {
		return common.Failed(ctx, w, errors.New("All experiments must end with a wait function"))
	}

	for _, action := range w.Spec.Actions {
		r.Logger.Info("Process Action", "type", action.ActionType, "name", action.Name, "depends", action.Depends)

		switch action.ActionType {
		case "Wait": // Wait command will block the entire controller
			if err := r.wait(ctx, w, action.Wait); err != nil {
				return common.Failed(ctx, w, errors.Wrapf(err, "wait %s failed", action.Name))
			}

		case "ServiceGroup":
			go r.createServiceGroup(ctx, w, action)

		case "Stop":
			go r.stop(ctx, w, action)

		default:
			return common.Failed(ctx, w, errors.Errorf("unknown action %s", action.ActionType))
		}
	}

	return common.Success(ctx, w)
}

func (r *Reconciler) wait(ctx context.Context, w *v1alpha1.Workflow, spec *v1alpha1.WaitSpec) error {
	if len(spec.Complete) > 0 {
		return common.WaitLifecycle(ctx, w.GetUID(), &v1alpha1.ServiceGroup{}, spec.Complete, v1alpha1.Complete)
	}

	if len(spec.Running) > 0 {
		return common.WaitLifecycle(ctx, w.GetUID(), &v1alpha1.ServiceGroup{}, spec.Running, v1alpha1.Running)
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

func (r *Reconciler) createServiceGroup(ctx context.Context, w *v1alpha1.Workflow, action v1alpha1.Action) {
	if action.Depends != nil {
		if err := r.wait(ctx, w, action.Depends); err != nil {
			runtime.HandleError(err)

			return
		}
	}

	group := v1alpha1.ServiceGroup{}
	group.Namespace = w.GetNamespace()
	group.Name = action.Name
	action.ServiceGroup.DeepCopyInto(&group.Spec)

	if err := common.SetOwner(w, &group); err != nil {
		runtime.HandleError(errors.Wrapf(err, "unable to set owner forgroup %s", group.GetName()))

		return
	}

	_, err := ctrl.CreateOrUpdate(ctx, r.Client, &group, func() error { return nil })
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "unable to create group %s", group.GetName()))

		return
	}
}

func (r *Reconciler) stop(ctx context.Context, w *v1alpha1.Workflow, action v1alpha1.Action) {
	if action.Depends != nil {
		if err := r.wait(ctx, w, action.Depends); err != nil {
			runtime.HandleError(err)

			return
		}
	}

	var services []v1alpha1.Service

	if action.Stop.Macro != "" {
		services = service.Select(ctx, selector.ExpandMacro(action.Stop.Macro))
	}

	for i := 0; i < len(services); i++ {
		// Change service Phase to Chaos so to ignore the failure caused by the following deletion.
		_, _ = common.Chaos(ctx, &services[i])

		if err := r.Client.Delete(ctx, &services[i]); err != nil {
			runtime.HandleError(errors.Wrapf(err, "unable to delete service %s", services[i].GetName()))

			return
		}
	}
}
