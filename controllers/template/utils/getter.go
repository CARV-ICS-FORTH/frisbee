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
	"github.com/pkg/errors"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetTemplate searches Frisbee for the given reference. By default, it searches the given namespace. If it is not found,
// then it searches the installation namespace.
func GetTemplate(ctx context.Context, c client.Client, who metav1.Object, ref string) (*v1alpha1.Template, error) {
	var template v1alpha1.Template

	key := client.ObjectKey{
		Namespace: who.GetNamespace(),
		Name:      ref,
	}

	err := c.Get(ctx, key, &template)
	switch {
	case k8errors.IsNotFound(err):
		return nil, errors.Wrapf(err, "cannot find '%s at namespace '%s", ref, who.GetNamespace())
	case err != nil:
		return nil, errors.Wrapf(err, "cannot retrieve template [%s]", key.String())
	default:
		return &template, nil
	}
}

func GenerateFromScheme(spec interface{}, scheme *v1alpha1.Scheme, userInputs map[string]string) error {
	if userInputs != nil {
		if scheme.Inputs == nil || scheme.Inputs.Parameters == nil {
			return errors.New("template is not parameterizable")
		}

		for key, value := range userInputs {
			_, exists := scheme.Inputs.Parameters[key]
			if !exists {
				return errors.Errorf("parameter '%s' does not exist", key)
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
