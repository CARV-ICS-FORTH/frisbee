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

package chaos

import (
	"context"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/pkg/errors"
)

func (r *Controller) runJob(ctx context.Context, cr *v1alpha1.Chaos) error {
	var f GenericFault

	if err := getRawManifest(cr, &f); err != nil {
		return errors.Wrapf(err, "cannot get manifest for chaos '%s'", cr.GetName())
	}

	f.SetLabels(labels.Merge(f.GetLabels(), cr.GetLabels()))
	f.SetAnnotations(labels.Merge(f.GetAnnotations(), cr.GetAnnotations()))

	if err := common.Create(ctx, r, cr, &f); err != nil {
		return errors.Wrapf(err, "cannot inject fault for chaos '%s'", cr.GetName())
	}

	return nil
}
