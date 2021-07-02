package template

import (
	"context"
	"strings"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var Client client.Client

// ParseRef parse the templateRef and returns a template selector. If the templateRef is invalid, the selector
// will be nil, and any subsequence select operation will return empty value.
func ParseRef(templateRef string) *v1alpha1.TemplateSelector {
	parsed := strings.Split(templateRef, "/")
	if len(parsed) != 2 {
		logrus.Warn("invalid reference format")
		return nil
	}

	family := parsed[0]
	ref := parsed[1]

	return &v1alpha1.TemplateSelector{
		Family: family,
		Selector: v1alpha1.TemplateSelectorSpec{
			Reference: ref,
		},
	}
}

func SelectService(ctx context.Context, ts *v1alpha1.TemplateSelector) (v1alpha1.ServiceSpec, error) {
	if ts == nil {
		return v1alpha1.ServiceSpec{}, nil
	}

	var template v1alpha1.Template

	key := client.ObjectKey{Namespace: "frisbee", Name: ts.Family}
	if err := Client.Get(ctx, key, &template); err != nil {
		return v1alpha1.ServiceSpec{}, err
	}

	// TODO: Change Get to List

	switch {
	case len(ts.Selector.Reference) > 0:
		serviceSpec, ok := template.Spec.Services[ts.Selector.Reference]
		if !ok {
			return v1alpha1.ServiceSpec{}, errors.Errorf("unable to find entry %s", ts.Selector.Reference)
		}

		return serviceSpec, nil

	default:
		return v1alpha1.ServiceSpec{}, errors.Errorf("unspecified selection criteria")
	}
}

func SelectMonitor(ctx context.Context, ts *v1alpha1.TemplateSelector) (*v1alpha1.MonitorSpec, error) {
	if ts == nil {
		return nil, nil
	}

	var template v1alpha1.Template

	key := client.ObjectKey{Namespace: "frisbee", Name: ts.Family}
	if err := Client.Get(ctx, key, &template); err != nil {
		return nil, err
	}

	// TODO: Change Get to List

	switch {
	case len(ts.Selector.Reference) > 0:
		monSpec, ok := template.Spec.Monitors[ts.Selector.Reference]
		if !ok {
			return nil, errors.Errorf("unable to find entry %s", ts.Selector.Reference)
		}

		return &monSpec, nil

	default:
		return nil, errors.Errorf("unspecified selection criteria")
	}
}
