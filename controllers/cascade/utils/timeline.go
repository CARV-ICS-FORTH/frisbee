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

package utils

import (
	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/pkg/distributions"
)

func SetTimeline(cascade *v1alpha1.Cascade) {
	if cascade.Spec.Schedule == nil || cascade.Spec.Schedule.Timeline == nil {
		return
	}

	probabilitySlice := distributions.GenerateProbabilitySliceFromSpec(int64(cascade.Spec.MaxInstances),
		cascade.Spec.Schedule.Timeline.DistributionSpec)

	cascade.Status.ExpectedTimeline = probabilitySlice.ApplyToTimeline(
		cascade.GetCreationTimestamp(),
		*cascade.Spec.Schedule.Timeline.TotalDuration,
	)
}
