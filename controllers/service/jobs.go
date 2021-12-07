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

package service

import (
	"context"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/utils"
	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
)

func (r *Controller) runJob(ctx context.Context, cr *v1alpha1.Service) error {
	if err := prepareRequirements(ctx, r, cr); err != nil {
		return errors.Wrapf(err, "requirements error")
	}

	if err := decoratePod(ctx, r, cr); err != nil {
		return errors.Wrapf(err, "decorator error")
	}

	discovery, discoveryLabels := constructDiscoveryService(cr)

	if err := utils.Create(ctx, r, cr, discovery); err != nil {
		return errors.Wrapf(err, "cannot create discovery service")
	}

	// finally, create the pod
	var pod corev1.Pod

	pod.SetName(cr.GetName())
	pod.SetAnnotations(cr.GetAnnotations())
	pod.SetLabels(labels.Merge(cr.GetLabels(), discoveryLabels))
	cr.Spec.PodSpec.DeepCopyInto(&pod.Spec)

	if err := utils.Create(ctx, r, cr, &pod); err != nil {
		return errors.Wrapf(err, "cannot create pod")
	}

	return nil
}

func prepareRequirements(ctx context.Context, r *Controller, cr *v1alpha1.Service) error {
	if cr.Spec.Requirements == nil {
		return nil
	}

	// Volume
	if req := cr.Spec.Requirements.PVC; req != nil {
		var pvc corev1.PersistentVolumeClaim

		pvc.SetName(cr.GetName())
		req.Spec.DeepCopyInto(&pvc.Spec)

		if err := utils.Create(ctx, r, cr, &pvc); err != nil {
			return errors.Wrapf(err, "cannot create pvc")
		}

		// auto-mount the created pvc.
		volume := corev1.Volume{
			Name: req.Name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.GetName(),
					ReadOnly:  false,
				},
			},
		}

		cr.Spec.Volumes = append(cr.Spec.Volumes, volume)
	}

	return nil
}

func decoratePod(ctx context.Context, r *Controller, cr *v1alpha1.Service) error {
	if cr.Spec.Decorators == nil {
		return nil
	}

	// set placement policies
	if req := cr.Spec.Decorators.Placement; req != nil {
		// for the moment simply match domain to a specific node. this will change in the future
		if len(req.Domain) > 0 {
			cr.Spec.Affinity = &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{ // Match pods to a node
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      "kubernetes.io/hostname",
										Operator: corev1.NodeSelectorOpIn,
										Values:   req.Domain,
									},
								},
							},
						},
					}, // Equally, for podAntiAffinity
				},
			}
		}
	}

	// set resources, to the first container only
	if req := cr.Spec.Decorators.Resources; req != nil {
		if len(cr.Spec.Containers) != 1 {
			return errors.New("Decoration resources are not applicable for multiple containers")
		}

		resources := make(map[corev1.ResourceName]resource.Quantity)

		if len(req.CPU) > 0 {
			resources[corev1.ResourceCPU] = resource.MustParse(req.CPU)
		}

		if len(req.Memory) > 0 {
			resources[corev1.ResourceMemory] = resource.MustParse(req.Memory)
		}

		cr.Spec.Containers[0].Resources = corev1.ResourceRequirements{
			Limits:   resources,
			Requests: resources,
		}
	}

	// import telemetry agents
	if req := cr.Spec.Decorators.Telemetry; req != nil {
		// import monitoring agents to the service
		for _, monRef := range req {
			monSpec, err := r.serviceControl.GetServiceSpec(ctx, cr.GetNamespace(), v1alpha1.GenerateFromTemplate{TemplateRef: monRef})
			if err != nil {
				return errors.Wrapf(err, "cannot get monitor")
			}

			if len(monSpec.Containers) != 1 {
				return errors.Wrapf(err, "invalid agent %s", monRef)
			}

			cr.Spec.Containers = append(cr.Spec.Containers, monSpec.Containers[0])
			cr.Spec.Volumes = append(cr.Spec.Volumes, monSpec.Volumes...)
			cr.Spec.Volumes = append(cr.Spec.Volumes, monSpec.Volumes...)
		}
	}

	// set default values
	cr.Spec.RestartPolicy = corev1.RestartPolicyNever

	return nil
}

func constructDiscoveryService(cr *v1alpha1.Service) (*corev1.Service, labels.Set) {
	// register ports from containers and sidecars
	var allPorts []corev1.ServicePort

	for _, container := range cr.Spec.Containers {
		for _, port := range container.Ports {
			if port.ContainerPort == 0 {
				spew.Dump(cr.Spec)

				panic("invalid port")
			}

			allPorts = append(allPorts, corev1.ServicePort{
				Name: port.Name,
				Port: port.ContainerPort,
			})
		}
	}

	// clusterIP should be specified only with ports
	var clusterIP string

	if len(allPorts) == 0 {
		clusterIP = "None"
	}

	kubeService := corev1.Service{}

	kubeService.SetName(cr.GetName())

	kubeService.Spec.Ports = allPorts
	kubeService.Spec.ClusterIP = clusterIP

	// bind service to the pod
	service2Pod := map[string]string{cr.GetName(): "discover"}

	kubeService.Spec.Selector = service2Pod

	return &kubeService, service2Pod
}
