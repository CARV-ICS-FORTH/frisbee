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

package cascade

import (
	"context"

	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/carv-ics-forth/frisbee/pkg/distributions"
	corev1 "k8s.io/api/core/v1"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	chaosutils "github.com/carv-ics-forth/frisbee/controllers/chaos/utils"
	"github.com/pkg/errors"
)

func (r *Controller) runJob(ctx context.Context, cr *v1alpha1.Cascade, i int) error {
	var job v1alpha1.Chaos

	// Populate the job
	job.SetName(common.GenerateName(cr, i, cr.Spec.MaxInstances))

	v1alpha1.SetScenarioLabel(&job.ObjectMeta, v1alpha1.GetScenarioLabel(cr))
	v1alpha1.SetComponentLabel(&job.ObjectMeta, v1alpha1.GetComponentLabel(cr))

	// modulo is needed to re-iterate the job list, required for the implementation of "Until".
	jobSpec := cr.Status.QueuedJobs[i%len(cr.Status.QueuedJobs)]

	jobSpec.DeepCopyInto(&job.Spec)

	if err := common.Create(ctx, r, cr, &job); err != nil {
		return err
	}

	r.GetEventRecorderFor(cr.GetName()).Event(cr, corev1.EventTypeNormal, "Scheduled", job.GetName())

	return nil
}

func (r *Controller) constructJobSpecList(ctx context.Context, cr *v1alpha1.Cascade) ([]v1alpha1.ChaosSpec, error) {
	specs, err := chaosutils.GetChaosSpecList(ctx, r.GetClient(), cr, cr.Spec.GenerateObjectFromTemplate)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get specs")
	}

	SetTimeline(cr)

	return specs, nil
}

func SetTimeline(cr *v1alpha1.Cascade) {
	if cr.Spec.Schedule == nil || cr.Spec.Schedule.Timeline == nil {
		return
	}

	generator := distributions.GetPointDistribution(int64(cr.Spec.MaxInstances),
		cr.Spec.Schedule.Timeline.DistributionSpec)

	cr.Status.Timeline = generator.ApplyToTimeline(cr.GetCreationTimestamp(),
		*cr.Spec.Schedule.Timeline.TotalDuration)

}
