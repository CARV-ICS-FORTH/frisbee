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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func AppendLabel(cr client.Object, key string, value string) {
	cr.SetLabels(labels.Merge(cr.GetLabels(), map[string]string{key: value}))
}

func AppendLabels(cr client.Object, pairs map[string]string) {
	cr.SetLabels(labels.Merge(cr.GetLabels(), pairs))
}

func AppendAnnotation(cr client.Object, key string, value string) {
	cr.SetAnnotations(labels.Merge(cr.GetAnnotations(), map[string]string{key: value}))
}

func AppendAnnotations(cr client.Object, pairs map[string]string) {
	cr.SetAnnotations(labels.Merge(cr.GetAnnotations(), pairs))
}

func IsSystemService(job client.Object) bool {
	if typeOf := job.GetLabels()[v1alpha1.LabelComponent]; typeOf == v1alpha1.ComponentSys {
		return true
	}
	return false
}

func SpecForSystemService(spec *v1alpha1.ServiceSpec) bool {
	if spec.Decorators != nil && spec.Decorators.Labels != nil {
		return spec.Decorators.Labels[v1alpha1.LabelComponent] == v1alpha1.ComponentSys
	}

	return false
}

func ExtractPartOfLabel(obj metav1.Object) (string, error) {
	plan, ok := obj.GetLabels()[v1alpha1.LabelPartOfPlan]
	if !ok {
		return "", errors.Errorf("Cannot extract label '%s' from resource '%s'. Labels: %s",
			v1alpha1.LabelPartOfPlan, obj.GetName(), obj.GetLabels())
	}

	return plan, nil
}

// Discover discovers a resource across different namespaces
func Discover(ctx context.Context, c client.Client, crList client.ObjectList, id string) error {
	// find the platform configuration (which may reside on a different namespace)
	filters := []client.ListOption{
		client.MatchingLabels{v1alpha1.ResourceDiscoveryLabel: id},
	}

	if err := c.List(ctx, crList, filters...); err != nil {
		return errors.Wrapf(err, "cannot list resources")
	}

	return nil
}
