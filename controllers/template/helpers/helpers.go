// Licensed to FORTH/ICS under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. FORTH/ICS licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

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

var sprigFuncMap = sprig.TxtFuncMap() // a singleton for better performance

func GetServiceSpec(ctx context.Context, r utils.Reconciler, ts *v1alpha1.TemplateSelector) (v1alpha1.ServiceSpec, error) {
	scheme := SelectServiceTemplate(ctx, r, ts)

	return GenerateSpecFromScheme(scheme)
}

// GenerateSpecFromScheme parses a given scheme and returns the respective ServiceSpec.
func GenerateSpecFromScheme(tspec *v1alpha1.Scheme) (v1alpha1.ServiceSpec, error) {
	if tspec == nil {
		return v1alpha1.ServiceSpec{}, errors.Errorf("empty service spec")
	}

	// replaced templated expression with actual values
	t := template.Must(
		template.New("").
			Funcs(sprigFuncMap).
			Option("missingkey=error").
			Parse(tspec.Spec))

	var out strings.Builder

	if err := t.Execute(&out, tspec); err != nil {
		return v1alpha1.ServiceSpec{}, errors.Wrapf(err, "execution error")
	}

	// convert the payload with actual values into a spec
	spec := v1alpha1.ServiceSpec{}

	if err := yaml.Unmarshal([]byte(out.String()), &spec); err != nil {
		return v1alpha1.ServiceSpec{}, errors.Wrapf(err, "service decode")
	}

	return spec, nil
}

func GetMonitorSpec(ctx context.Context, r utils.Reconciler, ts *v1alpha1.TemplateSelector) (*v1alpha1.MonitorSpec, error) {
	scheme := SelectMonitorTemplate(ctx, r, ts)

	return GenerateMonitorSpec(scheme)
}

// GenerateMonitorSpec parses a given scheme and returns the respective MonitorSpec.
func GenerateMonitorSpec(tspec *v1alpha1.Scheme) (*v1alpha1.MonitorSpec, error) {
	// replaced templated expression with actual values
	t := template.Must(
		template.New("").
			Funcs(sprigFuncMap).
			Option("missingkey=error").
			Parse(tspec.Spec))

	var out strings.Builder

	if err := t.Execute(&out, tspec); err != nil {
		return nil, errors.Wrapf(err, "execution error")
	}

	// convert the payload with actual values into a spec
	spec := v1alpha1.MonitorSpec{}

	if err := yaml.Unmarshal([]byte(out.String()), &spec); err != nil {
		return nil, errors.Wrapf(err, "monitor decode")
	}

	return &spec, nil
}

func ExpandInputs(ctx context.Context,
	r utils.Reconciler,
	nm string,
	dst,
	src map[string]string,
	cache map[string]v1alpha1.SList) error {
	for key := range dst {
		// if there is no user-given value, use the default.
		value, ok := src[key]
		if !ok {
			continue
		}

		// if the value is not a macro, write it directly to the inputs
		if !shelpers.IsMacro(value) {
			dst[key] = value
		} else { // expand macro
			services, ok := cache[value]
			if !ok {
				services = shelpers.Select(ctx, r, nm, &v1alpha1.ServiceSelector{Macro: &value})

				if len(services) == 0 {
					return errors.Errorf("macro %s yields no services", value)
				}

				cache[value] = services
			}

			dst[key] = services.ToString()
		}
	}

	return nil
}
