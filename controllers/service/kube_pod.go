package service

import (
	"context"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/controllers/common/lifecycle"
	"github.com/fnikolai/frisbee/controllers/common/selector/template"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func convert(obj interface{}) v1alpha1.Lifecycle {
	pod := obj.(*corev1.Pod)

	status := v1alpha1.Lifecycle{}

	switch pod.Status.Phase {
	case corev1.PodPending:
		status.Phase = v1alpha1.PhasePending
		status.StartTime = &metav1.Time{Time: pod.GetCreationTimestamp().Time}

	case corev1.PodRunning:
		status.Phase = v1alpha1.PhaseRunning
		status.StartTime = &metav1.Time{Time: pod.GetCreationTimestamp().Time}

	case corev1.PodSucceeded:
		status.Phase = v1alpha1.PhaseSuccess
		status.EndTime = pod.GetDeletionTimestamp()

	case corev1.PodFailed:
		status.Phase = v1alpha1.PhaseFailed
		status.EndTime = pod.GetDeletionTimestamp()

	case corev1.PodUnknown:
		status.Phase = v1alpha1.PhaseFailed
	}

	return status
}

func (r *Reconciler) createKubePod(ctx context.Context, obj *v1alpha1.Service) error {
	// populate missing fields in service container
	obj.Spec.Container.TTY = true

	privilege := true

	obj.Spec.Container.SecurityContext = &corev1.SecurityContext{
		Capabilities: &corev1.Capabilities{
			Add:  []corev1.Capability{"SYS_ADMIN"},
			Drop: nil,
		},
		Privileged: &privilege,
	}

	pod := corev1.Pod{}

	// set pod medata
	if err := common.SetOwner(obj, &pod, ""); err != nil {
		return errors.Wrapf(err, "ownership failed")
	}

	pod.SetLabels(obj.GetLabels())
	pod.SetAnnotations(obj.GetAnnotations())

	// set pod spec
	pod.Spec.Volumes = obj.Spec.Volumes
	pod.Spec.Containers = []corev1.Container{obj.Spec.Container}
	pod.Spec.RestartPolicy = corev1.RestartPolicyNever

	// add operational steps. order of operation is important. do not change it.
	{
		if err := r.setMonitoringAgents(ctx, obj, &pod); err != nil {
			return errors.Wrapf(err, "monitoring error")
		}

		if err := r.setDiscovery(ctx, obj, &pod); err != nil {
			return errors.Wrapf(err, "discovery error")
		}

		r.setPlacementConstraints(obj, &pod)
	}

	if err := r.Client.Create(ctx, &pod); err != nil {
		return errors.Wrapf(err, "pod error")
	}

	// convert external pod to inner object so to gain management of lifecycle
	err := lifecycle.New(ctx,
		lifecycle.WatchExternal(&pod, convert, pod.GetName()),
		lifecycle.WithFilter(lifecycle.FilterParent(obj.GetUID())),
		lifecycle.WithAnnotator(&lifecycle.PointAnnotation{}), // Register event to grafana
		lifecycle.WithLogger(r.Logger),
	).Update(obj)

	return errors.Wrapf(err, "lifecycle error")
}

func (r *Reconciler) setMonitoringAgents(ctx context.Context, obj *v1alpha1.Service, pod *corev1.Pod) error {
	// import monitoring agents to the service
	for _, ref := range obj.Spec.MonitorTemplateRefs {
		mon := template.SelectMonitor(ctx, template.ParseRef(obj.GetNamespace(), ref))

		// TODO: validate that agent has no fields set other than Volume and Container

		if mon == nil {
			return errors.New("nil monitor")
		}
		/*
			if len(spec.MonitorTemplateRef) > 0 ||
				len(spec.Volumes) > 0 ||
				len(spec.Domain) > 0 ||
				spec.Resources != nil {
				return errors.Errorf("sidecar service must have only the container specified")
			}
		*/

		pod.Spec.Volumes = append(pod.Spec.Volumes, mon.Agent.Volumes...)
		pod.Spec.Containers = append(pod.Spec.Containers, mon.Agent.Container)
	}

	return nil
}

func (r *Reconciler) setDiscovery(ctx context.Context, obj *v1alpha1.Service, pod *corev1.Pod) error {
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

	if err := common.SetOwner(obj, &kubeService, ""); err != nil {
		return errors.Wrapf(err, "ownership failed %s", pod.GetName())
	}

	kubeService.Spec.Ports = allPorts
	kubeService.Spec.ClusterIP = clusterIP

	// enable kubeservice to discover kubepod
	service2Pod := map[string]string{"discover": pod.GetName()}
	pod.SetLabels(labels.Merge(pod.GetLabels(), service2Pod))

	kubeService.Spec.Selector = pod.GetLabels()

	if err := r.Client.Create(ctx, &kubeService); err != nil {
		return errors.Wrapf(err, "cannot create kubernetes service")
	}

	return nil
}

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
