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

func GetChaosSpec(ctx context.Context, c client.Client, parent metav1.Object, fromTemplate v1alpha1.GenerateObjectFromTemplate) (v1alpha1.ChaosSpec, error) {
	template, err := templateutils.GetTemplate(ctx, c, parent, fromTemplate.TemplateRef)
	if err != nil {
		return v1alpha1.ChaosSpec{}, errors.Wrapf(err, "getTemplate error")
	}

	// convert the chaos to a json and then expand templated values.
	body, err := json.Marshal(template.Spec.Chaos)
	if err != nil {
		return v1alpha1.ChaosSpec{}, errors.Errorf("cannot marshal chaos of %s", fromTemplate.TemplateRef)
	}

	var spec v1alpha1.ChaosSpec

	// set extra runtime fields.
	if template.Spec.Inputs == nil {
		var inputs v1alpha1.TemplateInputs
		template.Spec.Inputs = &inputs
	}

	// add extra fields in the template
	template.Spec.Inputs.Scenario = v1alpha1.GetScenarioLabel(parent)
	template.Spec.Inputs.Namespace = parent.GetNamespace()

	if err := fromTemplate.Generate(&spec, 0, template.Spec, body); err != nil {
		return v1alpha1.ChaosSpec{}, errors.Wrapf(err, "evaluation of template '%s' has failed", fromTemplate.TemplateRef)
	}

	return spec, nil
}

func GetChaosSpecList(ctx context.Context, c client.Client, parent metav1.Object, fromTemplate v1alpha1.GenerateObjectFromTemplate) ([]v1alpha1.ChaosSpec, error) {
	template, err := templateutils.GetTemplate(ctx, c, parent, fromTemplate.TemplateRef)
	if err != nil {
		return nil, errors.Wrapf(err, "template %s error", fromTemplate.TemplateRef)
	}

	// convert the chaos to a json and then expand templated values.
	body, err := json.Marshal(template.Spec.Chaos)
	if err != nil {
		return nil, errors.Errorf("cannot marshal chaos of %s", fromTemplate.TemplateRef)
	}

	specs := make([]v1alpha1.ChaosSpec, 0, fromTemplate.MaxInstances)

	// add extra fields in the template
	template.Spec.Inputs.Scenario = v1alpha1.GetScenarioLabel(parent)
	template.Spec.Inputs.Namespace = parent.GetNamespace()

	if err := fromTemplate.IterateInputs(func(nextInputSet uint) error {
		var spec v1alpha1.ChaosSpec

		if err := fromTemplate.Generate(&spec, nextInputSet, template.Spec, body); err != nil {
			return errors.Wrapf(err, "evaluation of template '%s' has failed", fromTemplate.TemplateRef)
		}

		specs = append(specs, spec)

		return nil
	}); err != nil {
		return nil, errors.Wrapf(err, "cannot get specs")
	}

	return specs, nil
}
