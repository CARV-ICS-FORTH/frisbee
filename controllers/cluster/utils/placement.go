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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
