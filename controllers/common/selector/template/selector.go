package template

import (
	"context"
	"strings"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var Client client.Client

func IsRef(templateRef string) bool {
	parsed := strings.Split(templateRef, "/")
	return len(parsed) == 2
}

// ParseRef parse the templateRef and returns a template selector. If the templateRef is invalid, the selector
// will be nil, and any subsequence select operation will return empty value.
func ParseRef(nm, templateRef string) *v1alpha1.TemplateSelector {
	parsed := strings.Split(templateRef, "/")
	if len(parsed) != 2 {
		panic("invalid reference format")
	}

	family := parsed[0]
	ref := parsed[1]

	return &v1alpha1.TemplateSelector{
		Namespace: nm,
		Family:    family,
		Selector: v1alpha1.TemplateSelectorSpec{
			Reference: ref,
		},
	}
}

func SelectService(ctx context.Context, ts *v1alpha1.TemplateSelector) *v1alpha1.ServiceSpec {
	if ts == nil {
		return nil
	}

	var template v1alpha1.Template

	key := client.ObjectKey{
		Namespace: ts.Namespace,
		Name:      ts.Family,
	}

	// if the template is created in parallel with the workflow, it is possible to meet race conditions.
	// We avoid it with a simple retry mechanism based on adaptive backoff.
	err := retry.OnError(retry.DefaultRetry, k8errors.IsNotFound, func() error {
		return Client.Get(ctx, key, &template)
	})
	if err != nil {
		logrus.Warn(err)

		return nil
	}

	// TODO: Change Get to List

	switch {
	case len(ts.Selector.Reference) > 0:
		serviceSpec, ok := template.Spec.Services[ts.Selector.Reference]
		if !ok {
			logrus.Warn(errors.Errorf("unable to find entry %s", ts.Selector.Reference))

			return nil
		}

		return &serviceSpec

	default:
		panic(errors.Errorf("unspecified selection criteria"))
	}
}

func SelectMonitor(ctx context.Context, ts *v1alpha1.TemplateSelector) *v1alpha1.MonitorSpec {
	if ts == nil {
		return nil
	}

	var template v1alpha1.Template

	key := client.ObjectKey{
		Namespace: ts.Namespace,
		Name:      ts.Family,
	}

	// if the template is created in parallel with the workflow, it is possible to meet race conditions.
	// We avoid it with a simple retry mechanism based on adaptive backoff.
	err := retry.OnError(retry.DefaultRetry, k8errors.IsNotFound, func() error {
		return Client.Get(ctx, key, &template)
	})
	if err != nil {
		logrus.Warn(err)

		return nil
	}

	// TODO: Change Get to List

	switch {
	case len(ts.Selector.Reference) > 0:
		monSpec, ok := template.Spec.Monitors[ts.Selector.Reference]
		if !ok {
			logrus.Warn(errors.Errorf("unable to find entry %s", ts.Selector.Reference))
			return nil
		}

		return &monSpec

	default:
		panic(errors.Errorf("unspecified selection criteria"))
	}
}
