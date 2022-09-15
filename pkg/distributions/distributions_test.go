package distributions

import (
	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/pingcap/go-ycsb/pkg/generator"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"testing"
	"time"
)

func TestPointDistribution_ApplyToResources(t *testing.T) {
	var N = int64(10)

	// the way to distribute resources
	constant := NewPointDistribution(N, generator.NewConstant(N))
	uniform := NewPointDistribution(N, generator.NewUniform(1, N))
	zipfian := NewPointDistribution(N, generator.NewZipfianWithRange(1, N, generator.ZipfianConstant))
	histogram := NewPointDistribution(N, generator.NewHistogram([]int64{10, 10, 10, 10}, 5))

	// the total resources to distribute
	total := corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse("22"),
		corev1.ResourceMemory:           resource.MustParse("1G"),
		corev1.ResourcePods:             resource.MustParse("105"),
		corev1.ResourceStorage:          resource.MustParse("2G"),
		corev1.ResourceEphemeralStorage: resource.MustParse("3G"),
	}

	type args struct {
		total corev1.ResourceList
	}
	tests := []struct {
		name string
		dist PointDistribution
		args args
		want v1alpha1.ResourceDistribution
	}{
		{
			name: "constant",
			dist: constant,
			args: args{total: total},
			want: nil,
		},
		{
			name: "uniform",
			dist: uniform,
			args: args{total: total},
			want: nil,
		},
		{
			name: "zipfian",
			dist: zipfian,
			args: args{total: total},
			want: nil,
		},
		{
			name: "histogram",
			dist: histogram,
			args: args{total: total},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.dist.ApplyToResources(tt.args.total); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ApplyToResources() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPointDistribution_ApplyToTimeline(t *testing.T) {
	var N = int64(10)

	// the way to distribute resources
	constant := NewPointDistribution(N, generator.NewConstant(N))
	uniform := NewPointDistribution(N, generator.NewUniform(1, N))
	zipfian := NewPointDistribution(N, generator.NewZipfianWithRange(1, N, generator.ZipfianConstant))
	histogram := NewPointDistribution(N, generator.NewHistogram([]int64{10, 10, 10, 10}, 5))

	// the total resources to distribute
	total := metav1.Duration{Duration: 5 * time.Minute}

	type args struct {
		startingTime metav1.Time
		total        metav1.Duration
	}
	tests := []struct {
		name string
		dist PointDistribution
		args args
		want v1alpha1.Timeline
	}{
		{
			name: "constant",
			dist: constant,
			args: args{total: total},
			want: nil,
		},
		{
			name: "uniform",
			dist: uniform,
			args: args{total: total},
			want: nil,
		},
		{
			name: "zipfian",
			dist: zipfian,
			args: args{total: total},
			want: nil,
		},
		{
			name: "histogram",
			dist: histogram,
			args: args{total: total},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.dist.ApplyToTimeline(tt.args.startingTime, tt.args.total); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ApplyToTimeline() = %v, want %v", got, tt.want)
			}
		})
	}
}
