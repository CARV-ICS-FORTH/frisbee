package collocatedgroup

import (
	"context"
	"fmt"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common/selector/service"
	"github.com/fnikolai/frisbee/controllers/common/selector/template"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
)

func (r *Reconciler) prepare(ctx context.Context, group *v1alpha1.CollocatedGroup) error {
	var serviceSpec *v1alpha1.ServiceSpec
	{ // sanitize parameters
		switch {
		case group.Spec.ServiceSpec != nil && group.Spec.TemplateRef != "":
			return errors.New("only one of servicespec and templateref can be used")

		case group.Spec.ServiceSpec != nil:
			serviceSpec = group.Spec.ServiceSpec

		case group.Spec.TemplateRef != "":
			serviceSpec = template.SelectService(ctx, template.ParseRef(group.GetNamespace(), group.Spec.TemplateRef))

			if serviceSpec == nil {
				return errors.Errorf("template %s/%s was not found", group.GetNamespace(), group.Spec.TemplateRef)
			}
		}

		// all inputs are explicitly defined. no instances were given
		if group.Spec.Instances == 0 {
			if len(group.Spec.Inputs) == 0 {
				return errors.New("at least one of instances || inputs must be defined")
			}

			group.Spec.Instances = len(group.Spec.Inputs)
		}

		// instances were given, with one explicit input
		if len(group.Spec.Inputs) == 1 {
			inputs := make([]map[string]string, group.Spec.Instances)

			for i := 0; i < group.Spec.Instances; i++ {
				inputs[i] = group.Spec.Inputs[0]
			}

			group.Spec.Inputs = inputs
		}
	}

	// cache the results of macro as to avoid asking the Kubernetes API. This, however, is only applicable
	// to the level of a group, because different groups may be created in different momements throughout the experiment,
	// thus yielding different results.
	lookupCache := make(map[string]v1alpha1.ServiceSpecList)

	var serviceList v1alpha1.ServiceSpecList

	for i := 0; i < group.Spec.Instances; i++ {
		// prevent different services from sharing the same spec
		spec := serviceSpec.DeepCopy()

		spec.NamespacedName = generateName(group, i)

		// prepare service environment
		if len(group.Spec.Inputs) > 0 {
			if err := r.inputs2Env(ctx, group.Spec.Inputs[i], &spec.Container, lookupCache); err != nil {
				return errors.Wrapf(err, "macro expansion failed")
			}
		}

		serviceList = append(serviceList, spec)
	}

	group.Status.ExpectedServices = serviceList

	return nil
}

// if there is only one instance, it will be named after the group. otherwise, the instances will be named
// as Master-0, Master-1, ...
func generateName(group *v1alpha1.CollocatedGroup, i int) v1alpha1.NamespacedName {
	if group.Spec.Instances == 1 {
		return v1alpha1.NamespacedName{Namespace: group.GetNamespace(), Name: group.GetName()}
	}

	return v1alpha1.NamespacedName{Namespace: group.GetNamespace(), Name: fmt.Sprintf("%s-%d", group.GetName(), i)}
}

func (r *Reconciler) inputs2Env(ctx context.Context, inputs map[string]string, container *v1.Container,
	cache map[string]v1alpha1.ServiceSpecList) error {
	if len(inputs) != len(container.Env) {
		return errors.Errorf("mismatch inputs and env vars. vars:%d inputs:%d ", len(container.Env), len(inputs))
	}

	for i, evar := range container.Env {
		value, ok := inputs[evar.Name]
		if !ok {
			return errors.Errorf("%s parameter not set", evar.Name)
		}

		if service.IsMacro(value) {
			services, ok := cache[value]
			if !ok {
				services = service.Select(ctx, &v1alpha1.ServiceSelector{Macro: &value})

				if len(services) == 0 {
					return errors.Errorf("macro %s yields no services", value)
				}

				cache[value] = services
			}

			container.Env[i].Value = services.String()
		} else {
			container.Env[i].Value = value
		}
	}

	return nil
}
