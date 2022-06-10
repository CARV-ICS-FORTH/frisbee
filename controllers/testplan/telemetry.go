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
	"time"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/carv-ics-forth/frisbee/controllers/common/configuration"
	"github.com/carv-ics-forth/frisbee/controllers/common/expressions"
	"github.com/carv-ics-forth/frisbee/controllers/common/grafana"
	"github.com/carv-ics-forth/frisbee/controllers/common/labelling"
	serviceutils "github.com/carv-ics-forth/frisbee/controllers/service/utils"
	"github.com/carv-ics-forth/frisbee/pkg/netutils"
	"github.com/dustinkirkland/golang-petname"
	notifier "github.com/golanghelper/grafana-webhook"
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

	notRandomGrafanaName = "grafana"

	notRandomLogViewerName = "logviewer"
)

var (
	PrometheusPort = int64(9090)

	GrafanaPort = int64(3000)

	LogviewerPort = int64(80)
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

func (r *Controller) installPrometheus(ctx context.Context, t *v1alpha1.TestPlan) error {
	var job v1alpha1.Service

	job.SetName(notRandomPrometheusName)

	// set labels
	labelling.SetPlan(&job.ObjectMeta, t.GetName())
	labelling.SetAction(&job.ObjectMeta, job.GetName())
	labelling.SetComponent(&job.ObjectMeta, labelling.ComponentSys)

	{ // spec
		fromtemplate := &v1alpha1.GenerateFromTemplate{
			TemplateRef:  configuration.PrometheusTemplate,
			MaxInstances: 1,
			Inputs:       nil,
		}

		if err := fromtemplate.Prepare(false); err != nil {
			return errors.Wrapf(err, "template validation")
		}

		spec, err := serviceutils.GetServiceSpec(ctx, r, t, *fromtemplate)
		if err != nil {
			return errors.Wrapf(err, "cannot get spec")
		}

		spec.DeepCopyInto(&job.Spec)
	}

	if err := common.Create(ctx, r, t, &job); err != nil {
		return errors.Wrapf(err, "cannot create %s", job.GetName())
	}

	t.Status.PrometheusEndpoint = common.GenerateEndpoint(notRandomPrometheusName, t.GetNamespace(), PrometheusPort)

	return nil
}

func (r *Controller) installGrafana(ctx context.Context, t *v1alpha1.TestPlan, agentRefs []string) error {
	var job v1alpha1.Service

	job.SetName(notRandomGrafanaName)

	labelling.SetPlan(&job.ObjectMeta, t.GetName())
	labelling.SetAction(&job.ObjectMeta, job.GetName())
	labelling.SetComponent(&job.ObjectMeta, labelling.ComponentSys)

	{ // spec
		fromtemplate := &v1alpha1.GenerateFromTemplate{
			TemplateRef:  configuration.GrafanaTemplate,
			MaxInstances: 1,
			Inputs:       nil,
		}

		if err := fromtemplate.Prepare(false); err != nil {
			return errors.Wrapf(err, "template validation")
		}

		spec, err := serviceutils.GetServiceSpec(ctx, r, t, *fromtemplate)
		if err != nil {
			return errors.Wrapf(err, "cannot get spec")
		}

		spec.DeepCopyInto(&job.Spec)

		if err := r.importDashboards(ctx, t, &job.Spec, agentRefs); err != nil {
			return errors.Wrapf(err, "import dashboards")
		}
	}

	if err := common.Create(ctx, r, t, &job); err != nil {
		return errors.Wrapf(err, "cannot create %s", job.GetName())
	}

	t.Status.GrafanaEndpoint = common.GenerateEndpoint(notRandomGrafanaName, t.GetNamespace(), GrafanaPort)

	return nil
}

func (r *Controller) importDashboards(ctx context.Context, t *v1alpha1.TestPlan, spec *v1alpha1.ServiceSpec, telemetryAgents []string) error {
	imported := make(map[string]struct{})

	// iterate monitoring services
	for _, agentRef := range telemetryAgents {

		var dashboards corev1.ConfigMap

		key := client.ObjectKey{
			Namespace: t.GetNamespace(),
			Name:      agentRef + ".config",
		}

		if err := r.GetClient().Get(ctx, key, &dashboards); err != nil {
			return errors.Wrapf(err, "cannot find configmap with telemetry agents '%s'", key)
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

	ingress.SetName(t.GetName())

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
				Host: common.GenerateEndpoint(notRandomLogViewerName, t.GetNamespace(), LogviewerPort),
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

	if err := common.Create(ctx, r, t, &ingress); err != nil {
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

		spec, err := serviceutils.GetServiceSpec(ctx, r, plan, *fromTemplate)
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

var gracefulShutDown = 30 * time.Second

var WebhookURL string

// CreateWebhookServer  creates a Webhook for listening for events from Grafana *
func (r *Controller) CreateWebhookServer(ctx context.Context) error {
	webhook := http.DefaultServeMux

	webhook.Handle("/", notifier.HandleWebhook(func(w http.ResponseWriter, b *notifier.Body) {
		r.Logger.Info("Grafana Alert", "body", b)

		if err := expressions.DispatchAlert(ctx, r, b); err != nil {
			r.Logger.Error(err, "unable to process metrics alert", b)
		}
	}, 0))

	// If the controller runs within the Kubernetes cluster, we use the assigned name as the advertised host
	// If the controller runs externally to the Kubernetes cluster, we use the public IP of the local machine.
	if configuration.Global.DeveloperMode {
		ip, err := netutils.GetPublicIP()
		if err != nil {
			return errors.Wrapf(err, "cannot get controller's public ip")
		}

		WebhookURL = fmt.Sprintf("http://%s:%d", ip.String(), configuration.Global.WebhookPort)
	} else {
		WebhookURL = fmt.Sprintf("http://%s:%d", configuration.Global.ControllerName, configuration.Global.WebhookPort)
	}

	// Start the server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", configuration.Global.WebhookPort),
		Handler: webhook,
	}

	idleConnsClosed := make(chan error)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			idleConnsClosed <- err
		}
	}()

	go func() {
		select {
		case <-ctx.Done():
			r.Logger.Info("Shutdown signal received, waiting for webhook server to finish")

		case err := <-idleConnsClosed:
			r.Logger.Error(err, "Error received. Shutting down the webhook server")
		}

		// need a new background context for the graceful shutdown. the ctx is already cancelled.
		gracefulTimeout, cancel := context.WithTimeout(context.Background(), gracefulShutDown)
		defer cancel()

		if err := srv.Shutdown(gracefulTimeout); err != nil {
			r.Logger.Error(err, "error shutting down the webhook server")
		}
		close(idleConnsClosed)
	}()

	return nil
}
