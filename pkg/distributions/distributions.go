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

package distributions

import (
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/pingcap/go-ycsb/pkg/generator"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Generator interface {
	Next(r *rand.Rand) int64
	Last() int64
}

func GetPointDistribution(nodes int64, spec *v1alpha1.DistributionSpec) PointDistribution {
	switch strings.ToLower(spec.Distribution) {
	case "constant":
		if spec.DistParamsConstant == nil {
			spec.DistParamsConstant = &v1alpha1.DistParamsConstant{Value: nodes}
		}
		return NewPointDistribution(nodes,
			generator.NewConstant(spec.DistParamsConstant.Value))

	case "uniform":
		if spec.DistParamsUniform == nil {
			spec.DistParamsUniform = &v1alpha1.DistParamsUniform{MaxValue: nodes}
		}
		return NewPointDistribution(nodes,
			generator.NewUniform(1, spec.DistParamsUniform.MaxValue))

	case "zipfian":
		if spec.DistParamsZipfian == nil {
			spec.DistParamsZipfian = &v1alpha1.DistParamsZipfian{MaxValue: nodes}
		}

		return NewPointDistribution(nodes,
			generator.NewZipfianWithRange(1, spec.DistParamsZipfian.MaxValue, generator.ZipfianConstant))

	case "histogram":
		if spec.DistParamsHistogram == nil {
			spec.DistParamsHistogram = &v1alpha1.DistParamsHistogram{ // fixme: are these defaults reasonable ?
				Buckets:   []int64{10, 10, 10, 10},
				BlockSize: 5,
			}
		}

		return NewPointDistribution(nodes,
			generator.NewHistogram(spec.DistParamsHistogram.Buckets, spec.DistParamsHistogram.BlockSize))
	default:
		// This condition should be captured by upper layers.
		panic(errors.Errorf("unknown resource distribution %s", spec.Distribution))
	}
}

func NewPointDistribution(nodes int64, distgenerator Generator) PointDistribution {
	// calculate distribution
	s1 := rand.NewSource(0)
	r1 := rand.New(s1)

	r1.Int63()

	// calculate distribution
	dist := make(PointDistribution, nodes)
	for i := 0; i < int(nodes); i++ {
		dist[i] = float64(distgenerator.Next(r1))
	}

	// normalize not to exceed total.
	return dist.divide(dist.sum())
}

type PointDistribution []float64

func (dist PointDistribution) sum() float64 {
	var sum float64

	for _, v := range dist {
		sum += v
	}

	return sum
}

func (dist PointDistribution) divide(factor float64) PointDistribution {
	if factor == 0 {
		panic("divide by zero factor")
	}

	cp := make(PointDistribution, len(dist))

	for i, v := range dist {
		cp[i] = v / factor
	}

	return cp
}

func (dist PointDistribution) ApplyToFloat64(total float64) []float64 {
	cp := make([]float64, len(dist))

	for i, v := range dist {
		cp[i] = v * total
	}

	return cp
}

func (dist PointDistribution) ApplyToInt64(total int64) []int64 {
	cp := make([]int64, len(dist))

	for i, node := range dist {
		cp[i] = int64(math.Round(node * float64(total)))
	}

	return cp
}

func (dist PointDistribution) ApplyToTimeline(startingTime metav1.Time, total metav1.Duration) v1alpha1.Timeline {
	cp := make(v1alpha1.Timeline, len(dist))

	// progress normalizes the interval points to the starting time.
	progress := startingTime.Time

	for i, node := range dist {
		nextInterval := time.Duration(int64(math.Round(node*total.Seconds()))) * time.Second

		progress = progress.Add(nextInterval)

		cp[i] = metav1.Time{Time: progress}
	}

	return cp
}

func (dist PointDistribution) ApplyToResources(total corev1.ResourceList) v1alpha1.ResourceDistribution {
	cp := make(v1alpha1.ResourceDistribution, len(dist))

	for i, node := range dist {
		cp[i] = corev1.ResourceList{}

		if total.Cpu().Value() > 1 {
			val := int64(math.Round(node * float64(total.Cpu().ScaledValue(resource.Milli))))

			cp[i][corev1.ResourceCPU] = *resource.NewScaledQuantity(val, resource.Milli)
		}

		if total.Memory().Value() > 1 {
			val := int64(math.Round(node * float64(total.Memory().ScaledValue(resource.Mega))))

			cp[i][corev1.ResourceMemory] = *resource.NewScaledQuantity(val, resource.Mega)
		}

		if total.Pods().Value() > 1 {
			val := int64(math.Round(node * total.Pods().AsApproximateFloat64()))
			cp[i][corev1.ResourcePods] = *resource.NewQuantity(val, total.Pods().Format)
		}

		if total.Storage().Value() > 1 {
			val := int64(math.Round(node * float64(total.Storage().ScaledValue(resource.Mega))))

			cp[i][corev1.ResourceStorage] = *resource.NewScaledQuantity(val, resource.Mega)

		}

		if total.StorageEphemeral().Value() > 1 {
			val := int64(math.Round(node * float64(total.StorageEphemeral().ScaledValue(resource.Mega))))

			cp[i][corev1.ResourceEphemeralStorage] = *resource.NewScaledQuantity(val, resource.Mega)
		}
	}

	return cp
}
