package servicegroup

import (
	"context"
	"fmt"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/controllers/common/lifecycle"
	"github.com/fnikolai/frisbee/controllers/common/selector/service"
	"github.com/fnikolai/frisbee/controllers/common/selector/template"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *Reconciler) create(ctx context.Context, obj *v1alpha1.ServiceGroup) error {
	serviceSpec := template.SelectService(ctx, template.ParseRef(obj.GetNamespace(), obj.Spec.TemplateRef))
	if serviceSpec == nil {
		return errors.Errorf("template %s/%s was not found", obj.GetNamespace(), obj.Spec.TemplateRef)
	}

	// all inputs are explicitly defined. no instances were given
	if obj.Spec.Instances == 0 {
		if len(obj.Spec.Inputs) == 0 {
			return errors.New("at least one of instances || inputs must be defined")
		}

		obj.Spec.Instances = len(obj.Spec.Inputs)
	}

	// instances were given, with one explicit input
	if len(obj.Spec.Inputs) == 1 {
		inputs := make([]map[string]string, obj.Spec.Instances)

		for i := 0; i < obj.Spec.Instances; i++ {
			inputs[i] = obj.Spec.Inputs[0]
		}

		obj.Spec.Inputs = inputs
	}

	serviceKeys := make([]string, obj.Spec.Instances)

	// create services for this group
	for i := 0; i < obj.Spec.Instances; i++ {
		service := v1alpha1.Service{}

		if err := common.SetOwner(obj, &service, fmt.Sprintf("%s-%d", obj.GetName(), i)); err != nil {
			return errors.Wrapf(err, "setowner failed")
		}

		serviceSpec.DeepCopyInto(&service.Spec) // deep copy so to avoid different services from sharing the same spec

		if len(obj.Spec.Inputs) > 0 {
			if err := r.inputs2Env(ctx, obj.Spec.Inputs[i], &service.Spec.Container); err != nil {
				return errors.Wrapf(err, "macro expansion failed")
			}
		}

		if _, err := ctrl.CreateOrUpdate(ctx, r.Client, &service, func() error { return nil }); err != nil {
			return errors.Wrapf(err, "update failed")
		}

		serviceKeys[i] = service.GetName()
	}

	err := lifecycle.WatchObject(ctx,
		lifecycle.Watch(&v1alpha1.Service{}, serviceKeys...),
		lifecycle.WithFilter(lifecycle.FilterParent(obj.GetUID())),
		lifecycle.WithAnnotator(true), // Register event to grafana
		lifecycle.WithLogger(r.Logger),
	).UpdateParentLifecycle(obj)

	return errors.Wrapf(err, "lifecycle failed")
}

func (r *Reconciler) inputs2Env(ctx context.Context, inputs map[string]string, container *v1.Container) error {
	if len(inputs) != len(container.Env) {
		return errors.Errorf("mismatch inputs and env vars. inputs:%d vars:%d", len(inputs), len(container.Env))
	}

	for i, evar := range container.Env {
		value, ok := inputs[evar.Name]
		if !ok {
			return errors.Errorf("%s parameter not set", evar.Name)
		}

		if service.IsMacro(value) {
			services := service.Select(ctx, &v1alpha1.ServiceSelector{Macro: &value})

			if len(services) == 0 {
				return errors.Errorf("macro %s yields no services", value)
			}

			container.Env[i].Value = services.String()
		} else {
			container.Env[i].Value = value
		}
	}

	return nil
}
