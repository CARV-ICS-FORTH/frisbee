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

package cluster

import (
	"context"
	"fmt"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common/labelling"
	serviceutils "github.com/carv-ics-forth/frisbee/controllers/service/utils"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// if there is only one instance, it will be named after the group. otherwise, the instances will be named
// as Master-0, Master-1, ...
func generateName(group *v1alpha1.Cluster, i int) string {
	if group.Spec.MaxInstances == 1 {
		return group.GetName()
	}

	return fmt.Sprintf("%s-%d", group.GetName(), i)
}

func getJob(group *v1alpha1.Cluster, i int) *v1alpha1.Service {
	var instance v1alpha1.Service

	instance.SetName(generateName(group, i))

	labelling.SetScenario(&instance.ObjectMeta, labelling.GetScenario(group))
	labelling.SetAction(&instance.ObjectMeta, labelling.GetAction(group))
	labelling.SetComponent(&instance.ObjectMeta, labelling.GetComponent(group))

	// modulo is needed to re-iterate the job list, required for the implementation of "Until".
	jobSpec := group.Status.QueuedJobs[i%len(group.Status.QueuedJobs)]

	jobSpec.DeepCopyInto(&instance.Spec)

	return &instance
}

const placementLabel = "placement"

func (r *Controller) constructJobSpecList(ctx context.Context, cluster *v1alpha1.Cluster) ([]v1alpha1.ServiceSpec, error) {
	if err := cluster.Spec.GenerateFromTemplate.Prepare(true); err != nil {
		return nil, errors.Wrapf(err, "template validation")
	}

	specs, err := serviceutils.GetServiceSpecList(ctx, r.GetClient(), cluster, cluster.Spec.GenerateFromTemplate)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get specs")
	}

	/*
		Pod affinity and anti-affinity allows placing pods to nodes as a function of the labels of other pods.
		These Kubernetes features are useful in scenarios like: an application that consists of multiple services,
		some of which may require that they be co-located on the same node for performance reasons; replicas of
		critical services should not be placed onto the same node to avoid loss in the event of node failure.

		See: https://www.cncf.io/blog/2021/07/27/advanced-kubernetes-pod-to-node-scheduling/
	*/

	if placement := cluster.Spec.Placement; placement != nil {
		var affinity corev1.Affinity

		if placement.Nodes != nil { // Match pods to a node
			affinity.NodeAffinity = &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "kubernetes.io/hostname",
									Operator: corev1.NodeSelectorOpIn,
									Values:   placement.Nodes,
								},
							},
						},
					},
				},
			}
		}

		if placement.Collocate { // Place together all the Pods that belong to this cluster
			affinity.PodAffinity = &corev1.PodAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
					{
						LabelSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      placementLabel,
									Operator: metav1.LabelSelectorOperator(corev1.NodeSelectorOpIn),
									Values:   []string{cluster.GetName()},
								},
							},
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			}
		}

		if placement.ConflictsWith != nil { // Stay away from Pods that belong on  other clusters
			affinity.PodAntiAffinity = &corev1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
					{
						LabelSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      placementLabel,
									Operator: metav1.LabelSelectorOperator(corev1.NodeSelectorOpIn),
									Values:   placement.ConflictsWith,
								},
							},
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			}
		}

		// apply affinity rules to all specs
		for i := 0; i < len(specs); i++ {
			specs[i].Decorators.Labels = labels.Merge(specs[i].Decorators.Labels, map[string]string{
				placementLabel: cluster.GetName(),
			})

			specs[i].Affinity = &affinity
		}
	}

	return specs, nil
}
