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

package telemetry

import (
	"context"
	"path/filepath"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/utils"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// {{{ Internal types

const (
	// grafana specific.
	grafanaDashboards  = "/etc/grafana/provisioning/dashboards"
	prometheusTemplate = "prometheus"
	grafanaTemplate    = "grafana"
)

func (r *Controller) installPrometheus(ctx context.Context, w *v1alpha1.Telemetry) error {
	var prom v1alpha1.Service

	{ // metadata
		prom.SetName("prometheus")
		prom.SetNamespace(w.GetNamespace())
	}

	{ // spec
		fromtemplate := &v1alpha1.GenerateFromTemplate{
			TemplateRef: prometheusTemplate,
			Instances:   0,
			Inputs:      nil,
		}

		if err := fromtemplate.Validate(false); err != nil {
			return errors.Wrapf(err, "template validation")
		}

		spec, err := r.serviceControl.GetServiceSpec(ctx, w.GetNamespace(), *fromtemplate)
		if err != nil {
			return errors.Wrapf(err, "cannot get spec")
		}

		spec.DeepCopyInto(&prom.Spec)
	}

	return utils.Create(ctx, r, w, &prom)
}

func (r *Controller) installGrafana(ctx context.Context, w *v1alpha1.Telemetry) error {
	var grafana v1alpha1.Service

	{ // metadata
		grafana.SetName("grafana")
		grafana.SetNamespace(w.GetNamespace())
	}

	{ // spec
		fromtemplate := &v1alpha1.GenerateFromTemplate{
			TemplateRef: grafanaTemplate,
			Instances:   0,
			Inputs:      nil,
		}

		if err := fromtemplate.Validate(false); err != nil {
			return errors.Wrapf(err, "template validation")
		}

		spec, err := r.serviceControl.GetServiceSpec(ctx, w.GetNamespace(), *fromtemplate)
		if err != nil {
			return errors.Wrapf(err, "cannot get spec")
		}

		spec.DeepCopyInto(&grafana.Spec)

		if err := r.importDashboards(ctx, w, &grafana.Spec); err != nil {
			return errors.Wrapf(err, "import dashboards")
		}
	}

	return utils.Create(ctx, r, w, &grafana)
}

func (r *Controller) importDashboards(ctx context.Context, obj *v1alpha1.Telemetry, spec *v1alpha1.ServiceSpec) error {
	imported := make(map[string]struct{})

	// iterate monitoring services
	for _, monRef := range obj.Spec.ImportMonitors {
		monSpec, err := r.serviceControl.GetServiceSpec(ctx, obj.GetNamespace(), v1alpha1.GenerateFromTemplate{TemplateRef: monRef})
		if err != nil {
			return errors.Wrapf(err, "cannot get spec for %s", monRef)
		}

		var configMaps corev1.ConfigMapList

		selector, err := metav1.LabelSelectorAsSelector(&monSpec.Decorators.Dashboards)
		if err != nil {
			return errors.Wrapf(err, "invalid dashboard definition")
		}

		if err := r.GetClient().List(ctx, &configMaps, &client.ListOptions{LabelSelector: selector}); err != nil {
			return errors.Wrapf(err, "cannot find dashboards")
		}

		for _, configMap := range configMaps.Items {
			// avoid duplicates that may be caused when multiple agents share the same dashboard
			_, exists := imported[configMap.GetName()]
			if exists {
				continue
			}

			imported[configMap.GetName()] = struct{}{}

			// associate volume to grafana
			spec.Volumes = append(spec.Volumes, corev1.Volume{
				Name: configMap.GetName(),
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{Name: configMap.GetName()},
					},
				},
			})

			for file := range configMap.Data {
				r.Logger.Info("Import",
					"configMap", configMap.GetName(),
					"file", file)

				spec.Containers[0].VolumeMounts = append(spec.Containers[0].VolumeMounts, corev1.VolumeMount{
					Name:      configMap.GetName(), // Name of a Volume.
					ReadOnly:  true,
					MountPath: filepath.Join(grafanaDashboards, file), // Path within the container
					SubPath:   file,                                   //  Path within the volume
				})
			}
		}
	}

	return nil
}

/*
func (r *Controller) installIngress(ctx context.Context, cr *v1alpha1.Telemetry, prometheus, grafana *v1alpha1.Service) error {
	ingress := netv1.Ingress{}

	{ // metadata
		ingress.SetName("frisbee")

		if cr.Spec.Ingress.UseAmbassador {
			ingress.SetAnnotations(map[string]string{
				"kubernetes.io/ingress.class": "ambassador",
			})
		}
	}

	{ // spec
		pathtype := netv1.PathTypePrefix

		ingress.Spec.Rules = make([]netv1.IngressRule, 2)

		// prometheus
		ingress.Spec.Rules[0] = netv1.IngressRule{
			Host: cr.Spec.Ingress.DNSPrefix.Convert(prometheus.GetName()),
			IngressRuleValue: netv1.IngressRuleValue{
				HTTP: &netv1.HTTPIngressRuleValue{
					Paths: []netv1.HTTPIngressPath{
						{
							Path:     "/",
							PathType: &pathtype,
							Backend: netv1.IngressBackend{
								Service: &netv1.IngressServiceBackend{
									Name: prometheus.GetName(),
									Port: netv1.ServiceBackendPort{Number: prometheus.Spec.Container.Ports[0].ContainerPort},
								},
							},
						},
					},
				},
			},
		}

		// grafana
		ingress.Spec.Rules[1] = netv1.IngressRule{
			Host: cr.Spec.Ingress.DNSPrefix.Convert(grafana.GetName()),
			IngressRuleValue: netv1.IngressRuleValue{
				HTTP: &netv1.HTTPIngressRuleValue{
					Paths: []netv1.HTTPIngressPath{
						{
							Path:     "/",
							PathType: &pathtype,
							Backend: netv1.IngressBackend{
								Service: &netv1.IngressServiceBackend{
									Name: grafana.GetName(),
									Port: netv1.ServiceBackendPort{Number: grafana.Spec.Container.Ports[0].ContainerPort},
								},
							},
						},
					},
				},
			},
		}
	}

	{ // deployment
		if err := utils.Create(ctx, r, cr, &ingress); err != nil {
			return errors.Wrapf(err, "unable to create ingress")
		}
	}

	cr.Status.PrometheusURI = fmt.Sprintf("http://%s", cr.Spec.Ingress.DNSPrefix.Convert(prometheus.GetName()))

	cr.Status.GrafanaURI = fmt.Sprintf("http://%s", cr.Spec.Ingress.DNSPrefix.Convert(grafana.GetName()))

	return nil
}

*/
