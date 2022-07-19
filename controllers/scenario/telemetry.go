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

package scenario

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/carv-ics-forth/frisbee/controllers/common/configuration"
	"github.com/carv-ics-forth/frisbee/controllers/common/expressions"
	"github.com/carv-ics-forth/frisbee/controllers/common/grafana"
	"github.com/carv-ics-forth/frisbee/controllers/common/labelling"
	serviceutils "github.com/carv-ics-forth/frisbee/controllers/service/utils"
	notifier "github.com/golanghelper/grafana-webhook"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
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
)

var (
	GrafanaPort = int64(3000)
)

func (r *Controller) StartTelemetry(ctx context.Context, t *v1alpha1.Scenario) error {
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

	if err := r.connectToGrafana(ctx, t); err != nil {
		return errors.Wrapf(err, "cannot communicate with the telemetry stack")
	}

	return nil
}

// StopTelemetry removes the annotations from the target object, removes the Alert from Grafana, and deleted the
// client for the specific scenario.
func (r *Controller) StopTelemetry(t *v1alpha1.Scenario) {
	// If the resource is not initialized, then there is not registered telemetry client.
	if meta.IsStatusConditionTrue(t.Status.Conditions, v1alpha1.ConditionCRInitialized.String()) {
		grafana.DeleteClientFor(t)
	}
}

func (r *Controller) installPrometheus(ctx context.Context, t *v1alpha1.Scenario) error {
	var job v1alpha1.Service

	job.SetName(notRandomPrometheusName)

	// set labels
	labelling.SetScenario(&job.ObjectMeta, t.GetName())
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

		spec, err := serviceutils.GetServiceSpec(ctx, r.GetClient(), t, *fromtemplate)
		if err != nil {
			return errors.Wrapf(err, "cannot get spec")
		}

		spec.DeepCopyInto(&job.Spec)
	}

	if err := common.Create(ctx, r, t, &job); err != nil {
		return errors.Wrapf(err, "cannot create %s", job.GetName())
	}

	t.Status.PrometheusEndpoint = common.ExternalEndpoint(notRandomPrometheusName, t.GetNamespace())

	return nil
}

func (r *Controller) installGrafana(ctx context.Context, t *v1alpha1.Scenario, agentRefs []string) error {
	var job v1alpha1.Service

	job.SetName(notRandomGrafanaName)

	labelling.SetScenario(&job.ObjectMeta, t.GetName())
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

		spec, err := serviceutils.GetServiceSpec(ctx, r.GetClient(), t, *fromtemplate)
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

	t.Status.GrafanaEndpoint = common.ExternalEndpoint(notRandomGrafanaName, t.GetNamespace())

	return nil
}

func (r *Controller) importDashboards(ctx context.Context, t *v1alpha1.Scenario, spec *v1alpha1.ServiceSpec, telemetryAgents []string) error {
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
			if _, exists := imported[dashboards.GetName()]; exists {
				continue
			}

			imported[dashboards.GetName()] = struct{}{}
		}

		volumeName := fmt.Sprintf("vol-%d", len(spec.Volumes))

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

// ImportTelemetryDashboards iterates the referenced services (directly via Service or indirectly via Cluster) and list
// all telemetry dashboards that need to be imported
func (r *Controller) ImportTelemetryDashboards(ctx context.Context, scenario *v1alpha1.Scenario) ([]string, error) {
	dedup := make(map[string]struct{})

	var fromTemplate *v1alpha1.GenerateFromTemplate

	for _, action := range scenario.Spec.Actions {
		fromTemplate = nil

		switch action.ActionType {
		case v1alpha1.ActionService:
			fromTemplate = action.Service
		case v1alpha1.ActionCluster:
			fromTemplate = &action.Cluster.GenerateFromTemplate
		default:
			continue
		}

		spec, err := serviceutils.GetServiceSpec(ctx, r.GetClient(), scenario, *fromTemplate)
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

	// secondly, return a de-duplicated array
	imports := make([]string, 0, len(dedup))
	for dashboard := range dedup {
		imports = append(imports, dashboard)
	}

	return imports, nil
}

// connectToGrafana creates a dedicated link between the scenario controller and the Grafana service.
// The link must be destroyed if the scenario is deleted, since any new instance will change the ip of Grafana.
func (r *Controller) connectToGrafana(ctx context.Context, t *v1alpha1.Scenario) error {
	if t.Status.GrafanaEndpoint == "" {
		r.Logger.Info("The Grafana endpoint is empty. Skip telemetry.", "scenario", t.GetName())
		return nil
	}

	if grafana.ClientExistsFor(t) {
		return nil
	}

	var endpoint string

	if configuration.Global.DeveloperMode {
		/* If in developer mode, the operator runs outside the cluster, and will reach Grafana via the ingress */
		endpoint = common.ExternalEndpoint(notRandomGrafanaName, t.GetNamespace())
	} else {
		/* If the operator runs within the cluster, it will reach Grafana via the service */
		endpoint = common.InternalEndpoint(notRandomGrafanaName, t.GetNamespace(), GrafanaPort)
	}

	return grafana.New(ctx,
		grafana.WithHTTP(endpoint), // Connect to ...
		grafana.WithRegisterFor(t), // Used by grafana.GetClient(), grafana.ClientExistsFor(), ...
		grafana.WithLogger(r),      // Log info
		grafana.WithNotifications(WebhookURL),
	)
}

var gracefulShutDown = 30 * time.Second

var WebhookURL string

var startWebhookOnce sync.Once

const alertingWebhook = "alerting-service"

// CreateWebhookServer  creates a Webhook for listening for events from Grafana *
func (r *Controller) CreateWebhookServer(ctx context.Context, alertingPort int) error {
	WebhookURL = fmt.Sprintf("http://%s:%d", alertingWebhook, alertingPort)

	r.Logger.Info("Controller Webhook", "URL", WebhookURL)

	webhook := http.DefaultServeMux

	webhook.Handle("/", notifier.HandleWebhook(func(w http.ResponseWriter, b *notifier.Body) {
		if err := expressions.DispatchAlert(ctx, r, b); err != nil {
			r.Logger.Error(err, "Drop alert", "body", b)
		}
	}, 0))

	// Start the server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", alertingPort),
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
