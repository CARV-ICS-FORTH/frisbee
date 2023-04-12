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

func SetResources(cluster *v1alpha1.Cluster, services []v1alpha1.ServiceSpec) {
	if cluster.Spec.Resources == nil {
		return
	}

	var generator distributions.ProbabilitySlice

	// Default distributions means loads the evaluated distribution from the status of the resource.
	if cluster.Spec.Resources.DistributionSpec.Name == v1alpha1.DistributionDefault {
		generator = cluster.Status.DefaultDistribution
	} else {
		generator = distributions.GenerateProbabilitySliceFromSpec(int64(cluster.Spec.MaxInstances), cluster.Spec.Resources.DistributionSpec)
	}

	resources := generator.ApplyToResources(cluster.Spec.Resources.TotalResources)

	// apply the resource distribution to the Main container of each pod.
	for i := range services {
		for ci, c := range services[i].Containers {
			if c.Name == v1alpha1.MainContainerName {
				services[i].Containers[ci].Resources.Requests = resources[i]
				services[i].Containers[ci].Resources.Limits = resources[i]
			}
		}
	}
}
