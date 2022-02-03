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
	"github.com/carv-ics-forth/frisbee/controllers/utils"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ChaosControlInterface interface {
	GetChaosSpec(ctx context.Context, namespace string, fromTemplate v1alpha1.GenerateFromTemplate) (v1alpha1.ChaosSpec, error)

	GetChaosSpecList(ctx context.Context, namespace string, fromTemplate v1alpha1.GenerateFromTemplate) ([]v1alpha1.ChaosSpec, error)
}

type ChaosControl struct {
	utils.Reconciler
}

func NewChaosControl(r utils.Reconciler) *ChaosControl {
	return &ChaosControl{
		Reconciler: r,
	}
}

func (s *ChaosControl) GetChaosSpec(ctx context.Context, namespace string, fromTemplate v1alpha1.GenerateFromTemplate) (v1alpha1.ChaosSpec, error) {
	template, err := s.getTemplate(ctx, namespace, fromTemplate.TemplateRef)
	if err != nil {
		return v1alpha1.ChaosSpec{}, errors.Wrapf(err, "getTemplate error")
	}

	// convert the chaos to a json and then expand templated values.
	chaosSpec, err := json.Marshal(template.Spec.Chaos)
	if err != nil {
		return v1alpha1.ChaosSpec{}, errors.Errorf("cannot marshal chaos of %s", fromTemplate.TemplateRef)
	}

	scheme := templateutils.Scheme{
		Inputs: template.Spec.Inputs,
		Spec:   string(chaosSpec),
	}

	return s.generateSpecFromScheme(&scheme, fromTemplate.GetInput(0))
}

func (s *ChaosControl) GetChaosSpecList(ctx context.Context, namespace string, fromTemplate v1alpha1.GenerateFromTemplate) ([]v1alpha1.ChaosSpec, error) {
	template, err := s.getTemplate(ctx, namespace, fromTemplate.TemplateRef)
	if err != nil {
		return nil, errors.Wrapf(err, "template %s error", fromTemplate.TemplateRef)
	}

	// convert the chaos to a json and then expand templated values.
	chaosSpec, err := json.Marshal(template.Spec.Chaos)
	if err != nil {
		return nil, errors.Errorf("cannot marshal chaos of %s", fromTemplate.TemplateRef)
	}

	specs := make([]v1alpha1.ChaosSpec, 0, fromTemplate.MaxInstances)

	if err := fromTemplate.IterateInputs(func(userInputs map[string]string) error {
		scheme := templateutils.Scheme{
			Inputs: template.Spec.Inputs,
			Spec:   string(chaosSpec),
		}

		spec, err := s.generateSpecFromScheme(&scheme, userInputs)
		if err != nil {
			return errors.Wrapf(err, "macro expansion failed")
		}

		specs = append(specs, spec)

		return nil
	}); err != nil {
		return nil, errors.Wrapf(err, "cannot get specs")
	}

	return specs, nil
}

func (s *ChaosControl) getTemplate(ctx context.Context, namespace string, ref string) (*v1alpha1.Template, error) {
	var template v1alpha1.Template

	key := client.ObjectKey{
		Namespace: namespace,
		Name:      ref,
	}

	if err := s.GetClient().Get(ctx, key, &template); err != nil {
		return nil, errors.Wrapf(err, "cannot find template [%s]", key.String())
	}

	return &template, nil
}

func (s *ChaosControl) generateSpecFromScheme(scheme *templateutils.Scheme, userInputs map[string]string) (v1alpha1.ChaosSpec, error) {
	if userInputs != nil {
		if scheme.Inputs == nil || scheme.Inputs.Parameters == nil {
			return v1alpha1.ChaosSpec{}, errors.New("template is not parameterizable")
		}

		for key, value := range userInputs {
			_, exists := scheme.Inputs.Parameters[key]
			if !exists {
				return v1alpha1.ChaosSpec{}, errors.Errorf("parameter [%s] does not exist", key)
			}

			scheme.Inputs.Parameters[key] = value
		}
	}

	genericSpec, err := templateutils.Evaluate(scheme)
	if err != nil {
		return v1alpha1.ChaosSpec{}, errors.Wrapf(err, "cannot convert scheme to spec")
	}

	var spec v1alpha1.ChaosSpec

	if err := yaml.Unmarshal([]byte(genericSpec), &spec); err != nil {
		return v1alpha1.ChaosSpec{}, errors.Wrapf(err, "decoding error")
	}

	return spec, nil
}
