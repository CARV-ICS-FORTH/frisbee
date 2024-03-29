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

package cascade

import (
	"context"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	cascadeutils "github.com/carv-ics-forth/frisbee/controllers/cascade/utils"
	chaosutils "github.com/carv-ics-forth/frisbee/controllers/chaos/utils"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

func (r *Controller) runJob(ctx context.Context, cascade *v1alpha1.Cascade, jobIndex int) error {
	var job v1alpha1.Chaos

	// Populate the job
	job.SetName(common.GenerateName(cascade, jobIndex))
	v1alpha1.PropagateLabels(&job, cascade)

	// modulo is needed to re-iterate the job list, required for the implementation of "Until".
	jobSpec := cascade.Status.QueuedJobs[jobIndex%len(cascade.Status.QueuedJobs)]

	jobSpec.DeepCopyInto(&job.Spec)

	if err := common.Create(ctx, r, cascade, &job); err != nil {
		return err
	}

	r.GetEventRecorderFor(cascade.GetName()).Event(cascade, corev1.EventTypeNormal, "Scheduled", job.GetName())

	return nil
}

// buildJobQueue creates a list of job templates that will be scheduled throughout execution.
func (r *Controller) buildJobQueue(ctx context.Context, cascade *v1alpha1.Cascade) ([]v1alpha1.ChaosSpec, error) {
	chaosSpecs, err := chaosutils.GetChaosSpecList(ctx, r.GetClient(), cascade, cascade.Spec.GenerateObjectFromTemplate)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get chaosSpecs")
	}

	cascadeutils.SetTimeline(cascade)

	return chaosSpecs, nil
}
