package selector

import (
	"context"
	"strings"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common/selector/template"
	"github.com/pkg/errors"
)

/*
// ExpandMacroToSelector translates a given macro to the appropriate selector and executes it.
// If the input is not a macro, it will be returned immediately as the first element of a string slice.
func ExpandMacroToSelector(ctx context.Context, macro string) ServiceIDList {
	if !strings.HasPrefix(macro, ".") {
		return []string{macro}
	}

	fields := strings.Split(macro, ".")

	if len(fields) != 4 {
		panic(errors.Errorf("%s is not a valid macro", macro))
	}

	kind := fields[1]
	object := fields[2]
	filter := fields[3]

	_ = object

	switch kind {
	case "servicegroup":
		criteria := &v1alpha1.ServiceSelector{}
		criteria.Selector.ServiceGroup = object
		criteria.Mode = v1alpha1.Mode(filter)

		services, err := service.Select(ctx, criteria)
		if err != nil {
			panic(err)
		}

		ids := make([]string, len(services))
		for i, service := range services {
			ids[i] = service.GetName()
		}

		return ids

	default:
		panic(errors.Errorf("%s is not a valid macro", macro))
	}
}

*/

// SelectTemplate loads the specification of a templated service
func SelectTemplate(ctx context.Context, templateRef string) (v1alpha1.ServiceSpec, error) {
	parsed := strings.Split(templateRef, "/")
	if len(parsed) != 2 {
		return v1alpha1.ServiceSpec{}, errors.Errorf("invalid template format")
	}

	templateFamily := parsed[0]
	serviceRef := parsed[1]

	spec, err := template.Select(ctx, &v1alpha1.TemplateSelector{
		Family: templateFamily,
		Selector: v1alpha1.TemplateSelectorSpec{
			Service: serviceRef,
		},
	})

	return spec, err
}

func ExpandMacro(macro string) *v1alpha1.ServiceSelector {
	var criteria v1alpha1.ServiceSelector

	if !strings.HasPrefix(macro, ".") {
		return nil
	}

	fields := strings.Split(macro, ".")

	if len(fields) != 4 {
		panic(errors.Errorf("%s is not a valid macro", macro))
	}

	kind := fields[1]
	object := fields[2]
	filter := fields[3]

	switch kind {
	case "servicegroup":
		criteria.Selector.ServiceGroup = object
		criteria.Mode = v1alpha1.Mode(filter)

		return &criteria

	default:
		panic(errors.Errorf("%s is not a valid macro", macro))
	}
}
