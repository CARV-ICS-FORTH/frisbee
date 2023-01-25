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

package cluster

import (
	"context"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	serviceutils "github.com/carv-ics-forth/frisbee/controllers/service/utils"
	"github.com/carv-ics-forth/frisbee/pkg/distributions"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *Controller) runJob(ctx context.Context, cluster *v1alpha1.Cluster, jobIndex int) error {
	var job v1alpha1.Service

	// Populate the job
	job.SetName(common.GenerateName(cluster, jobIndex))
	v1alpha1.PropagateLabels(&job, cluster)

	// modulo is needed to re-iterate the job list, required for the implementation of "Until".
	jobSpec := cluster.Status.QueuedJobs[jobIndex%len(cluster.Status.QueuedJobs)]

	jobSpec.DeepCopyInto(&job.Spec)

	job.AttachTestDataVolume(cluster.Spec.TestData, true)

	if err := common.Create(ctx, r, cluster, &job); err != nil {
		return err
	}

	r.GetEventRecorderFor(cluster.GetName()).Event(cluster, corev1.EventTypeNormal, "Scheduled", job.GetName())

	return nil
}

func (r *Controller) constructJobSpecList(ctx context.Context, cluster *v1alpha1.Cluster) ([]v1alpha1.ServiceSpec, error) {
	serviceSpecs, err := serviceutils.GetServiceSpecList(ctx, r.GetClient(), cluster, cluster.Spec.GenerateObjectFromTemplate)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get serviceSpecs")
	}

	SetPlacement(cluster, serviceSpecs)

	SetResources(cluster, serviceSpecs)

	SetTimeline(cluster)

	return serviceSpecs, nil
}

func SetPlacement(cluster *v1alpha1.Cluster, services []v1alpha1.ServiceSpec) {
	if cluster.Spec.Placement == nil {
		return
	}

	/*
		Pod affinity and anti-affinity allows placing pods to nodes as a function of the labels of other pods.
		These Kubernetes features are useful in scenarios like: an application that consists of multiple services,
		some of which may require that they be co-located on the same node for performance reasons; replicas of
		critical services should not be placed onto the same node to avoid loss in the event of node failure.

		See: https://www.cncf.io/blog/2021/07/27/advanced-kubernetes-pod-to-node-scheduling/
	*/
	var affinity corev1.Affinity

	if cluster.Spec.Placement.Nodes != nil { // Match pods to a node
		affinity.NodeAffinity = &corev1.NodeAffinity{
			/*
				If the affinity requirements specified by this field are not met at
				scheduling time, the pod will not be scheduled onto the node.
				If the affinity requirements specified by this field cease to be met
				at some point during pod execution (e.g. due to an update), the system
				 may or may not try to eventually evict the pod from its node.
			*/
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      "kubernetes.io/hostname",
								Operator: corev1.NodeSelectorOpIn,
								Values:   cluster.Spec.Placement.Nodes,
							},
						},
					},
				},
			},
		}
	}

	if cluster.Spec.Placement.Collocate { // Place together all the Pods that belong to this cluster
		affinity.PodAffinity = &corev1.PodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
				{
					LabelSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      v1alpha1.LabelAction,
								Operator: metav1.LabelSelectorOpIn,
								Values:   []string{cluster.GetName()},
							},
						},
					},
					TopologyKey: "kubernetes.io/hostname",
				},
			},
		}
	}

	if cluster.Spec.Placement.ConflictsWith != nil { // Stay away from Pods that belong to other Clusters
		affinity.PodAntiAffinity = &corev1.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
				{
					LabelSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      v1alpha1.LabelAction,
								Operator: metav1.LabelSelectorOpIn,
								Values:   cluster.Spec.Placement.ConflictsWith,
							},
						},
					},
					TopologyKey: "kubernetes.io/hostname",
				},
			},
		}
	}

	// apply affinity rules to all specs
	for i := 0; i < len(services); i++ {
		// Apply the current rules.
		services[i].Affinity = &affinity
	}
}

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
