package distributions_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/pkg/distributions"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_ProbabilityGenerator(t *testing.T) {
	// number of totals bins
	Samples := int64(5)

	tests := []struct {
		name     string
		dist     distributions.ProbabilitySlice
		expected distributions.ProbabilitySlice
	}{
		{
			name: "constant",
			dist: distributions.GenerateProbabilitySliceFromSpec(Samples,
				&v1alpha1.DistributionSpec{Name: "constant"},
			),
			expected: distributions.ProbabilitySlice{1, 1, 1, 1, 1},
		},
		{
			name: "uniform",
			dist: distributions.GenerateProbabilitySliceFromSpec(Samples,
				&v1alpha1.DistributionSpec{Name: "uniform"},
			),
			expected: distributions.ProbabilitySlice{0.2, 0.2, 0.2, 0.2, 0.2},
		},
		{
			name: "normal",
			dist: distributions.GenerateProbabilitySliceFromSpec(Samples,
				&v1alpha1.DistributionSpec{Name: "normal"},
			),
			expected: distributions.ProbabilitySlice{0.19, 0.21, 0.21, 0.21, 0.19},
		},
		{
			name: "pareto",
			dist: distributions.GenerateProbabilitySliceFromSpec(Samples,
				&v1alpha1.DistributionSpec{
					Name: "pareto",
					DistParamsPareto: &v1alpha1.DistParamsPareto{
						Scale: 1,
						Shape: 0.1,
					},
				},
			),
			expected: distributions.ProbabilitySlice{0.46, 0.22, 0.14, 0.1, 0.08},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !reflect.DeepEqual(tt.dist, tt.expected) {
				t.Errorf("expected '%v' but got '%v'", tt.expected, tt.dist)
			}
		})
	}
}

func Test_ResourceDistribution(t *testing.T) {
	Nodes := int64(5)

	// the total resources to distribute
	total := corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("40"),
		corev1.ResourceMemory: resource.MustParse("40G"),
	}

	type args struct {
		total corev1.ResourceList
	}

	tests := []struct {
		name string
		dist distributions.ProbabilitySlice
		args args
		want v1alpha1.ResourceDistribution
	}{
		{
			name: "constant",
			dist: distributions.GenerateProbabilitySliceFromSpec(Nodes,
				&v1alpha1.DistributionSpec{Name: "constant"},
			),
			args: args{total: total},
			want: []corev1.ResourceList{
				{
					corev1.ResourceCPU:    resource.MustParse("40"),
					corev1.ResourceMemory: resource.MustParse("40G"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("40"),
					corev1.ResourceMemory: resource.MustParse("40G"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("40"),
					corev1.ResourceMemory: resource.MustParse("40G"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("40"),
					corev1.ResourceMemory: resource.MustParse("40G"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("40"),
					corev1.ResourceMemory: resource.MustParse("40G"),
				},
			},
		},
		{
			name: "uniform",
			dist: distributions.GenerateProbabilitySliceFromSpec(Nodes,
				&v1alpha1.DistributionSpec{Name: "uniform"},
			),
			args: args{total: total},
			want: []corev1.ResourceList{
				{
					corev1.ResourceCPU:    resource.MustParse("8"),
					corev1.ResourceMemory: resource.MustParse("8G"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("8"),
					corev1.ResourceMemory: resource.MustParse("8G"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("8"),
					corev1.ResourceMemory: resource.MustParse("8G"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("8"),
					corev1.ResourceMemory: resource.MustParse("8G"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("8"),
					corev1.ResourceMemory: resource.MustParse("8G"),
				},
			},
		},
		{
			name: "normal",
			dist: distributions.GenerateProbabilitySliceFromSpec(Nodes,
				&v1alpha1.DistributionSpec{Name: "normal"},
			),
			args: args{total: total},
			want: []corev1.ResourceList{
				{
					corev1.ResourceCPU:    resource.MustParse("7.6"),
					corev1.ResourceMemory: resource.MustParse("7.6G"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("8.4"),
					corev1.ResourceMemory: resource.MustParse("8.4G"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("8.4"),
					corev1.ResourceMemory: resource.MustParse("8.4G"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("8.4"),
					corev1.ResourceMemory: resource.MustParse("8.4G"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("7.6"),
					corev1.ResourceMemory: resource.MustParse("7.6G"),
				},
			},
		},
		{
			name: "pareto",
			dist: distributions.GenerateProbabilitySliceFromSpec(Nodes,
				&v1alpha1.DistributionSpec{
					Name: "pareto",
					DistParamsPareto: &v1alpha1.DistParamsPareto{
						Scale: 1,
						Shape: 0.1,
					},
				},
			),
			args: args{total: total},
			want: []corev1.ResourceList{
				{
					corev1.ResourceCPU:    resource.MustParse("18.4"),
					corev1.ResourceMemory: resource.MustParse("18.4G"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("8.8"),
					corev1.ResourceMemory: resource.MustParse("8.8G"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("5.6"),
					corev1.ResourceMemory: resource.MustParse("5.6G"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("4G"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("3.2"),
					corev1.ResourceMemory: resource.MustParse("3.2G"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resourceDistribution := tt.dist.ApplyToResources(tt.args.total)

			for i, elem := range resourceDistribution {
				wanted := tt.want[i]
				if elem.Cpu().String() != wanted.Cpu().String() || elem.Memory().String() != wanted.Memory().String() {
					t.Errorf("ApplyToResources(%d) = %v, want %v", i, elem, wanted)
				}
			}
		})
	}
}

func Test_TimelineDistribution(t *testing.T) {
	Timesteps := int64(5)

	// the total duration of the experiment
	startingTime := metav1.Time{}
	total := metav1.Duration{Duration: 5 * time.Minute}

	type args struct {
		total metav1.Duration
	}

	tests := []struct {
		name string
		dist distributions.ProbabilitySlice
		args args
		want v1alpha1.Timeline
	}{
		{
			name: "constant",
			dist: distributions.GenerateProbabilitySliceFromSpec(Timesteps,
				&v1alpha1.DistributionSpec{Name: "constant"},
			),
			args: args{total: total},
			want: []metav1.Time{
				{Time: startingTime.Add(300 * time.Second)},
				{Time: startingTime.Add(600 * time.Second)},
				{Time: startingTime.Add(900 * time.Second)},
				{Time: startingTime.Add(1200 * time.Second)},
				{Time: startingTime.Add(1500 * time.Second)},
			},
		},
		{
			name: "uniform",
			dist: distributions.GenerateProbabilitySliceFromSpec(Timesteps,
				&v1alpha1.DistributionSpec{Name: "uniform"},
			),
			args: args{total: total},
			want: []metav1.Time{
				{Time: startingTime.Add(60 * time.Second)},
				{Time: startingTime.Add(120 * time.Second)},
				{Time: startingTime.Add(180 * time.Second)},
				{Time: startingTime.Add(240 * time.Second)},
				{Time: startingTime.Add(300 * time.Second)},
			},
		},
		{
			name: "normal",
			dist: distributions.GenerateProbabilitySliceFromSpec(Timesteps,
				&v1alpha1.DistributionSpec{Name: "normal"},
			),
			args: args{total: total},
			want: []metav1.Time{
				{Time: startingTime.Add(57 * time.Second)},
				{Time: startingTime.Add(120 * time.Second)},
				{Time: startingTime.Add(183 * time.Second)},
				{Time: startingTime.Add(246 * time.Second)},
				{Time: startingTime.Add(303 * time.Second)},
			},
		},
		{
			name: "pareto",
			dist: distributions.GenerateProbabilitySliceFromSpec(Timesteps,
				&v1alpha1.DistributionSpec{
					Name: "pareto",
					DistParamsPareto: &v1alpha1.DistParamsPareto{
						Scale: 1,
						Shape: 0.1,
					},
				},
			),
			args: args{total: total},
			want: []metav1.Time{
				{Time: startingTime.Add(138 * time.Second)},
				{Time: startingTime.Add(204 * time.Second)},
				{Time: startingTime.Add(246 * time.Second)},
				{Time: startingTime.Add(276 * time.Second)},
				{Time: startingTime.Add(300 * time.Second)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timelineDistribution := tt.dist.ApplyToTimeline(startingTime, tt.args.total)

			if !reflect.DeepEqual(timelineDistribution, tt.want) {
				t.Errorf("ApplyToTimeline = %v, want %v", timelineDistribution, tt.want)
			}
		})
	}
}
