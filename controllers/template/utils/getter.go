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
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/carv-ics-forth/frisbee/controllers/common/configuration"
	"github.com/pkg/errors"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetTemplate searches Frisbee for the given reference. By default, it searches the given namespace. If it is not found,
// then it searches the installation namespace.
func GetTemplate(ctx context.Context, r common.Reconciler, testplanNamespace string, ref string) (*v1alpha1.Template, error) {
	var template v1alpha1.Template

	key := client.ObjectKey{
		Namespace: testplanNamespace,
		Name:      ref,
	}

	err := r.GetClient().Get(ctx, key, &template)
	switch {
	case k8errors.IsNotFound(err):
		if testplanNamespace == configuration.Global.Namespace {
			return nil, errors.Wrapf(err, "cannot find template [%s]", key.String())
		}

		// If it not found on the default namespace, try with the installation namespace
		key.Namespace = configuration.Global.Namespace

		if err := r.GetClient().Get(ctx, key, &template); err != nil {
			return nil, errors.Wrapf(err, "failed to discover '%s' at both test '%s' and installation '%s' namespaces",
				ref, testplanNamespace, configuration.Global.Namespace)
		}
	case err != nil:
		return nil, errors.Wrapf(err, "cannot retrieve template [%s]", key.String())
	}

	return &template, nil
}

func GenerateFromScheme(spec interface{}, scheme *Scheme, userInputs map[string]string) error {
	if userInputs != nil {
		if scheme.Inputs == nil || scheme.Inputs.Parameters == nil {
			return errors.New("template is not parameterizable")
		}

		for key, value := range userInputs {
			_, exists := scheme.Inputs.Parameters[key]
			if !exists {
				return errors.Errorf("parameter [%s] does not exist", key)
			}

			scheme.Inputs.Parameters[key] = value
		}
	}

	expandedSpecBody, err := Evaluate(scheme)
	if err != nil {
		return errors.Wrapf(err, "cannot convert scheme to spec")
	}

	if err := yaml.Unmarshal([]byte(expandedSpecBody), spec); err != nil {
		return errors.Wrapf(err, "decoding error")
	}

	return nil
}
