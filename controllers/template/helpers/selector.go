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

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/utils"
	"github.com/pkg/errors"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ParseRef parse the templateRef and returns a template selector. If the templateRef is invalid, the selector
// will be nil, and any subsequence select operation will return empty value.
func ParseRef(nm, templateRef string) *v1alpha1.TemplateSelector {
	parsed := strings.Split(templateRef, "/")
	if len(parsed) != 2 {
		panic("invalid reference format")
	}

	family := parsed[0]
	ref := parsed[1]

	return &v1alpha1.TemplateSelector{
		Namespace: nm,
		Family:    family,
		Selector: v1alpha1.TemplateSelectorSpec{
			Reference: ref,
		},
	}
}

func Select(ctx context.Context, r utils.Reconciler, ts *v1alpha1.TemplateSelector) (v1alpha1.Scheme, error) {
	if ts == nil {
		return v1alpha1.Scheme{}, errors.New("empty selector")
	}

	var template v1alpha1.Template

	templateKey := client.ObjectKey{
		Namespace: ts.Namespace,
		Name:      ts.Family,
	}

	// if the template is created in parallel with the workflow, it is possible to meet race conditions.
	// We avoid it with a simple retry mechanism based on adaptive backoff.
	err := retry.OnError(retry.DefaultRetry, k8errors.IsNotFound, func() error {
		return r.GetClient().Get(ctx, templateKey, &template)
	})
	if err != nil {
		return v1alpha1.Scheme{}, errors.Wrapf(err, "cannot find template %s", templateKey.String())
	}

	// TODO: Change Get to List

	switch {
	case ts.Selector.Reference != "":
		scheme, ok := template.Spec.Entries[ts.Selector.Reference]
		if !ok {
			return v1alpha1.Scheme{}, errors.Errorf("no %s scheme is defined in template %s",
				ts.Selector.Reference, templateKey.String())
		}

		return scheme, nil

	default:
		panic(errors.Errorf("unspecified selection criteria"))
	}
}
