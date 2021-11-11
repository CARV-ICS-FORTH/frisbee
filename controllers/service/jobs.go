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

	"github.com/davecgh/go-spew/spew"
	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/template/helpers"
	"github.com/fnikolai/frisbee/controllers/utils"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func (r *Controller) runJob(ctx context.Context, cr *v1alpha1.Service) error {
	if cr.Spec.FromTemplate != nil {
		if err := r.populateSpecFromTemplate(ctx, cr); err != nil {
			return errors.Wrapf(err, "spec generation")
		}

		// from now on, we will use the populated fields, not the template.
		cr.Spec.FromTemplate = nil

		if err := utils.Update(ctx, r, cr); err != nil {
			return errors.Wrapf(err, "cannot update populated fields")
		}
	}

	if err := prepareRequirements(ctx, r, cr); err != nil {
		return errors.Wrapf(err, "requirements error")
	}

	pod, err := constructPod(ctx, r, cr)
	if err != nil {
		return err
	}

	discovery := constructDiscoveryService(r, cr, pod)

	if err := utils.Create(ctx, r, cr, discovery); err != nil {
		return errors.Wrapf(err, "cannot create discovery service")
	}

	if err := utils.Create(ctx, r, cr, pod); err != nil {
		return errors.Wrapf(err, "cannot create pod")
	}

	return nil
}

func (r *Controller) populateSpecFromTemplate(ctx context.Context, obj *v1alpha1.Service) error {
	if err := obj.Spec.FromTemplate.Validate(false); err != nil {
		return errors.Wrapf(err, "template validation")
	}

	var genspec thelpers.GenericSpec

	var err error

	ts := thelpers.ParseRef(obj.GetNamespace(), obj.Spec.FromTemplate.TemplateRef)

	if inputs := obj.Spec.FromTemplate.Inputs; inputs != nil {
		lookupCache := make(map[string]v1alpha1.SList)

		genspec, err = thelpers.GetParameterizedSpec(ctx, r, ts, obj.GetNamespace(), inputs[0], lookupCache)
		if err != nil {
			return errors.Wrapf(err, "parameterized spec")
		}
	} else {
		genspec, err = thelpers.GetDefaultSpec(ctx, r, ts)
		if err != nil {
			return errors.Wrapf(err, "default spec")
		}
	}

	spec, err := genspec.ToServiceSpec()
	if err != nil {
		return errors.Wrapf(err, "invalid spec")
	}

	spec.DeepCopyInto(&obj.Spec)

	return nil
}

const (
	RequirePersistentVolumeClaim = "frisbee.io/pvc"
	PersistentVolumeClaimSpec    = "pvc.frisbee.io/spec"
)

func prepareRequirements(ctx context.Context, r *Controller, cr *v1alpha1.Service) error {
	requirements := cr.Spec.Requirements
	if requirements == nil {
		return nil
	}

	// handle persistent volume claims
	volumeName, exists := requirements[RequirePersistentVolumeClaim]
	if !exists {
		return nil
	}

	config, ok := requirements[PersistentVolumeClaimSpec]
	if !ok {
		return errors.New("no PVC config")
	}

	var content map[string]interface{}

	if err := yaml.Unmarshal([]byte(config), &content); err != nil {
		return errors.Wrapf(err, "cannot unmarshal pvc content")
	}

	var pvc unstructured.Unstructured

	pvc.SetUnstructuredContent(map[string]interface{}{
		"spec": content,
	})
	pvc.SetAPIVersion("v1")
	pvc.SetKind("PersistentVolumeClaim")
	pvc.SetName(cr.GetName())

	if err := utils.Create(ctx, r, cr, &pvc); err != nil {
		return errors.Wrapf(err, "cannot create pvc")
	}

	volume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: pvc.GetName(),
				ReadOnly:  false,
			},
		},
	}

	cr.Spec.Volumes = append(cr.Spec.Volumes, volume)

	return nil
}

func constructPod(ctx context.Context, r *Controller, obj *v1alpha1.Service) (*corev1.Pod, error) {
	var pod corev1.Pod

	{ // metadata
		pod.SetName(obj.GetName())

		pod.SetLabels(obj.GetLabels())
		pod.SetAnnotations(obj.GetAnnotations())
	}

	{ // spec
		setPlacement(obj, &pod)

		pod.Spec.RestartPolicy = corev1.RestartPolicyNever
		pod.Spec.Volumes = obj.Spec.Volumes

		pod.Spec.Containers = []corev1.Container{setContainer(obj)}
	}

	{ // deployment
		if err := setAgents(ctx, r, obj, &pod); err != nil {
			return nil, errors.Wrapf(err, "agent deployment error")
		}
	}

	return &pod, nil
}

func setContainer(obj *v1alpha1.Service) corev1.Container {
	spec := obj.Spec

	container := obj.Spec.Container

	// security
	privilege := true

	securityContext := corev1.SecurityContext{
		Capabilities: &corev1.Capabilities{
			Add:  []corev1.Capability{"SYS_ADMIN", "CAP_SYS_RESOURCE", "IPC_LOCK"},
			Drop: nil,
		},
		Privileged: &privilege,
	}

	container.SecurityContext = &securityContext
	container.TTY = true

	// deployment
	if spec.Resources != nil {
		resources := make(map[corev1.ResourceName]resource.Quantity)

		if len(spec.Resources.CPU) > 0 {
			resources[corev1.ResourceCPU] = resource.MustParse(spec.Resources.CPU)
		}

		if len(spec.Resources.Memory) > 0 {
			resources[corev1.ResourceMemory] = resource.MustParse(spec.Resources.Memory)
		}

		container.Resources = corev1.ResourceRequirements{
			Limits:   resources,
			Requests: resources,
		}
	}

	return container
}

func setAgents(ctx context.Context, r *Controller, obj *v1alpha1.Service, pod *corev1.Pod) error {
	spec := obj.Spec

	if spec.Agents == nil {
		return nil
	}

	// import monitoring agents to the service
	for _, ref := range spec.Agents.Telemetry {
		ts := thelpers.ParseRef(obj.GetNamespace(), ref)

		genSpec, err := thelpers.GetDefaultSpec(ctx, r, ts)
		if err != nil {
			return errors.Wrapf(err, "cannot get monitor")
		}

		mon, err := genSpec.ToMonitorSpec()
		if err != nil {
			return errors.Wrapf(err, "cannot get monitor")
		}

		pod.Spec.Volumes = append(pod.Spec.Volumes, mon.Agent.Volumes...)
		pod.Spec.Containers = append(pod.Spec.Containers, mon.Agent.Container)
	}

	return nil
}

func setPlacement(obj *v1alpha1.Service, pod *corev1.Pod) {
	spec := obj.Spec

	// for the moment simply match domain to a specific node. this will change in the future
	if len(spec.Domain) > 0 {
		pod.Spec.Affinity = &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{ // Match pods to a node
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "kubernetes.io/hostname",
									Operator: corev1.NodeSelectorOpIn,
									Values:   spec.Domain,
								},
							},
						},
					},
				},
			},
		}
	}

	/*
	   affinity:
	      podAffinity:
	        requiredDuringSchedulingIgnoredDuringExecution:
	        - labelSelector:
	            matchExpressions:
	            - key: app
	              operator: In
	              values:
	              - local-test-affinity
	          topologyKey: kubernetes.io/hostname

	      podAntiAffinity:
	        requiredDuringSchedulingIgnoredDuringExecution:
	        - labelSelector:
	            matchExpressions:
	            - key: app
	              operator: In
	              values:
	              - local-test-anti-affinity
	          topologyKey: kubernetes.io/hostname
	*/
}

func constructDiscoveryService(r *Controller, obj *v1alpha1.Service, pod *corev1.Pod) *corev1.Service {
	spec := obj.Spec

	// register ports from containers and sidecars
	var allPorts []corev1.ServicePort

	for _, container := range pod.Spec.Containers {
		for _, port := range container.Ports {
			if port.ContainerPort == 0 {
				spew.Dump(spec)

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

	kubeService.SetName(pod.GetName())

	kubeService.Spec.Ports = allPorts
	kubeService.Spec.ClusterIP = clusterIP

	// bind service to the pod
	service2Pod := map[string]string{pod.GetName(): "discover"}

	kubeService.Spec.Selector = service2Pod
	pod.SetLabels(labels.Merge(pod.GetLabels(), service2Pod))

	return &kubeService
}

/*
func (*Controller) setPlacementConstraints(obj *v1alpha1.Service, pod *corev1.Pod) {
	domainLabels := map[string]string{"domain": obj.Spec.Domain}
	obj.SetLabels(labels.Merge(obj.GetLabels(), domainLabels))

	pod.Spec.Affinity = &corev1.Affinity{
		NodeAffinity: nil,
		PodAffinity: &corev1.PodAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{
					Weight: 1,
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: domainLabels,
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
		},
		PodAntiAffinity: nil,
	}
}
*/
