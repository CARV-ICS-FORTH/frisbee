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

package distributions

import (
	"math"
	"time"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Generator interface {
	Next() float64
	Last() float64
}

func GenerateProbabilitySliceFromSpec(samples int64, spec *v1alpha1.DistributionSpec) ProbabilitySlice {
	switch spec.Name {
	case v1alpha1.DistributionDefault:
		panic("default distribution is a pointer to an already evaluated distribution, and therefore it should be handled before reaching this point")

	case v1alpha1.DistributionUniform:
		return GenerateProbabilitySlice(samples, NewUniform(1, samples))

	case v1alpha1.DistributionNormal:
		return GenerateProbabilitySlice(samples, NewNormal(1, samples))

	case v1alpha1.DistributionPareto:
		if spec.DistParamsPareto == nil {
			spec.DistParamsPareto = &v1alpha1.DistParamsPareto{
				Scale: DefaultParetoScale,
				Shape: DefaultParetoShape,
			}
		}

		return GenerateProbabilitySlice(samples, NewPareto(spec.DistParamsPareto.Scale, spec.DistParamsPareto.Shape))

	default:
		// This condition should be captured by upper layers.
		panic(errors.Errorf("unknown resource distribution %s", spec.Name))
	}
}

func GenerateProbabilitySlice(samples int64, distgenerator Generator) ProbabilitySlice {
	// discard the first 0 value
	distgenerator.Next()

	dist := make(ProbabilitySlice, samples)
	for i := 0; i < int(samples); i++ {
		// computes the value of the probability density function at x.
		dist[i] = distgenerator.Next()
	}

	// normalize not to exceed total.
	return dist.divide(dist.sum())
}

// ProbabilitySlice provides the value of the probability density function at x.
type ProbabilitySlice []float64

func (dist ProbabilitySlice) sum() float64 {
	var sum float64

	for _, v := range dist {
		sum += v
	}

	return sum
}

func (dist ProbabilitySlice) divide(factor float64) ProbabilitySlice {
	if factor == 0 {
		panic("divide by zero factor")
	}

	divided := make(ProbabilitySlice, len(dist))

	for i, v := range dist {
		// round-up to two decimal places
		divided[i] = math.Round(100*v/factor) / 100
	}

	return divided
}

func (dist ProbabilitySlice) ApplyToFloat64(total float64) []float64 {
	float64Distribution := make([]float64, len(dist))

	for i, v := range dist {
		float64Distribution[i] = v * total
	}

	return float64Distribution
}

func (dist ProbabilitySlice) ApplyToInt64(total int64) []int64 {
	int64Distribution := make([]int64, len(dist))

	for i, node := range dist {
		int64Distribution[i] = int64(math.Round(node * float64(total)))
	}

	return int64Distribution
}

func (dist ProbabilitySlice) ApplyToTimeline(startingTime metav1.Time, total metav1.Duration) v1alpha1.Timeline {
	timelineDistribution := make(v1alpha1.Timeline, len(dist))

	// progress normalizes the interval points to the starting time.
	progress := startingTime.Time

	for i, node := range dist {
		nextInterval := time.Duration(int64(math.Round(node*total.Seconds()))) * time.Second

		progress = progress.Add(nextInterval)

		timelineDistribution[i] = metav1.Time{Time: progress}
	}

	return timelineDistribution
}

func (dist ProbabilitySlice) ApplyToResources(total corev1.ResourceList) v1alpha1.ResourceDistribution {
	resourceDistribution := make(v1alpha1.ResourceDistribution, len(dist))

	for i, prob := range dist {
		resourceDistribution[i] = corev1.ResourceList{}

		if total.Cpu().Value() > 1 {
			val := int64(math.Round(prob * float64(total.Cpu().ScaledValue(resource.Milli))))

			resourceDistribution[i][corev1.ResourceCPU] = *resource.NewScaledQuantity(val, resource.Milli)
		}

		if total.Memory().Value() > 1 {
			val := int64(math.Round(prob * float64(total.Memory().ScaledValue(resource.Mega))))

			resourceDistribution[i][corev1.ResourceMemory] = *resource.NewScaledQuantity(val, resource.Mega)
		}

		if total.Pods().Value() > 1 {
			val := int64(math.Round(prob * total.Pods().AsApproximateFloat64()))

			resourceDistribution[i][corev1.ResourcePods] = *resource.NewQuantity(val, total.Pods().Format)
		}

		if total.Storage().Value() > 1 {
			val := int64(math.Round(prob * float64(total.Storage().ScaledValue(resource.Mega))))

			resourceDistribution[i][corev1.ResourceStorage] = *resource.NewScaledQuantity(val, resource.Mega)
		}

		if total.StorageEphemeral().Value() > 1 {
			val := int64(math.Round(prob * float64(total.StorageEphemeral().ScaledValue(resource.Mega))))

			resourceDistribution[i][corev1.ResourceEphemeralStorage] = *resource.NewScaledQuantity(val, resource.Mega)
		}
	}

	return resourceDistribution
}
