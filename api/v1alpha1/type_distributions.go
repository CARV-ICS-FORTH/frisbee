/*
Copyright 2022-2023 ICS-FORTH.

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

package v1alpha1

import (
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DistributionName string

const (
	// DistributionConstant is a fixed distributed with all elements having a probability of 1.
	DistributionConstant DistributionName = "constant"

	// DistributionUniform draws samples from a continuous uniform distribution
	DistributionUniform DistributionName = "uniform"

	// DistributionNormal draws samples from a normal (Gaussian) distribution
	DistributionNormal DistributionName = "normal"

	// DistributionPareto draws samples from a Pareto distribution
	DistributionPareto DistributionName = "pareto"

	// DistributionDefault instructs the controller to use an already evaluated distribution.
	DistributionDefault DistributionName = "default"
)

type DistributionSpec struct {
	// +kubebuilder:validation:Enum=constant;uniform;normal;pareto;default
	Name DistributionName `json:"name"`

	// +optional
	*DistParamsPareto `json:"histogram,omitempty"`
}

// DistParamsPareto are parameters for the Pareto distribution.
type DistParamsPareto struct {
	Scale float64 `json:"scale"`
	Shape float64 `json:"shape"`
}

/*

	Timeline Distribution

*/

type TimelineDistributionSpec struct {
	// DistributionSpec defines how the TotalDuration will be divided into time-based events.
	DistributionSpec *DistributionSpec `json:"distribution"`

	// TotalDuration defines the total duration within which events will happen.
	TotalDuration *metav1.Duration `json:"total"`
}

type Timeline []metav1.Time

// Next returns the next activation time, later than the given time.
func (in Timeline) Next(ref time.Time) time.Time {
	for _, t := range in {
		if t.After(ref) {
			return t.Time
		}
	}

	// bad hack. If there is no actual schedule, return something far in the future
	// for the controller to keep running, but also to raise trigger to the test.
	return time.Now().Add(12 * time.Hour)
}

func (in Timeline) String() string {
	var out strings.Builder

	out.WriteString("\n=== Timeline ===\n")

	for _, node := range in {
		out.WriteString(fmt.Sprintf("\n * %s", node.Time.Format(time.StampMilli)))
	}

	return out.String()
}

/*

	Resource Distribution

*/

type ResourceDistributionSpec struct {
	// DistributionSpec defines how the TotalResources will be assigned to resources.
	DistributionSpec *DistributionSpec `json:"distribution,omitempty"`

	// TotalResources defines the total resources that will be distributed among the cluster's services.
	TotalResources corev1.ResourceList `json:"total"`
}

type ResourceDistribution []corev1.ResourceList

func (in ResourceDistribution) Table() (header []string, data [][]string) {
	header = []string{
		"CPU",
		"Memory",
		"Pods",
		"Storage",
		"Ephemeral",
	}

	for _, node := range in {
		data = append(data, []string{
			fmt.Sprintf("%.2f", node.Cpu().AsApproximateFloat64()),
			fmt.Sprintf("%.2f", node.Memory().AsApproximateFloat64()),
			fmt.Sprintf("%.2f", node.Pods().AsApproximateFloat64()),
			fmt.Sprintf("%.2f", node.Storage().AsApproximateFloat64()),
			fmt.Sprintf("%.2f", node.StorageEphemeral().AsApproximateFloat64()),
		})
	}

	return header, data
}

func (in ResourceDistribution) String() string {
	var out strings.Builder

	out.WriteString("\n=== Resource Distribution ===\n")

	for i, node := range in {
		out.WriteString(fmt.Sprintf("\n=== node_%d ===", i))
		out.WriteString(fmt.Sprintf("\n* CPU: %.2f", node.Cpu().AsApproximateFloat64()))
		out.WriteString(fmt.Sprintf("\n* Memory: %.2f", node.Memory().AsApproximateFloat64()))
		out.WriteString(fmt.Sprintf("\n* Pods: %.2f", node.Pods().AsApproximateFloat64()))
		out.WriteString(fmt.Sprintf("\n* Storage: %.2f", node.Storage().AsApproximateFloat64()))
		out.WriteString(fmt.Sprintf("\n* StorageEphemeral: %.2f", node.StorageEphemeral().AsApproximateFloat64()))
	}

	/*
		sum := dist.Sum()
		out.WriteString(fmt.Sprintf("\n -- TotalResources -- \nCPU:%.2f Memory:%.2f Pods:%.2f Storage:%.2f Ephemeral:%.2f",
			sum.Cpu().AsApproximateFloat64(), sum.Memory().AsApproximateFloat64(), sum.Pods().AsApproximateFloat64(),
			sum.Storage().AsApproximateFloat64(), sum.StorageEphemeral().AsApproximateFloat64()))

	*/

	return out.String()
}
