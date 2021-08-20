package workflow

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/controllers/common/lifecycle"
	"github.com/fnikolai/frisbee/controllers/common/selector/template"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
)

// {{{ Internal types

const (
	// grafana specific.
	grafanaDashboards  = "/etc/grafana/provisioning/dashboards"
	prometheusTemplate = "observability/prometheus"
	grafanaTemplate    = "observability/grafana"
)

func (r *Reconciler) newMonitoringStack(ctx context.Context, obj *v1alpha1.Workflow) error {
	if len(obj.Spec.ImportMonitors) == 0 {
		return nil
	}

	prometheus, err := r.installPrometheus(ctx, obj)
	if err != nil {
		return errors.Wrapf(err, "prometheus error")
	}

	grafana, err := r.installGrafana(ctx, obj)
	if err != nil {
		return errors.Wrapf(err, "grafana error")
	}

	// Make Prometheus and Grafana accessible from outside the Cluster
	if len(obj.Spec.Ingress) > 0 {
		if err := r.installIngress(ctx, obj, prometheus, grafana); err != nil {
			return errors.Wrapf(err, "ingress error")
		}

		// use the public Grafana address (via Ingress) because the controller runs outside of the cluster
		grafanaPublicURI := fmt.Sprintf("http://%s", virtualhost(grafana.GetName(), obj.Spec.Ingress))

		if err := common.EnableGrafanaAnnotator(ctx, grafanaPublicURI); err != nil {
			return errors.Wrapf(err, "grafana client error")
		}
	}

	r.Logger.Info("Monitoring stack is ready")

	return nil
}

func (r *Reconciler) installPrometheus(ctx context.Context, obj *v1alpha1.Workflow) (*v1alpha1.DistributedGroup, error) {
	prom := v1alpha1.DistributedGroup{}

	{ // metadata
		prom.SetName("prometheus")
		if err := common.SetOwner(obj, &prom); err != nil {
			return nil, errors.Wrapf(err, "ownership error")
		}
	}

	{ // spec
		prom.Spec.Instances = 1
		prom.Spec.TemplateRef = prometheusTemplate
	}

	{ // deployment
		if err := r.Create(ctx, &prom); err != nil {
			return nil, errors.Wrapf(err, "request error")
		}

		if err := lifecycle.WaitRunningAndUpdate(ctx, &prom); err != nil {
			return nil, errors.Wrapf(err, "prometheus is not running")
		}
	}

	r.Logger.Info("Prometheus is installed")

	return &prom, nil
}

func (r *Reconciler) installGrafana(ctx context.Context, obj *v1alpha1.Workflow) (*v1alpha1.DistributedGroup, error) {
	grafana := v1alpha1.DistributedGroup{}

	{ // metadata
		grafana.SetName("grafana")
		if err := common.SetOwner(obj, &grafana); err != nil {
			return nil, errors.Wrapf(err, "ownership error")
		}
	}

	{ // spec
		grafana.Spec.Instances = 1

		// to perform the necessary automations, we load the spec locally and push the modified version for creation.
		grafana.Spec.ServiceSpec = template.SelectService(ctx, template.ParseRef(obj.GetNamespace(), grafanaTemplate))
		if grafana.Spec.ServiceSpec == nil {
			return nil, errors.New("cannot find template")
		}

		if err := r.importDashboards(ctx, obj, grafana.Spec.ServiceSpec); err != nil {
			return nil, errors.Wrapf(err, "import dashboards")
		}
	}

	{ // deployment
		if err := r.Client.Create(ctx, &grafana); err != nil {
			return nil, errors.Wrapf(err, "request error")
		}

		if err := lifecycle.WaitRunningAndUpdate(ctx, &grafana); err != nil {
			return nil, errors.Wrapf(err, "grafana is not running")
		}
	}

	r.Logger.Info("Grafana is installed")

	return &grafana, nil
}

func (r *Reconciler) importDashboards(ctx context.Context, obj *v1alpha1.Workflow, spec *v1alpha1.ServiceSpec) error {
	// create configmap
	configMap := corev1.ConfigMap{}
	configMap.Name = "dashboards"

	configMap.Data = make(map[string]string, len(obj.Spec.ImportMonitors))
	{
		for _, monRef := range obj.Spec.ImportMonitors {
			monSpec := template.SelectMonitor(ctx, template.ParseRef(obj.GetNamespace(), monRef))
			if monSpec == nil {
				return errors.Errorf("monitor failed")
			}

			configMap.Data[monSpec.Dashboard.File] = monSpec.Dashboard.Payload
		}

		if err := common.SetOwner(obj, &configMap); err != nil {
			return errors.Wrapf(err, "ownership failed")
		}

		if err := r.Client.Create(ctx, &configMap); err != nil {
			return errors.Wrapf(err, "configmap failed")
		}
	}

	// create volume from the configmap
	volume := corev1.Volume{}
	volumeMounts := make([]corev1.VolumeMount, 0, len(configMap.Data))
	{
		volume.Name = "grafana-dashboards"
		volume.VolumeSource = corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: configMap.GetName()},
			},
		}

		for file := range configMap.Data {
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      volume.Name, // Name of a Volume.
				ReadOnly:  true,
				MountPath: filepath.Join(grafanaDashboards, file), // Path within the container
				SubPath:   file,                                   //  Path within the volume
			})
		}
	}

	// associate volume with grafana
	spec.Volumes = append(spec.Volumes, volume)
	spec.Container.VolumeMounts = append(spec.Container.VolumeMounts, volumeMounts...)

	r.Logger.Info("Import Grafana packages", "dashboards", obj.Spec.ImportMonitors)

	return nil
}

func (r *Reconciler) installIngress(ctx context.Context, obj *v1alpha1.Workflow, groups ...*v1alpha1.DistributedGroup) error {
	ingress := netv1.Ingress{}

	{ // metadata
		ingress.SetName("ingress")

		// Ingresses annotated with 'kubernetes.io/ingress.class=ambassador' will be managed by Ambassador.
		// Without annotation, the default Ingress is used.
		ingress.SetAnnotations(map[string]string{
			"kubernetes.io/ingress.class": "ambassador",
		})

		if err := common.SetOwner(obj, &ingress); err != nil {
			return errors.Wrapf(err, "ownership failed")
		}
	}

	{ // spec
		pathtype := netv1.PathTypePrefix

		rules := make([]netv1.IngressRule, 0, len(groups))

		for _, group := range groups {
			service := group.Status.ExpectedServices[0]

			// we now that prometheus and grafana have a single container
			port := service.Container.Ports[0]

			rule := netv1.IngressRule{
				Host: virtualhost(service.Name, obj.Spec.Ingress),
				IngressRuleValue: netv1.IngressRuleValue{
					HTTP: &netv1.HTTPIngressRuleValue{
						Paths: []netv1.HTTPIngressPath{
							{
								Path:     "/",
								PathType: &pathtype,
								Backend: netv1.IngressBackend{
									Service: &netv1.IngressServiceBackend{
										Name: service.Name,
										Port: netv1.ServiceBackendPort{Number: port.ContainerPort},
									},
								},
							},
						},
					},
				},
			}

			rules = append(rules, rule)

			r.Logger.Info("Ingress", "host", rule.Host)
		}

		ingress.Spec.Rules = rules
	}

	{ // deployment
		if err := r.Client.Create(ctx, &ingress); err != nil {
			return errors.Wrapf(err, "unable to create ingress")
		}
	}

	return nil
}

func virtualhost(serviceName, ingress string) string {
	return fmt.Sprintf("%s.%s", serviceName, ingress)
}
