package service

import (
	"context"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/pkg/structure"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *Reconciler) createKubePod(ctx context.Context, owner *v1alpha1.Service, volumes []corev1.Volume, mounts []corev1.VolumeMount) error {
	containers := createContainers(owner, mounts)

	placement := func(spec v1alpha1.ServiceSpec) *corev1.Affinity {
		if len(spec.Domain) > 0 {
			domainLabels := map[string]string{"domain": spec.Domain}
			owner.SetLabels(structure.MergeMap(owner.GetLabels(), domainLabels))

			return &corev1.Affinity{
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

		return nil
	}

	pod := corev1.Pod{}
	pod.SetNamespace(owner.GetNamespace())
	pod.SetName(owner.GetName() + "-pod")

	pod.Spec.Volumes = volumes
	pod.Spec.RestartPolicy = corev1.RestartPolicyNever
	pod.Spec.Containers = containers
	pod.Spec.Affinity = placement(owner.Spec)

	if err := common.SetOwner(owner, &pod); err != nil {
		return errors.Wrapf(err, "unable to set owner for pod %s", pod.GetName())
	}

	if err := r.makeDiscoverable(ctx, owner, &pod); err != nil {
		return errors.Wrapf(err, "unable to make pod %s discoverable", pod.GetName())
	}

	if err := r.Client.Create(ctx, &pod); err != nil {
		return errors.Wrapf(err, "unable to create pod %s", pod.GetName())
	}

	common.UpdateLifecycle(ctx, owner, &common.ExternalToInnerObject{Object: &pod, StatusFunc: convert}, pod.GetName())

	return nil
}

func convert(obj interface{}) v1alpha1.EtherStatus {
	pod := obj.(*corev1.Pod)

	status := v1alpha1.EtherStatus{}

	switch pod.Status.Phase {
	case corev1.PodPending:
		status.Phase = v1alpha1.Uninitialized

	case corev1.PodRunning:
		status.Phase = v1alpha1.Running
		status.StartTime = pod.GetDeletionTimestamp()

	case corev1.PodSucceeded:
		status.Phase = v1alpha1.Complete
		status.StartTime = pod.GetDeletionTimestamp()

	case corev1.PodFailed:
		status.Phase = v1alpha1.Failed
		status.StartTime = pod.GetDeletionTimestamp()

	case corev1.PodUnknown:
		status.Phase = v1alpha1.Failed
	}

	return status
}

func createContainers(obj *v1alpha1.Service, volumemounts []corev1.VolumeMount) []corev1.Container {
	resources := func(resources *v1alpha1.Resources) corev1.ResourceRequirements {
		if resources == nil {
			return corev1.ResourceRequirements{}
		}

		list := corev1.ResourceList{}

		// native to kubernetes
		if len(resources.CPU) != 0 {
			list[corev1.ResourceCPU] = resource.MustParse(resources.CPU)
		}

		if len(resources.Memory) != 0 {
			list[corev1.ResourceMemory] = resource.MustParse(resources.Memory)
		}

		return corev1.ResourceRequirements{
			Limits:   list,
			Requests: list,
		}
	}

	ports := func(ports []v1alpha1.Port) []corev1.ContainerPort {
		asPorts := make([]corev1.ContainerPort, len(ports))
		for i := 0; i < len(ports); i++ {
			asPorts[i] = corev1.ContainerPort{
				Name:          ports[i].Name,
				ContainerPort: ports[i].Port,
			}
		}

		return asPorts
	}

	spec := obj.Spec

	// If true, it leads to admission webhook error in chaos-mesh
	privilege := false

	// TODO: do it for one, and then for sidecars
	container := corev1.Container{
		Name:         obj.GetName(),
		Image:        spec.Image,
		Command:      spec.Command,
		Args:         spec.Args,
		Env:          spec.Env,
		Resources:    resources(spec.Resources),
		VolumeMounts: volumemounts,
		Ports:        ports(spec.Ports),
		TTY:          true,
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add:  []corev1.Capability{"SYS_ADMIN"},
				Drop: nil,
			},
			Privileged: &privilege,
		},
	}

	return []corev1.Container{container}
}

func (r *Reconciler) makeDiscoverable(ctx context.Context, owner *v1alpha1.Service, pod *corev1.Pod) error {

	// register ports from containers and sidecars
	var allPorts []corev1.ServicePort

	for _, container := range pod.Spec.Containers {
		for _, port := range container.Ports {
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
	kubeService.Namespace = owner.GetNamespace()
	kubeService.Name = owner.GetName()

	kubeService.Spec.Ports = allPorts
	kubeService.Spec.ClusterIP = clusterIP

	if err := common.SetOwner(owner, &kubeService); err != nil {
		return errors.Wrap(err, "unable to set owner")
	}

	// add discovery labels
	discoverylabels := map[string]string{"discover": pod.GetName()}
	pod.SetLabels(structure.MergeMap(pod.GetLabels(), discoverylabels))

	kubeService.Spec.Selector = pod.GetLabels()

	return r.Client.Create(ctx, &kubeService)
}
