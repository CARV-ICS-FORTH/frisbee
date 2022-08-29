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
	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	serviceutils "github.com/carv-ics-forth/frisbee/controllers/service/utils"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *Controller) runJob(ctx context.Context, cr *v1alpha1.Cluster, i int) error {
	var job v1alpha1.Service

	// Populate the job
	job.SetName(common.GenerateName(cr, i, cr.Spec.MaxInstances))

	v1alpha1.SetScenarioLabel(&job.ObjectMeta, v1alpha1.GetScenarioLabel(cr))
	v1alpha1.SetComponentLabel(&job.ObjectMeta, v1alpha1.GetComponentLabel(cr))

	// modulo is needed to re-iterate the job list, required for the implementation of "Until".
	jobSpec := cr.Status.QueuedJobs[i%len(cr.Status.QueuedJobs)]

	jobSpec.DeepCopyInto(&job.Spec)

	if err := common.Create(ctx, r, cr, &job); err != nil {
		return err
	}

	r.GetEventRecorderFor(cr.GetName()).Event(cr, corev1.EventTypeNormal, "Scheduled", job.GetName())

	return nil
}

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
									Key:      v1alpha1.LabelAction,
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
									Key:      v1alpha1.LabelAction,
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
			// Apply the current rules.
			specs[i].Affinity = &affinity
		}
	}

	return specs, nil
}
