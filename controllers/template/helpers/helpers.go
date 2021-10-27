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

package thelpers

import (
	"context"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/fnikolai/frisbee/api/v1alpha1"
	shelpers "github.com/fnikolai/frisbee/controllers/service/helpers"
	"github.com/fnikolai/frisbee/controllers/utils"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type GenericSpec string

func (s GenericSpec) ToServiceSpec() (v1alpha1.ServiceSpec, error) {
	// convert the payload with actual values into a spec
	spec := v1alpha1.ServiceSpec{}

	if err := yaml.Unmarshal([]byte(s), &spec); err != nil {
		return v1alpha1.ServiceSpec{}, errors.Wrapf(err, "decoding error")
	}

	return spec, nil
}

func (s GenericSpec) ToMonitorSpec() (v1alpha1.MonitorSpec, error) {
	// convert the payload with actual values into a spec
	spec := v1alpha1.MonitorSpec{}

	if err := yaml.Unmarshal([]byte(s), &spec); err != nil {
		return v1alpha1.MonitorSpec{}, errors.Wrapf(err, "decoding error")
	}

	return spec, nil
}

func GetDefaultSpec(ctx context.Context, r utils.Reconciler, ts *v1alpha1.TemplateSelector) (GenericSpec, error) {
	scheme, err := Select(ctx, r, ts)
	if err != nil {
		return "", errors.Wrapf(err, "scheme selection")
	}

	return GenerateSpecFromScheme(&scheme)
}

func GetParameterizedSpec(ctx context.Context, r utils.Reconciler, ts *v1alpha1.TemplateSelector,
	namespace string, inputs map[string]string, cache map[string]v1alpha1.SList,

) (GenericSpec, error) {
	scheme, err := Select(ctx, r, ts)
	if err != nil {
		return "", errors.Wrapf(err, "unable to create service")
	}

	if scheme.Inputs == nil {
		return "", errors.New("template is not parameterizable")
	}

	if err := ExpandInputs(ctx, r, namespace, &scheme, inputs, cache); err != nil {
		return "", errors.Wrapf(err, "input error")
	}

	specStr, err := GenerateSpecFromScheme(&scheme)
	if err != nil {
		return "", errors.Wrapf(err, "spec creation error. Inputs: %v", scheme.Inputs.Parameters)
	}

	return specStr, nil
}

var sprigFuncMap = sprig.TxtFuncMap() // a singleton for better performance

// GenerateSpecFromScheme parses a given scheme and returns the respective ServiceSpec.
func GenerateSpecFromScheme(tspec *v1alpha1.Scheme) (GenericSpec, error) {
	if tspec == nil {
		return "", errors.Errorf("empty scheme")
	}

	// replaced templated expression with actual values
	t := template.Must(
		template.New("").
			Funcs(sprigFuncMap).
			Option("missingkey=error").
			Parse(tspec.Spec))

	var out strings.Builder

	if err := t.Execute(&out, tspec); err != nil {
		return "", errors.Wrapf(err, "execution error")
	}

	return GenericSpec(out.String()), nil
}

func ExpandInputs(ctx context.Context,
	r utils.Reconciler,
	nm string,
	scheme *v1alpha1.Scheme,
	userInputs map[string]string,
	cache map[string]v1alpha1.SList) error {

	if scheme.Inputs == nil {
		return errors.New("scheme does not support inputs")
	}

	if scheme.Inputs.Parameters == nil {
		return errors.New("scheme does not support parameters")
	}

	for key := range scheme.Inputs.Parameters {
		// if there is no user-given value, use the default.
		value, ok := userInputs[key]
		if !ok {
			continue
		}

		// if the value is not a macro, write it directly to the inputs
		if !shelpers.IsMacro(value) {
			scheme.Inputs.Parameters[key] = value
		} else { // expand macro
			services, ok := cache[value]
			if !ok {
				services = shelpers.Select(ctx, r, nm, &v1alpha1.ServiceSelector{Macro: &value})

				if len(services) == 0 {
					return errors.Errorf("macro %s yields no services", value)
				}

				cache[value] = services
			}

			scheme.Inputs.Parameters[key] = services.ToString()
		}
	}

	return nil
}
