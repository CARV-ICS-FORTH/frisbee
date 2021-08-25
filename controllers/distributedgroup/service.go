package distributedgroup

import (
	"context"

	"github.com/davecgh/go-spew/spew"
	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/controllers/template/helpers"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func (r *Reconciler) createService(ctx context.Context, group *v1alpha1.DistributedGroup, spec *v1alpha1.ServiceSpec) error {
	pod := corev1.Pod{}

	{ // metadata
		pod.SetName(spec.NamespacedName.Name)
		pod.SetLabels(group.GetLabels())
		pod.SetAnnotations(group.GetAnnotations())

		if err := common.SetOwner(group, &pod); err != nil {
			return errors.Wrapf(err, "ownership failed")
		}
	}

	{ // spec
		// populate missing fields in service container
		spec.Container.TTY = true

		privilege := true

		spec.Container.SecurityContext = &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add:  []corev1.Capability{"SYS_ADMIN"},
				Drop: nil,
			},
			Privileged: &privilege,
		}

		pod.Spec.Containers = []corev1.Container{spec.Container}

		pod.Spec.Volumes = spec.Volumes
		pod.Spec.RestartPolicy = corev1.RestartPolicyNever

		// r.setPlacementConstraints(spec, &pod)
	}

	{ // deployment
		if err := r.deployAgents(ctx, group, spec, &pod); err != nil {
			return errors.Wrapf(err, "agent deployment error")
		}

		if err := r.deployDiscoveryService(ctx, group, spec, &pod); err != nil {
			return errors.Wrapf(err, "discovery deployment error")
		}

		if err := r.Client.Create(ctx, &pod); err != nil {
			return errors.Wrapf(err, "pod error")
		}
	}

	return nil
}

func (r *Reconciler) deployAgents(ctx context.Context, group *v1alpha1.DistributedGroup, spec *v1alpha1.ServiceSpec, pod *corev1.Pod) error {
	if spec.Agents == nil {
		return nil
	}

	// import monitoring agents to the service
	for _, ref := range spec.Agents.Telemetry {
		mon, err := helpers.GetMonitorSpec(ctx, helpers.ParseRef(group.GetNamespace(), ref))
		if err != nil {
			return errors.Wrapf(err, "cannot get monitor")
		}

		pod.Spec.Volumes = append(pod.Spec.Volumes, mon.Agent.Volumes...)
		pod.Spec.Containers = append(pod.Spec.Containers, mon.Agent.Container)
	}

	return nil
}

func (r *Reconciler) deployDiscoveryService(ctx context.Context, group *v1alpha1.DistributedGroup, spec *v1alpha1.ServiceSpec, pod *corev1.Pod) error {
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

	if err := common.SetOwner(group, &kubeService); err != nil {
		return errors.Wrapf(err, "ownership failed %s", pod.GetName())
	}

	kubeService.Spec.Ports = allPorts
	kubeService.Spec.ClusterIP = clusterIP

	// enable kubeservice to discover kubepod
	service2Pod := map[string]string{pod.GetName(): "discover"}

	kubeService.Spec.Selector = service2Pod

	if err := r.Client.Create(ctx, &kubeService); err != nil {
		return errors.Wrapf(err, "cannot create discovery service")
	}

	pod.SetLabels(labels.Merge(pod.GetLabels(), service2Pod))

	return nil
}

/*
func (*Reconciler) setPlacementConstraints(obj *v1alpha1.Service, pod *corev1.Pod) {
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
