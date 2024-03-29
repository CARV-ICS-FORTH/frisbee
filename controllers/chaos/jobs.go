/*
Copyright 2021-2023 ICS-FORTH.

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

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/labels"
)

func (r *Controller) runJob(ctx context.Context, chaos *v1alpha1.Chaos) error {
	var fault GenericFault

	if err := getRawManifest(chaos, &fault); err != nil {
		return errors.Wrapf(err, "cannot get manifest for chaos '%s'", chaos.GetName())
	}

	fault.SetLabels(labels.Merge(fault.GetLabels(), chaos.GetLabels()))
	fault.SetAnnotations(labels.Merge(fault.GetAnnotations(), chaos.GetAnnotations()))

	if err := common.Create(ctx, r, chaos, &fault); err != nil {
		return errors.Wrapf(err, "failed to inject chaos type '%s'", chaos.Kind)
	}

	return nil
}
