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
)

type ServiceControlInterface interface {
	GetServiceSpec(ctx context.Context, namespace string, fromTemplate v1alpha1.GenerateFromTemplate) (v1alpha1.ServiceSpec, error)

	GetServiceSpecList(ctx context.Context, namespace string, fromTemplate v1alpha1.GenerateFromTemplate) ([]v1alpha1.ServiceSpec, error)

	// LoadSpecFromTemplate(ctx context.Context, obj *v1alpha1.Service) error

	Select(ctx context.Context, nm string, ss *v1alpha1.ServiceSelector) (v1alpha1.SList, error)
}

type ServiceControl struct {
	utils.Reconciler
}

func NewServiceControl(r utils.Reconciler) *ServiceControl {
	return &ServiceControl{
		Reconciler: r,
	}
}

func (s *ServiceControl) GetServiceSpec(ctx context.Context, namespace string, fromTemplate v1alpha1.GenerateFromTemplate) (v1alpha1.ServiceSpec, error) {
	template, err := s.getTemplate(ctx, namespace, fromTemplate.TemplateRef)
	if err != nil {
		return v1alpha1.ServiceSpec{}, errors.Wrapf(err, "getTemplate error")
	}

	// convert the service to a json and then expand templated values.
	serviceSpec, err := json.Marshal(template.Spec.Service)
	if err != nil {
		return v1alpha1.ServiceSpec{}, errors.Errorf("cannot marshal service of %s", fromTemplate.TemplateRef)
	}

	scheme := templateutils.Scheme{
		Inputs: template.Spec.Inputs,
		Spec:   string(serviceSpec),
	}

	lookupCache := make(map[string]v1alpha1.SList)

	return s.generateSpecFromScheme(ctx, namespace, &scheme, fromTemplate.GetInput(0), lookupCache)
}

func (s *ServiceControl) GetServiceSpecList(ctx context.Context, namespace string, fromTemplate v1alpha1.GenerateFromTemplate) ([]v1alpha1.ServiceSpec, error) {
	template, err := s.getTemplate(ctx, namespace, fromTemplate.TemplateRef)
	if err != nil {
		return nil, errors.Wrapf(err, "template %s error", fromTemplate.TemplateRef)
	}

	// convert the service to a json and then expand templated values.
	serviceSpec, err := json.Marshal(template.Spec.Service)
	if err != nil {
		return nil, errors.Errorf("cannot marshal service of %s", fromTemplate.TemplateRef)

	}

	// cache the results of macro as to avoid asking the Kubernetes API. This, however, is only applicable
	// to the level of a cluster, because different groups may be created in different moments
	// throughout the experiment,  thus yielding different results.
	lookupCache := make(map[string]v1alpha1.SList)

	specs := make([]v1alpha1.ServiceSpec, 0, fromTemplate.Instances)

	if err := fromTemplate.Iterate(func(userInputs map[string]string) error {
		scheme := templateutils.Scheme{
			Inputs: template.Spec.Inputs,
			Spec:   string(serviceSpec),
		}

		spec, err := s.generateSpecFromScheme(ctx, namespace, &scheme, userInputs, lookupCache)
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

func (s *ServiceControl) Select(ctx context.Context, nm string, ss *v1alpha1.ServiceSelector) (v1alpha1.SList, error) {
	if ss == nil {
		return nil, errors.New("empty service selector")
	}

	if ss.Macro != nil {
		parseMacro(nm, ss)
	}

	runningServices, err := selectServices(ctx, s.Reconciler, &ss.Match)
	if err != nil {
		return nil, errors.Wrapf(err, "service selection error")
	}

	// filter services based on the pods
	filteredServices, err := filterByMode(runningServices, ss.Mode, ss.Value)
	if err != nil {
		return nil, errors.Wrapf(err, "filter by mode")
	}

	return filteredServices, nil
}
