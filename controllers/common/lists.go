package common

import (
	"strings"

	"github.com/fnikolai/frisbee/api/v1alpha1"
)

const (
	idListSeparator = " "
)

type ServiceList []v1alpha1.Service

func (list ServiceList) String() string {
	return strings.Join(list.Names(), idListSeparator)
}

func (list ServiceList) Names() []string {
	names := make([]string, len(list))

	for i, service := range list {
		names[i] = service.GetName()
	}

	return names
}

// ByNamespace return the services by the namespace they belong to.
func (list ServiceList) ByNamespace() map[string][]string {
	all := make(map[string][]string)

	for _, s := range list {
		// get namespace
		sublist := all[s.GetNamespace()]

		// append service to the namespace
		sublist = append(sublist, s.GetName())

		// update namespace
		all[s.GetNamespace()] = sublist
	}

	return all
}
