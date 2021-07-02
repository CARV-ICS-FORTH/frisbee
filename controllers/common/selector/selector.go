package selector

import (
	"strings"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
)

// ParseMacro translates a given macro to the appropriate selector and executes it.
// If the input is not a macro, it will be returned immediately as the first element of a string slice.
func ParseMacro(macro string) *v1alpha1.ServiceSelector {
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
