/*
Copyright 2021-2023 ICS-FORTH.

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
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetServiceSpec(ctx context.Context, cli client.Client, parent metav1.Object, fromTemplate v1alpha1.GenerateObjectFromTemplate) (v1alpha1.ServiceSpec, error) {
	specs, err := GetServiceSpecList(ctx, cli, parent, fromTemplate)
	if err != nil {
		return v1alpha1.ServiceSpec{}, errors.Wrapf(err, "failed to get service spec")
	}

	if len(specs) != 1 {
		panic(errors.Errorf("Expected 1 spec but got '%d'", len(specs)))
	}

	return specs[0], nil
}

func GetServiceSpecList(ctx context.Context, cli client.Client, parent metav1.Object, fromTemplate v1alpha1.GenerateObjectFromTemplate) ([]v1alpha1.ServiceSpec, error) {
	/*
		Get Service Templates
	*/
	var template v1alpha1.Template

	key := client.ObjectKey{
		Namespace: parent.GetNamespace(),
		Name:      fromTemplate.TemplateRef,
	}

	if err := cli.Get(ctx, key, &template); err != nil {
		return []v1alpha1.ServiceSpec{}, errors.Wrapf(err, "cannot find template '%s'", key.String())
	}

	/*
		Convert Service Template to JSON and expand inputs
	*/
	body, err := json.Marshal(template.Spec.Service)
	if err != nil {
		return nil, errors.Errorf("cannot marshal service of '%s'", fromTemplate.TemplateRef)
	}

	specs := make([]v1alpha1.ServiceSpec, 0, fromTemplate.MaxInstances)

	// add extra fields in the template
	if template.Spec.Inputs == nil {
		var inputs v1alpha1.TemplateInputs
		template.Spec.Inputs = &inputs
	}

	template.Spec.Inputs.Scenario = v1alpha1.GetScenarioLabel(parent)
	template.Spec.Inputs.Namespace = parent.GetNamespace()

	/*
		Generate Service Specs using the expanded inputs
	*/
	if err := fromTemplate.IterateInputs(func(nextInputSet uint) error {
		var spec v1alpha1.ServiceSpec

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
