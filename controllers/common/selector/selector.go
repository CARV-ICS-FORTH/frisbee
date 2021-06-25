package selector

import (
	"context"
	"strings"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common/selector/service"
	"github.com/pkg/errors"
)

// ExpandMacroToSelector translates a given macro to the appropriate selector and executes it.
// If the input is not a macro, it will be returned immediately as the first element of a string slice.
func ExpandMacroToSelector(ctx context.Context, macro string) []string {
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
