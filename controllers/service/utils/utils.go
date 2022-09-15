/*
Copyright 2021 ICS-FORTH.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"context"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	templateutils "github.com/carv-ics-forth/frisbee/controllers/template/utils"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetServiceSpec(ctx context.Context, c client.Client, parent metav1.Object, fromTemplate v1alpha1.GenerateObjectFromTemplate) (v1alpha1.ServiceSpec, error) {
	// get the template
	t, err := templateutils.GetTemplate(ctx, c, parent, fromTemplate.TemplateRef)
	if err != nil {
		return v1alpha1.ServiceSpec{}, errors.Wrapf(err, "template '%s' is not installed", fromTemplate.TemplateRef)
	}

	// convert the service to a json and then expand templated values.
	body, err := json.Marshal(t.Spec.Service)
	if err != nil {
		return v1alpha1.ServiceSpec{}, errors.Errorf("cannot marshal service of '%s'", fromTemplate.TemplateRef)
	}

	spec := v1alpha1.ServiceSpec{}

	// set extra runtime fields.
	if t.Spec.Inputs == nil {
		t.Spec.Inputs = &v1alpha1.TemplateInputs{}
	}

	// add extra fields in the template
	t.Spec.Inputs.Scenario = v1alpha1.GetScenarioLabel(parent)
	t.Spec.Inputs.Namespace = parent.GetNamespace()

	if err := fromTemplate.Generate(&spec, 0, t.Spec, body); err != nil {
		return v1alpha1.ServiceSpec{}, errors.Wrapf(err, "evaluation of template '%s' has failed", fromTemplate.TemplateRef)
	}

	return spec, nil
}

func GetServiceSpecList(ctx context.Context, c client.Client, parent metav1.Object, fromTemplate v1alpha1.GenerateObjectFromTemplate) ([]v1alpha1.ServiceSpec, error) {
	t, err := templateutils.GetTemplate(ctx, c, parent, fromTemplate.TemplateRef)
	if err != nil {
		return nil, errors.Wrapf(err, "template '%s' is not installed", fromTemplate.TemplateRef)
	}

	// convert the service to a json and then expand templated values.
	body, err := json.Marshal(t.Spec.Service)
	if err != nil {
		return nil, errors.Errorf("cannot marshal service of '%s'", fromTemplate.TemplateRef)
	}

	specs := make([]v1alpha1.ServiceSpec, 0, fromTemplate.MaxInstances)

	// add extra fields in the template
	t.Spec.Inputs.Scenario = v1alpha1.GetScenarioLabel(parent)
	t.Spec.Inputs.Namespace = parent.GetNamespace()

	if err := fromTemplate.IterateInputs(func(nextInputSet uint) error {
		spec := v1alpha1.ServiceSpec{}

		if err := fromTemplate.Generate(&spec, nextInputSet, t.Spec, body); err != nil {
			return errors.Wrapf(err, "evaluation of template '%s' has failed", fromTemplate.TemplateRef)
		}

		specs = append(specs, spec)

		return nil
	}); err != nil {
		return nil, errors.Wrapf(err, "cannot get specs")
	}

	return specs, nil
}
