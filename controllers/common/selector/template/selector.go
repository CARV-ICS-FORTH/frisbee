package template

import (
	"context"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var Client client.Client

func Select(ctx context.Context, ts *v1alpha1.TemplateSelector) (v1alpha1.ServiceSpec, error) {
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
	case len(ts.Selector.Service) > 0:
		serviceSpec, ok := template.Spec.Services[ts.Selector.Service]
		if !ok {
			return v1alpha1.ServiceSpec{}, errors.Errorf("unable to find service %s", ts.Selector.Service)
		}

		return serviceSpec, nil

	default:
		return v1alpha1.ServiceSpec{}, errors.Errorf("unspecified selection criteratia")
	}
}
