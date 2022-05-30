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
	"fmt"
	"path/filepath"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/utils"
	"github.com/carv-ics-forth/frisbee/controllers/utils/configuration"
	"github.com/dustinkirkland/golang-petname"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// {{{ Internal types

const (
	// grafana specific.
	grafanaDashboards = "/etc/grafana/provisioning/dashboards"
)

const (
	// Prometheus should be a fixed name because it is used within the Grafana configuration.
	// Otherwise, we should find a way to replace the value.
	notRandomPrometheusName = "prometheus"

	notRandomGrafanaName = "grafanaaaa"
)

func (r *Controller) prepareEnvironment(ctx context.Context, t *v1alpha1.Telemetry) error {
	/*
		For the telemetry we must differentiate between the installation namespace (which holds the templates and
		configuration files for Prometheus, Grafana, and agents, and the testing namespace on which the respective
		objects will be created.

		Importantly, the configurations cannot be shared among different namespaces. Therefore, the only way
		is to copy them from the installation namespace to the testing namespace.

		The same is for ingress. Every different testplan must have its own ingress that points to the create services.
	*/
	installationNamespace := configuration.Global.Namespace

	copyConfigMap := func(name string) error {
		var config corev1.ConfigMap

		key := client.ObjectKey{
			Namespace: installationNamespace,
			Name:      name,
		}

		if err := r.GetClient().Get(ctx, key, &config); err != nil {
			return errors.Wrapf(err, "cannot get config '%s'", name)
		}

		config.SetResourceVersion("")
		config.SetNamespace(t.GetNamespace())

		if err := utils.Create(ctx, r, t, &config); err != nil {
			return errors.Wrapf(err, "cannot create config '%s'", name)
		}

		return nil
	}

	if err := copyConfigMap(configuration.PrometheusConfig); err != nil {
		return errors.Wrapf(err, "cannot copy config '%s' from '%s' to '%s'", configuration.PrometheusConfig,
			installationNamespace, t.GetNamespace())
	}

	if err := copyConfigMap(configuration.GrafanaConfig); err != nil {
		return errors.Wrapf(err, "cannot copy config '%s' from '%s' to '%s'", configuration.GrafanaConfig,
			installationNamespace, t.GetNamespace())
	}

	if err := copyConfigMap(configuration.AgentConfig); err != nil {
		return errors.Wrapf(err, "cannot copy config '%s' from '%s' to '%s'", configuration.AgentConfig,
			installationNamespace, t.GetNamespace())
	}

	return nil
}

func createEndpoint(name string, port int64) string {
	/* If the operator runs within the cluster, it will reach Grafana via the service */

	/* If in developer mode, the operator runs outside the cluster, and will reach Grafana via the ingress */
	if configuration.Global.DeveloperMode {
		return fmt.Sprintf("http://%s:%d", name, port)
	}

	// If the operator runs outside the cluster, it will reach Grafana via the ingress.
	return fmt.Sprintf("http://%s.%s", name, configuration.Global.DomainName)
}

func (r *Controller) installPrometheus(ctx context.Context, t *v1alpha1.Telemetry) error {
	installationNamespace := configuration.Global.Namespace

	var prometheus v1alpha1.Service

	prometheus.SetName(notRandomPrometheusName)
	prometheus.SetNamespace(t.GetNamespace())

	{ // spec
		fromtemplate := &v1alpha1.GenerateFromTemplate{
			TemplateRef:  configuration.PrometheusTemplate,
			MaxInstances: 1,
			Inputs:       nil,
		}

		if err := fromtemplate.Prepare(false); err != nil {
			return errors.Wrapf(err, "template validation")
		}

		spec, err := r.serviceControl.GetServiceSpec(ctx, installationNamespace, *fromtemplate)
		if err != nil {
			return errors.Wrapf(err, "cannot get spec")
		}

		spec.DeepCopyInto(&prometheus.Spec)
	}

	if err := utils.Create(ctx, r, t, &prometheus); err != nil {
		return errors.Wrapf(err, "cannot create %s", prometheus.GetName())
	}

	t.Status.PrometheusEndpoint = createEndpoint(prometheus.GetName(), configuration.Global.PrometheusPort)

	return nil
}

func (r *Controller) installGrafana(ctx context.Context, t *v1alpha1.Telemetry) error {
	installationNamespace := configuration.Global.Namespace

	var grafana v1alpha1.Service

	grafana.SetName(notRandomGrafanaName)
	grafana.SetNamespace(t.GetNamespace())

	{ // spec
		fromtemplate := &v1alpha1.GenerateFromTemplate{
			TemplateRef:  configuration.GrafanaTemplate,
			MaxInstances: 1,
			Inputs:       nil,
		}

		if err := fromtemplate.Prepare(false); err != nil {
			return errors.Wrapf(err, "template validation")
		}

		spec, err := r.serviceControl.GetServiceSpec(ctx, installationNamespace, *fromtemplate)
		if err != nil {
			return errors.Wrapf(err, "cannot get spec")
		}

		spec.DeepCopyInto(&grafana.Spec)

		if err := r.importDashboards(ctx, t, &grafana.Spec); err != nil {
			return errors.Wrapf(err, "import dashboards")
		}
	}

	if err := utils.Create(ctx, r, t, &grafana); err != nil {
		return errors.Wrapf(err, "cannot create %s", grafana.GetName())
	}

	t.Status.GrafanaEndpoint = createEndpoint(grafana.GetName(), configuration.Global.GrafanaPort)
	return nil
}

func (r *Controller) importDashboards(ctx context.Context, t *v1alpha1.Telemetry, spec *v1alpha1.ServiceSpec) error {
	imported := make(map[string]struct{})

	var namespace string

	// iterate monitoring services
	for _, agentRef := range t.Spec.Import {
		// search for the agent
		if agentRef == configuration.AgentTemplate {
			namespace = configuration.Global.Namespace
		} else {
			namespace = t.GetNamespace()
		}

		// the configuration name is expected to be in the form `agent.config'

		var dashboards corev1.ConfigMap

		key := client.ObjectKey{
			Namespace: namespace,
			Name:      agentRef + ".config",
		}

		if err := r.GetClient().Get(ctx, key, &dashboards); err != nil {
			return errors.Wrapf(err, "cannot find configmap with dashboards '%s'", key)
		}

		// avoid duplicates that may be caused when multiple agents share the same dashboard
		{
			_, exists := imported[dashboards.GetName()]
			if exists {
				continue
			}

			imported[dashboards.GetName()] = struct{}{}
		}

		volumeName := petname.Name()

		// associate volume to grafana
		spec.Volumes = append(spec.Volumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: dashboards.GetName()},
				},
			},
		})

		// mount the volume
		for file := range dashboards.Data {
			r.Logger.Info("Import",
				"configMap", dashboards.GetName(),
				"file", file)

			spec.Containers[0].VolumeMounts = append(spec.Containers[0].VolumeMounts, corev1.VolumeMount{
				Name:      volumeName, // Name of a Volume.
				ReadOnly:  true,
				MountPath: filepath.Join(grafanaDashboards, file), // Path within the container
				SubPath:   file,                                   //  Path within the volume
			})
		}
	}

	return nil
}

func (r *Controller) createIngress(ctx context.Context, t *v1alpha1.Telemetry) error {
	ingressClassName := configuration.Global.IngressClassName
	pathType := netv1.PathTypePrefix

	var ingress netv1.Ingress

	ingress.SetName("ingress")
	ingress.SetNamespace(t.GetNamespace())

	ingress.Spec = netv1.IngressSpec{
		IngressClassName: &ingressClassName,
		Rules: []netv1.IngressRule{
			{
				Host: fmt.Sprintf("%s.%s", notRandomPrometheusName, configuration.Global.DomainName),
				IngressRuleValue: netv1.IngressRuleValue{
					HTTP: &netv1.HTTPIngressRuleValue{
						Paths: []netv1.HTTPIngressPath{
							{
								Path:     "/",
								PathType: &pathType,
								Backend: netv1.IngressBackend{
									Service: &netv1.IngressServiceBackend{
										Name: notRandomPrometheusName,
										Port: netv1.ServiceBackendPort{
											Name: "http",
										},
									},
								},
							},
						},
					},
				},
			},
			{
				Host: fmt.Sprintf("%s.%s", notRandomGrafanaName, configuration.Global.DomainName),
				IngressRuleValue: netv1.IngressRuleValue{
					HTTP: &netv1.HTTPIngressRuleValue{
						Paths: []netv1.HTTPIngressPath{
							{
								Path:     "/",
								PathType: &pathType,
								Backend: netv1.IngressBackend{
									Service: &netv1.IngressServiceBackend{
										Name: notRandomGrafanaName,
										Port: netv1.ServiceBackendPort{
											Name: "http",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := utils.Create(ctx, r, t, &ingress); err != nil {
		return errors.Wrapf(err, "cannot create ingress")
	}

	return nil
}

/*
   - host: logviewer-frisbee.{{.Values.global.domainName}}
     http:
       paths:
         - path: /
           pathType: Prefix
           backend:
             service:
               name: logviewer
               port:
                 number: 80

   {{- if .Values.chaos.enabled }}
   - host: chaos-frisbee.{{.Values.global.domainName}}
     http:
       paths:
         - path: /
           pathType: Prefix
           backend:
             service:
               name: chaos-dashboard
               port:
                 number: 2333
   {{- end}}
*/
