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
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ServiceControlInterface interface {
	GetServiceSpec(ctx context.Context, namespace string, fromTemplate v1alpha1.FromTemplate) (v1alpha1.ServiceSpec, error)

	GetServiceSpecList(ctx context.Context, namespace string, fromTemplate v1alpha1.FromTemplate) ([]v1alpha1.ServiceSpec, error)

	GetMonitorSpec(ctx context.Context, namespace string, fromTemplate v1alpha1.FromTemplate) (v1alpha1.MonitorSpec, error)

	LoadSpecFromTemplate(ctx context.Context, obj *v1alpha1.Service) error

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

func (s *ServiceControl) GetServiceSpecList(ctx context.Context, namespace string, fromTemplate v1alpha1.FromTemplate) ([]v1alpha1.ServiceSpec, error) {
	templateRef := fromTemplate.GetTemplateRef()

	var template v1alpha1.Template

	key := client.ObjectKey{
		Namespace: namespace,
		Name:      templateRef.Template,
	}

	if err := s.GetClient().Get(ctx, key, &template); err != nil {
		return nil, errors.Wrapf(err, "cannot find template")
	}

	scheme, ok := template.Spec.Entries[templateRef.Ref]
	if !ok {
		return nil, errors.Errorf("invalid template ref %s", templateRef.String())
	}

	// cache the results of macro as to avoid asking the Kubernetes API. This, however, is only applicable
	// to the level of a cluster, because different groups may be created in different moments
	// throughout the experiment,  thus yielding different results.
	lookupCache := make(map[string]v1alpha1.SList)

	specs := make([]v1alpha1.ServiceSpec, 0, fromTemplate.Instances)

	if err := fromTemplate.Iterate(func(in map[string]string) error {
		spec, err := s.generateSpecFromScheme(ctx, namespace, &scheme, in, lookupCache)
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

func (s *ServiceControl) GetServiceSpec(ctx context.Context, namespace string, fromTemplate v1alpha1.FromTemplate) (v1alpha1.ServiceSpec, error) {
	templateRef := fromTemplate.GetTemplateRef()

	var template v1alpha1.Template

	key := client.ObjectKey{
		Namespace: namespace,
		Name:      templateRef.Template,
	}

	if err := s.GetClient().Get(ctx, key, &template); err != nil {
		return v1alpha1.ServiceSpec{}, errors.Wrapf(err, "cannot find template")
	}

	scheme, ok := template.Spec.Entries[templateRef.Ref]
	if !ok {
		return v1alpha1.ServiceSpec{}, errors.Errorf("invalid template ref %s", templateRef.String())
	}

	lookupCache := make(map[string]v1alpha1.SList)

	return s.generateSpecFromScheme(ctx, namespace, &scheme, fromTemplate.GetInput(0), lookupCache)
}

func (s *ServiceControl) LoadSpecFromTemplate(ctx context.Context, obj *v1alpha1.Service) error {
	if err := obj.Spec.FromTemplate.Validate(false); err != nil {
		return errors.Wrapf(err, "template validation")
	}

	spec, err := s.GetServiceSpec(ctx, obj.GetNamespace(), *obj.Spec.FromTemplate)
	if err != nil {
		return errors.Wrapf(err, "cannot get spec")
	}

	spec.DeepCopyInto(&obj.Spec)

	return nil
}

func (s *ServiceControl) GetMonitorSpec(ctx context.Context, namespace string, fromTemplate v1alpha1.FromTemplate) (v1alpha1.MonitorSpec, error) {
	templateRef := fromTemplate.GetTemplateRef()

	var template v1alpha1.Template

	key := client.ObjectKey{
		Namespace: namespace,
		Name:      templateRef.Template,
	}

	if err := s.GetClient().Get(ctx, key, &template); err != nil {
		return v1alpha1.MonitorSpec{}, errors.Wrapf(err, "cannot find template")
	}

	scheme, ok := template.Spec.Entries[templateRef.Ref]
	if !ok {
		return v1alpha1.MonitorSpec{}, errors.Errorf("invalid template ref %s", templateRef.String())
	}

	genericSpec, err := templateutils.Evaluate(&scheme)
	if err != nil {
		return v1alpha1.MonitorSpec{}, errors.Wrapf(err, "cannot convert scheme to spec")
	}

	var spec v1alpha1.MonitorSpec

	if err := yaml.Unmarshal([]byte(genericSpec), &spec); err != nil {
		return v1alpha1.MonitorSpec{}, errors.Wrapf(err, "decoding error")
	}

	return spec, nil

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
