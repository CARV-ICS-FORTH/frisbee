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

func GetServiceSpec(ctx context.Context, c client.Client, parent metav1.Object, fromTemplate v1alpha1.GenerateFromTemplate) (v1alpha1.ServiceSpec, error) {
	template, err := templateutils.GetTemplate(ctx, c, parent, fromTemplate.TemplateRef)
	if err != nil {
		return v1alpha1.ServiceSpec{}, errors.Wrapf(err, "template [%s] is not installed", fromTemplate.TemplateRef)
	}

	// convert the service to a json and then expand templated values.
	specBody, err := json.Marshal(template.Spec.Service)
	if err != nil {
		return v1alpha1.ServiceSpec{}, errors.Errorf("cannot marshal service of %s", fromTemplate.TemplateRef)
	}

	scheme, err := templateutils.NewScheme(parent, template.Spec.Inputs, specBody)
	if err != nil {
		return v1alpha1.ServiceSpec{}, errors.Wrapf(err, "cannot get scheme for '%s'", fromTemplate.TemplateRef)
	}

	var spec v1alpha1.ServiceSpec

	if err := templateutils.GenerateFromScheme(&spec, scheme, fromTemplate.GetInput(0)); err != nil {
		return v1alpha1.ServiceSpec{}, errors.Wrapf(err, "cannot create spec")
	}

	return spec, nil
}

func GetServiceSpecList(ctx context.Context, c client.Client, parent metav1.Object, fromTemplate v1alpha1.GenerateFromTemplate) ([]v1alpha1.ServiceSpec, error) {
	template, err := templateutils.GetTemplate(ctx, c, parent, fromTemplate.TemplateRef)
	if err != nil {
		return nil, errors.Wrapf(err, "template [%s] is not installed", fromTemplate.TemplateRef)
	}

	// convert the service to a json and then expand templated values.
	specBody, err := json.Marshal(template.Spec.Service)
	if err != nil {
		return nil, errors.Errorf("cannot marshal service of %s", fromTemplate.TemplateRef)

	}

	specs := make([]v1alpha1.ServiceSpec, 0, fromTemplate.MaxInstances)

	if err := fromTemplate.IterateInputs(func(userInputs map[string]string) error {
		scheme, err := templateutils.NewScheme(parent, template.Spec.Inputs, specBody)
		if err != nil {
			return errors.Wrapf(err, "cannot get scheme for '%s'", fromTemplate.TemplateRef)
		}

		var spec v1alpha1.ServiceSpec

		if err := templateutils.GenerateFromScheme(&spec, scheme, userInputs); err != nil {
			return errors.Wrapf(err, "macro expansion failed")
		}

		specs = append(specs, spec)

		return nil
	}); err != nil {
		return nil, errors.Wrapf(err, "cannot get specs")
	}

	return specs, nil
}
