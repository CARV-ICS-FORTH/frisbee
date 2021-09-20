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

package helpers

import (
	"context"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func GetServiceSpec(ctx context.Context, ts *v1alpha1.TemplateSelector) (*v1alpha1.ServiceSpec, error) {
	scheme := SelectServiceTemplate(ctx, ts)

	return GenerateServiceSpec(scheme)
}

// GenerateServiceSpec parses a given scheme and returns the respective ServiceSpec.
func GenerateServiceSpec(tspec *v1alpha1.Scheme) (*v1alpha1.ServiceSpec, error) {
	// replaced templated expression with actual values
	t := template.Must(
		template.New("").
			Funcs(sprig.TxtFuncMap()).
			Option("missingkey=error").
			Parse(tspec.Spec))

	var out strings.Builder

	if err := t.Execute(&out, tspec); err != nil {
		return nil, errors.Wrapf(err, "execution error")
	}

	// convert the payload with actual values into a spec
	spec := v1alpha1.ServiceSpec{}

	if err := yaml.Unmarshal([]byte(out.String()), &spec); err != nil {
		return nil, errors.Wrapf(err, "service decode")
	}

	return &spec, nil
}

func GetMonitorSpec(ctx context.Context, ts *v1alpha1.TemplateSelector) (*v1alpha1.MonitorSpec, error) {
	scheme := SelectMonitorTemplate(ctx, ts)

	return GenerateMonitorSpec(scheme)
}

// GenerateMonitorSpec parses a given scheme and returns the respective MonitorSpec.
func GenerateMonitorSpec(tspec *v1alpha1.Scheme) (*v1alpha1.MonitorSpec, error) {
	// replaced templated expression with actual values
	t := template.Must(
		template.New("").
			Funcs(sprig.TxtFuncMap()).
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
