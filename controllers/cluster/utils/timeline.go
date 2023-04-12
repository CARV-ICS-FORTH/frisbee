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

func SetTimeline(cluster *v1alpha1.Cluster) {
	if cluster.Spec.Schedule == nil || cluster.Spec.Schedule.Timeline == nil {
		return
	}

	var probabilitySlice distributions.ProbabilitySlice

	if cluster.Spec.Schedule.Timeline.DistributionSpec.Name == v1alpha1.DistributionDefault {
		probabilitySlice = cluster.Status.DefaultDistribution
	} else {
		probabilitySlice = distributions.GenerateProbabilitySliceFromSpec(int64(cluster.Spec.MaxInstances),
			cluster.Spec.Schedule.Timeline.DistributionSpec)
	}

	cluster.Status.ExpectedTimeline = probabilitySlice.ApplyToTimeline(
		cluster.GetCreationTimestamp(),
		*cluster.Spec.Schedule.Timeline.TotalDuration,
	)
}
