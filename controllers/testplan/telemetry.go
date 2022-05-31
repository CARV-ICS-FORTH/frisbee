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

package testplan

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/testplan/grafana"
	"github.com/carv-ics-forth/frisbee/controllers/utils"
	"github.com/carv-ics-forth/frisbee/controllers/utils/configuration"
	"github.com/carv-ics-forth/frisbee/controllers/utils/expressions"
	"github.com/carv-ics-forth/frisbee/pkg/netutils"
	"github.com/dustinkirkland/golang-petname"
	notifier "github.com/golanghelper/grafana-webhook"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

	notRandomGrafanaName = "grafana"

	notRandomLogViewerName = "logviewer"
)

var pathType = netv1.PathTypePrefix

func (r *Controller) StartTelemetry(ctx context.Context, t *v1alpha1.TestPlan) error {
	telemetryAgents, err := r.ImportTelemetryDashboards(ctx, t)
	if err != nil {
		return errors.Wrapf(err, "errors with importing dashboards")
	}

	if len(telemetryAgents) == 0 { // there is no need to import the stack of the is no dashboard.
		return nil
	}

	if err := r.copyEnvironment(ctx, t); err != nil {
		return errors.Wrapf(err, "environment error")
	}

	if err := r.installPrometheus(ctx, t); err != nil {
		return errors.Wrapf(err, "prometheus error")
	}

	if err := r.installGrafana(ctx, t, telemetryAgents); err != nil {
		return errors.Wrapf(err, "grafana error")
	}

	if err := r.createIngress(ctx, t); err != nil {
		return errors.Wrapf(err, "ingress error")
	}

	return nil
}

func (r *Controller) copyEnvironment(ctx context.Context, t *v1alpha1.TestPlan) error {
	/*
		For the telemetry we must differentiate between the installation namespace (which holds the templates and
		configuration files for Prometheus, Grafana, and agents, and the testing namespace on which the respective
		objects will be created.

		Importantly, the configurations cannot be shared among different namespaces. Therefore, the only way
		is to copy them from the installation namespace to the testing namespace.

		The same is for ingress. Every different testplan must have its own ingress that points to the create services.
	*/
	installationNamespace := configuration.Global.Namespace

	// nothing to do. the test runs on the default installation namespace.
	if t.GetNamespace() == installationNamespace {
		return nil
	}

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

func (r *Controller) installPrometheus(ctx context.Context, t *v1alpha1.TestPlan) error {
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

	t.Status.PrometheusEndpoint = utils.GenerateEndpoint(notRandomPrometheusName, t.GetName(), configuration.Global.PrometheusPort)

	return nil
}

func (r *Controller) installGrafana(ctx context.Context, t *v1alpha1.TestPlan, telemetryAgents []string) error {
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

		if err := r.importDashboards(ctx, t, &grafana.Spec, telemetryAgents); err != nil {
			return errors.Wrapf(err, "import telemetryAgents")
		}
	}

	if err := utils.Create(ctx, r, t, &grafana); err != nil {
		return errors.Wrapf(err, "cannot create %s", grafana.GetName())
	}

	t.Status.GrafanaEndpoint = utils.GenerateEndpoint(notRandomGrafanaName, t.GetName(), configuration.Global.GrafanaPort)

	return nil
}

func (r *Controller) importDashboards(ctx context.Context, t *v1alpha1.TestPlan, spec *v1alpha1.ServiceSpec, telemetryAgents []string) error {
	imported := make(map[string]struct{})

	var namespace string

	// iterate monitoring services
	for _, agentRef := range telemetryAgents {
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
			return errors.Wrapf(err, "cannot find configmap with telemetryAgents '%s'", key)
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

func (r *Controller) createIngress(ctx context.Context, t *v1alpha1.TestPlan) error {
	ingressClassName := configuration.Global.IngressClassName

	var ingress netv1.Ingress

	ingress.SetName("ingress")
	ingress.SetNamespace(t.GetNamespace())

	ingress.Spec = netv1.IngressSpec{
		IngressClassName: &ingressClassName,
		Rules: []netv1.IngressRule{
			{
				Host: t.Status.PrometheusEndpoint,
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
				Host: t.Status.GrafanaEndpoint,
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

			{ // Create a placeholder for the logviewer.
				Host: utils.GenerateEndpoint(notRandomLogViewerName, t.GetName(), configuration.Global.LogviewerPort),
				IngressRuleValue: netv1.IngressRuleValue{
					HTTP: &netv1.HTTPIngressRuleValue{
						Paths: []netv1.HTTPIngressPath{
							{
								Path:     "/",
								PathType: &pathType,
								Backend: netv1.IngressBackend{
									Service: &netv1.IngressServiceBackend{
										Name: notRandomLogViewerName,
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

// ImportTelemetryDashboards iterates the referenced services (directly via Service or indirectly via Cluster) and list
// all telemetry dashboards that need to be imported
func (r *Controller) ImportTelemetryDashboards(ctx context.Context, plan *v1alpha1.TestPlan) ([]string, error) {
	dedup := make(map[string]struct{})

	var fromTemplate *v1alpha1.GenerateFromTemplate

	for _, action := range plan.Spec.Actions {
		fromTemplate = nil

		switch action.ActionType {
		case v1alpha1.ActionService:
			fromTemplate = action.Service
		case v1alpha1.ActionCluster:
			fromTemplate = &action.Cluster.GenerateFromTemplate
		default:
			continue
		}

		spec, err := r.serviceControl.GetServiceSpec(ctx, plan.GetNamespace(), *fromTemplate)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot retrieve service spec")
		}

		// firstly store everything on a map to avoid duplicates
		if spec.Decorators != nil {
			for _, dashboard := range spec.Decorators.Telemetry {
				dedup[dashboard] = struct{}{}
			}
		}
	}

	// secondly, return a deduped array
	imports := make([]string, 0, len(dedup))
	for dashboard, _ := range dedup {
		imports = append(imports, dashboard)
	}

	return imports, nil
}

func (r *Controller) ConnectToGrafana(ctx context.Context, t *v1alpha1.TestPlan) error {
	if grafana.ClientExistsFor(t) {
		return nil
	}

	return grafana.New(ctx,
		grafana.WithHTTP(t.Status.GrafanaEndpoint), // Connect to ...
		grafana.WithRegisterFor(t),                 // Used by grafana.GetClient(), grafana.ClientExistsFor(), ...
		grafana.WithLogger(r),                      // Log info
		grafana.WithNotifications(WebhookURL),
	)
}

var WebhookURL string

var WebhookPort = "6666"

// CreateWebhookServer  creates a Webhook for listening for events from Grafana *
func (r *Controller) CreateWebhookServer(ctx context.Context) error {
	webhook := http.DefaultServeMux

	webhook.Handle("/", notifier.HandleWebhook(func(w http.ResponseWriter, b *notifier.Body) {
		r.Logger.Info("Grafana Alert", "body", b)

		if err := expressions.DispatchAlert(context.Background(), r, b); err != nil {
			r.Logger.Error(err, "unable to process metrics alert", b)
		}
	}, 0))

	// If the controller runs within the Kubernetes cluster, we use the assigned name as the advertised host
	// If the controller runs externally to the Kubernetes cluster, we use the public IP of the local machine.
	if configuration.Global.AdvertisedHost == "" {
		ip, err := netutils.GetPublicIP()
		if err != nil {
			return errors.Wrapf(err, "cannot get controller's public ip")
		}

		WebhookURL = fmt.Sprintf("http://%s:%s", ip.String(), WebhookPort)
	} else {
		WebhookURL = fmt.Sprintf("http://%s:%s", configuration.Global.AdvertisedHost, WebhookPort)
	}

	logrus.Warn("START WEBHOOK AT ", WebhookURL)

	go http.ListenAndServe(":"+WebhookPort, webhook)

	return nil
}
