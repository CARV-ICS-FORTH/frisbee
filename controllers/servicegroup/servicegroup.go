package servicegroup

import (
	"context"
	"fmt"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/controllers/common/selector"
	"github.com/fnikolai/frisbee/controllers/common/selector/service"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *Reconciler) create(ctx context.Context, obj *v1alpha1.ServiceGroup) (ctrl.Result, error) {
	serviceSpec, err := selector.SelectTemplate(ctx, obj.Spec.TemplateRef)
	if err != nil {
		return common.Failed(ctx, obj, err)
	}

	// validate service requests (resolve instances, inputs, and env variables)
	if (obj.Spec.Instances == 0 && len(obj.Spec.Inputs) == 0) ||
		(obj.Spec.Instances > 0 && len(obj.Spec.Inputs) > 0) {
		return common.Failed(ctx, obj, errors.Errorf("one of instances || inputs must be defined"))
	}

	if len(obj.Spec.Inputs) > 0 {
		obj.Spec.Instances = len(obj.Spec.Inputs)
	}

	// submit service creation requests
	serviceKeys := make([]string, obj.Spec.Instances)

	for i := 0; i < obj.Spec.Instances; i++ {
		var service v1alpha1.Service

		service.Name = fmt.Sprintf("%s-%d", obj.GetName(), i)
		service.Namespace = obj.GetNamespace()
		service.Spec = serviceSpec

		if err := common.SetOwner(obj, &service); err != nil {
			return common.Failed(ctx, obj, err)
		}

		// Give group env priority over class env
		service.Spec.Env = convertVars(ctx, obj.Spec.Env)
		if len(obj.Spec.Inputs) > 0 {
			service.Spec.Env = convertVars(ctx, obj.Spec.Inputs[i])
		}

		if _, err := ctrl.CreateOrUpdate(ctx, r.Client, &service, func() error { return nil }); err != nil {
			return common.Failed(ctx, obj, errors.Wrapf(err, "unable to update %s ", obj.GetName()))
		}

		serviceKeys[i] = service.GetName()
	}

	common.UpdateLifecycle(ctx, obj, &v1alpha1.Service{}, serviceKeys...)

	return common.DoNotRequeue()
}

func convertVars(ctx context.Context, in map[string]string) []v1.EnvVar {
	if len(in) == 0 {
		return nil
	}

	out := make([]v1.EnvVar, 0, len(in))

	for key, value := range in {
		out = append(out, v1.EnvVar{
			Name:  key,
			Value: service.Select(ctx, selector.ExpandMacro(value)).String(),
		})
	}

	return out
}
