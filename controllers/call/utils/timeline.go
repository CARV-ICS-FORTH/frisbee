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

func SetTimeline(call *v1alpha1.Call) {
	if call.Spec.Schedule == nil || call.Spec.Schedule.Timeline == nil {
		return
	}

	probabilitySlice := distributions.GenerateProbabilitySliceFromSpec(int64(len(call.Spec.Services)),
		call.Spec.Schedule.Timeline.DistributionSpec)

	call.Status.ExpectedTimeline = probabilitySlice.ApplyToTimeline(
		call.GetCreationTimestamp(),
		*call.Spec.Schedule.Timeline.TotalDuration,
	)
}
