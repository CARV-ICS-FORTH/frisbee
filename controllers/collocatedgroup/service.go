package collocatedgroup

import (
	"context"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/controllers/template/helpers"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// create collocated services consists of two steps.
// first, to create a common pods
// second, populate the pod spec for every  service -- which may involve more than one containers.
func (r *Reconciler) create(ctx context.Context, group *v1alpha1.CollocatedGroup, serviceList v1alpha1.ServiceSpecList) error {
	pod := corev1.Pod{}

	{ // metadata
		pod.SetName(group.GetName())
		pod.SetLabels(group.GetLabels())
		pod.SetAnnotations(group.GetAnnotations())

		if err := common.SetOwner(group, &pod); err != nil {
			return errors.Wrapf(err, "ownership failed")
		}
	}

	{ // spec
		// populate service requirements
		var bundles []bundle

		for _, spec := range serviceList {
			b := bundle{spec: spec}

			if err := r.createServiceBundle(ctx, group, &b); err != nil {
				return errors.Wrapf(err, "cannot create service %s", spec.NamespacedName)
			}

			bundles = append(bundles, b)
		}

		// merge them into a pod
		for _, b := range bundles {
			pod.Spec.Containers = append(pod.Spec.Containers, b.containers...)
			pod.Spec.Volumes = append(pod.Spec.Volumes, b.spec.Volumes...)

			logrus.Warn("required label ", b.labels)

			pod.SetLabels(labels.Merge(pod.GetLabels(), b.labels))
		}

		logrus.Warn("Pod labels ", pod.Labels)
	}

	{ // deploy
		pod.Spec.RestartPolicy = corev1.RestartPolicyNever

		if err := r.Client.Create(ctx, &pod); err != nil {
			return errors.Wrapf(err, "pod error")
		}
	}

	return nil
}

type bundle struct {
	spec *v1alpha1.ServiceSpec

	containers []corev1.Container

	labels map[string]string
}

func (r *Reconciler) createServiceBundle(ctx context.Context, group *v1alpha1.CollocatedGroup, b *bundle) error {
	// name the container after the service. this is to avoid conflicts between containers tha conceptually
	// belong on different services.
	b.spec.Container.Name = b.spec.Name

	{ // populate missing fields in service container
		b.spec.Container.TTY = true

		privilege := true

		b.spec.Container.SecurityContext = &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add:  []corev1.Capability{"SYS_ADMIN"},
				Drop: nil,
			},
			Privileged: &privilege,
		}
		// r.setPlacementConstraints(spec, &pod)

		// a bundle must contain at least the service container
		b.containers = []corev1.Container{b.spec.Container}
	}

	// fix application container
	if err := r.deployAgents(ctx, group, b); err != nil {
		return errors.Wrapf(err, "agent deployment error")
	}

	// fix discovery
	if err := r.deployDiscoveryService(ctx, group, b); err != nil {
		return errors.Wrapf(err, "discovery deployment error")
	}

	return nil
}

func (r *Reconciler) deployAgents(ctx context.Context, group *v1alpha1.CollocatedGroup, b *bundle) error {
	// import monitoring agents to the service
	for _, ref := range b.spec.MonitorTemplateRefs {
		mon, err := helpers.GetMonitorSpec(ctx, helpers.ParseRef(group.GetNamespace(), ref))
		if err != nil {
			return errors.Wrapf(err, "cannot get monitor")
		}

		// prefix the sidecar (e.g, Telegraf) by the name of service it works for.
		mon.Agent.Container.Name = fmt.Sprintf("%s-%s", b.spec.Name, mon.Agent.Container.Name)

		b.spec.Volumes = append(b.spec.Volumes, mon.Agent.Volumes...)
		b.containers = append(b.containers, mon.Agent.Container)
	}

	return nil
}

func (r *Reconciler) deployDiscoveryService(ctx context.Context, group *v1alpha1.CollocatedGroup, b *bundle) error {
	// register ports from containers and containers
	var allPorts []corev1.ServicePort

	for _, container := range b.containers {
		for _, port := range container.Ports {
			if port.ContainerPort == 0 {
				spew.Dump(b.spec)

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
	kubeService.SetName(b.spec.Name)

	if err := common.SetOwner(group, &kubeService); err != nil {
		return errors.Wrapf(err, "ownership failed %s", kubeService.GetName())
	}

	kubeService.Spec.Ports = allPorts
	kubeService.Spec.ClusterIP = clusterIP

	// enable kubeservice to discover the service containers
	b.labels = map[string]string{b.spec.Name: "discover"}

	kubeService.Spec.Selector = b.labels

	if err := r.Client.Create(ctx, &kubeService); err != nil {
		return errors.Wrapf(err, "cannot create discovery service")
	}

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
