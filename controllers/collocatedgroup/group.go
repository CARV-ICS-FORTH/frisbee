package collocatedgroup

import (
	"context"
	"fmt"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common/selector/service"
	"github.com/fnikolai/frisbee/controllers/template/helpers"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (r *Reconciler) prepare(ctx context.Context, group *v1alpha1.CollocatedGroup) error {
	switch {
	case group.Spec.ServiceSpec != nil && group.Spec.TemplateRef != "":
		return errors.New("only one of servicespec and templateref can be used")

	case group.Spec.ServiceSpec != nil:
		if len(group.Spec.Inputs) > 0 {
			return errors.New("no inputs  are allowed when service spec is defined")
		}

		for i := 0; i < group.Spec.Instances; i++ {
			// if the service is specifically define, we can create only one instance
			spec := group.Spec.ServiceSpec.DeepCopy()

			spec.NamespacedName = generateName(group, i)

			group.Status.ExpectedServices = append(group.Status.ExpectedServices, spec)
		}

		return nil

	case group.Spec.TemplateRef != "":
		// all inputs are explicitly defined. no instances were given
		if group.Spec.Instances == 0 {
			if len(group.Spec.Inputs) == 0 {
				return errors.New("at least one of instances || inputs must be defined")
			}

			group.Spec.Instances = len(group.Spec.Inputs)
		}

		// cache the results of macro as to avoid asking the Kubernetes API. This, however, is only applicable
		// to the level of a group, because different groups may be created in different moments
		// throughout the experiment,  thus yielding different results.
		lookupCache := make(map[string]v1alpha1.ServiceSpecList)

		scheme := helpers.SelectServiceTemplate(ctx, helpers.ParseRef(group.GetNamespace(), group.Spec.TemplateRef))

		for i := 0; i < group.Spec.Instances; i++ {
			switch len(group.Spec.Inputs) {
			case 0:
				// no inputs
			case 1:
				// use a common set of inputs for all instances
				if err := inputs2Env(ctx, scheme.Inputs.Parameters, group.Spec.Inputs[0], lookupCache); err != nil {
					return errors.Wrapf(err, "macro expansion failed")
				}

			default:
				// use a different set of inputs for every instance
				if err := inputs2Env(ctx, scheme.Inputs.Parameters, group.Spec.Inputs[i], lookupCache); err != nil {
					return errors.Wrapf(err, "macro expansion failed")
				}
			}

			logrus.Warn("Generate scheme", scheme.Inputs)

			service, err := helpers.GenerateServiceSpec(scheme)
			if err != nil {
				return errors.Wrapf(err, "scheme to service")
			}

			service.NamespacedName = generateName(group, i)

			group.Status.ExpectedServices = append(group.Status.ExpectedServices, service)
		}

		return nil

	default:
		return errors.Errorf("at least one of Service or TemplateRef must be defined")
	}
}

// if there is only one instance, it will be named after the group. otherwise, the instances will be named
// as Master-0, Master-1, ...
func generateName(group *v1alpha1.CollocatedGroup, i int) v1alpha1.NamespacedName {
	if group.Spec.Instances == 1 {
		return v1alpha1.NamespacedName{Namespace: group.GetNamespace(), Name: group.GetName()}
	}

	return v1alpha1.NamespacedName{Namespace: group.GetNamespace(), Name: fmt.Sprintf("%s-%d", group.GetName(), i)}
}

func inputs2Env(ctx context.Context, dst, src map[string]string, cache map[string]v1alpha1.ServiceSpecList) error {
	for key := range dst {
		// if there is no user-given value, use the default.
		value, ok := src[key]
		if !ok {
			continue
		}

		// if the value is not a macro, write it directly to the inputs
		if !service.IsMacro(value) {
			dst[key] = value
		} else { // expand macro
			services, ok := cache[value]
			if !ok {
				services = service.Select(ctx, &v1alpha1.ServiceSelector{Macro: &value})

				if len(services) == 0 {
					return errors.Errorf("macro %s yields no services", value)
				}

				cache[value] = services
			}

			dst[key] = services.String()
		}
	}

	return nil
}
