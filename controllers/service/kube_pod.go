package service

import (
	"context"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/controllers/common/selector/template"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

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

func (r *Reconciler) createKubePod(ctx context.Context, obj *v1alpha1.Service) error {
	pod := corev1.Pod{}
	pod.Spec.Volumes = obj.Spec.Volumes

	// If true, it leads to admission webhook error in chaos-mesh
	privilege := false

	pod.Spec.RestartPolicy = corev1.RestartPolicyNever
	obj.Spec.Container.TTY = true
	obj.Spec.Container.SecurityContext = &corev1.SecurityContext{
		Capabilities: &corev1.Capabilities{
			Add:  []corev1.Capability{"SYS_ADMIN"},
			Drop: nil,
		},
		Privileged: &privilege,
	}
	pod.Spec.Containers = []corev1.Container{obj.Spec.Container}

	if err := common.SetOwner(obj, &pod, ""); err != nil {
		return errors.Wrapf(err, "ownership failed %s", obj.GetName())
	}

	// order of operation is important. do not change it.
	if err := r.addMonitoring(ctx, obj, &pod); err != nil {
		return errors.Wrapf(err, "unable to make pod %s discoverable", pod.GetName())
	}

	if err := r.makeDiscoverable(ctx, obj, &pod); err != nil {
		return errors.Wrapf(err, "unable to make pod %s discoverable", pod.GetName())
	}

	r.placement(obj, &pod)

	if err := r.Client.Create(ctx, &pod); err != nil {
		return errors.Wrapf(err, "unable to create pod %s", pod.GetName())
	}

	// because lifecycle operation require access to the status, we need to wrap externally managed
	// objects like Pods into managed (inner) objects
	podWraper := &common.ExternalToInnerObject{Object: &pod, StatusFunc: convert}

	// wait until the pod is up and running. this step is necessary to obtain the ip.
	if err := common.WaitLifecycle(ctx, obj.GetUID(), podWraper, v1alpha1.Running, pod.GetName()); err != nil {
		return errors.Wrapf(err, "pod is not running")
	}

	// TODO: NEED TO FIND A WAY FOR GETTING THE IP

	obj.Status.IP = pod.Status.PodIP

	// continuously update the service with pod's phase
	if err := common.UpdateLifecycle(ctx, obj, podWraper, pod.GetName()); err != nil {
		return errors.Wrapf(err, "cannot update lifecycle for %s", pod.GetName())
	}

	return nil
}

func (r *Reconciler) addMonitoring(ctx context.Context, obj *v1alpha1.Service, pod *corev1.Pod) error {
	// import monitoring agents to the service
	for _, ref := range obj.Spec.MonitorTemplateRefs {
		mon := template.SelectMonitor(ctx, template.ParseRef(ref))

		if err := validateMonitor(mon); err != nil {
			return err
		}

		pod.Spec.Volumes = append(pod.Spec.Volumes, mon.Agent.Volumes...)
		pod.Spec.Containers = append(pod.Spec.Containers, mon.Agent.Container)
	}

	return nil
}

func validateMonitor(spec *v1alpha1.MonitorSpec) error {
	if spec == nil {
		return errors.New("nil monitor is not allowed")
	}

	// TODO: validate that agent has no fields set other than Volume and Container
	/*
		if len(spec.MonitorTemplateRef) > 0 ||
			len(spec.Volumes) > 0 ||
			len(spec.Domain) > 0 ||
			spec.Resources != nil {
			return errors.Errorf("sidecar service must have only the container specified")
		}


	*/
	return nil
}

func (r *Reconciler) makeDiscoverable(ctx context.Context, obj *v1alpha1.Service, pod *corev1.Pod) error {
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

	return r.Client.Create(ctx, &kubeService)
}

func (*Reconciler) placement(obj *v1alpha1.Service, pod *corev1.Pod) {
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

/*
func createContainers(obj *v1alpha1.Reference, volumemounts []corev1.VolumeMount) []corev1.Container {
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

	// If true, it leads to admission webhook error in chaos-mesh
	privilege := false


	for i:=0; i < len(obj.Spec.Containers); i++ {
		cont := &obj.Spec.Containers[i]

		// cont.Name = obj.GetName()
		cont.Resources = resources(obj.Spec.Resources)
		cont.VolumeMounts =  volumemounts
		cont.TTY = true
		cont.SecurityContext = &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add:  []corev1.Capability{"SYS_ADMIN"},
				Drop: nil,
			},
			Privileged: &privilege,
		}
	}


	return []corev1.Container{container}
}

*/
