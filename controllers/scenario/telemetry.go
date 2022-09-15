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

	"github.com/carv-ics-forth/frisbee/pkg/configuration"
	"github.com/carv-ics-forth/frisbee/pkg/expressions"
	"github.com/carv-ics-forth/frisbee/pkg/grafana"
	"github.com/carv-ics-forth/frisbee/pkg/structure"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
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
	defaultPathToGrafanaDashboards = "/etc/grafana/provisioning/dashboards"
)

const (
	// Prometheus should be a fixed name because it is used within the Grafana configuration.
	// Otherwise, we should find a way to replace the value.
	defaultPrometheusName = "prometheus"

	defaultGrafanaName = "grafana"

	defaultDataviewerName = "dataviewer"
)

var (
	GrafanaPort = int64(3000)
)

func (r *Controller) StartTelemetry(ctx context.Context, t *v1alpha1.Scenario) error {
	// the filebrowser makes sense only if test data are enabled.
	if t.Spec.TestData != nil {
		if err := r.installDataviewer(ctx, t); err != nil {
			return errors.Wrapf(err, "cannot provision testdata")
		}
	}

	// there is no need to import the stack of the is no dashboard.
	telemetryAgents, err := r.ListTelemetryAgents(ctx, t)
	if err != nil {
		return errors.Wrapf(err, "importing dashboards")
	}

	if len(telemetryAgents) > 0 {
		if err := r.installPrometheus(ctx, t); err != nil {
			return errors.Wrapf(err, "prometheus error")
		}

		if err := r.installGrafana(ctx, t, telemetryAgents); err != nil {
			return errors.Wrapf(err, "grafana error")
		}
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

func (r *Controller) installDataviewer(ctx context.Context, t *v1alpha1.Scenario) error {
	// Ensure the claim exists, and we do not wait indefinitely.
	if t.Spec.TestData != nil {
		claimName := t.Spec.TestData.Claim.ClaimName
		var claim corev1.PersistentVolumeClaim

		key := client.ObjectKey{Namespace: t.GetNamespace(), Name: claimName}

		if err := r.GetClient().Get(ctx, key, &claim); err != nil {
			return errors.Wrapf(err, "cannot verify existence of testdata claim '%s'", claimName)
		}
	}

	// Now we can use it to create the data viewer
	var job v1alpha1.Service

	job.SetName(defaultDataviewerName)

	// set labels
	v1alpha1.SetScenarioLabel(&job.ObjectMeta, t.GetName())
	v1alpha1.SetComponentLabel(&job.ObjectMeta, v1alpha1.ComponentSys)

	{ // spec
		spec, err := serviceutils.GetServiceSpec(ctx, r.GetClient(), t, v1alpha1.GenerateObjectFromTemplate{
			TemplateRef:  configuration.DataviewerTemplate,
			MaxInstances: 1,
			Inputs:       nil,
		})

		if err != nil {
			return errors.Wrapf(err, "cannot get spec")
		}

		spec.DeepCopyInto(&job.Spec)

		// the dataviewer is the only service that has complete access to the volume's content.
		job.AttachTestDataVolume(t.Spec.TestData, false)
	}

	if err := common.Create(ctx, r, t, &job); err != nil {
		return errors.Wrapf(err, "cannot create %s", job.GetName())
	}

	t.Status.DataviewerEndpoint = common.ExternalEndpoint(defaultDataviewerName, t.GetNamespace())

	return nil
}

func (r *Controller) installPrometheus(ctx context.Context, t *v1alpha1.Scenario) error {
	var job v1alpha1.Service

	job.SetName(defaultPrometheusName)

	// set labels
	v1alpha1.SetScenarioLabel(&job.ObjectMeta, t.GetName())
	v1alpha1.SetComponentLabel(&job.ObjectMeta, v1alpha1.ComponentSys)

	{ // spec
		spec, err := serviceutils.GetServiceSpec(ctx, r.GetClient(), t, v1alpha1.GenerateObjectFromTemplate{
			TemplateRef:  configuration.PrometheusTemplate,
			MaxInstances: 1,
			Inputs:       nil,
		})

		if err != nil {
			return errors.Wrapf(err, "cannot get spec")
		}

		spec.DeepCopyInto(&job.Spec)

		// NOTICE: Prometheus does not support NFS or other distributed filesystems. It returns
		// panic: Unable to create mmap-ed active query log
		// We have this line here commented, just to make the point of **DO NOT UNCOMMENT IT**.
		// job.AttachTestDataVolume(t.Spec.TestData, true)
	}

	if err := common.Create(ctx, r, t, &job); err != nil {
		return errors.Wrapf(err, "cannot create %s", job.GetName())
	}

	t.Status.PrometheusEndpoint = common.ExternalEndpoint(defaultPrometheusName, t.GetNamespace())

	return nil
}

func (r *Controller) installGrafana(ctx context.Context, t *v1alpha1.Scenario, agentRefs []string) error {
	var job v1alpha1.Service

	job.SetName(defaultGrafanaName)

	v1alpha1.SetScenarioLabel(&job.ObjectMeta, t.GetName())
	v1alpha1.SetComponentLabel(&job.ObjectMeta, v1alpha1.ComponentSys)

	{ // spec
		spec, err := serviceutils.GetServiceSpec(ctx, r.GetClient(), t, v1alpha1.GenerateObjectFromTemplate{
			TemplateRef:  configuration.GrafanaTemplate,
			MaxInstances: 1,
			Inputs:       nil,
		})

		if err != nil {
			return errors.Wrapf(err, "cannot get spec")
		}

		spec.DeepCopyInto(&job.Spec)

		job.AttachTestDataVolume(t.Spec.TestData, true)

		if err := r.importDashboards(ctx, t, &job.Spec, agentRefs); err != nil {
			return errors.Wrapf(err, "import dashboards")
		}
	}

	if err := common.Create(ctx, r, t, &job); err != nil {
		return errors.Wrapf(err, "cannot create %s", job.GetName())
	}

	t.Status.GrafanaEndpoint = common.ExternalEndpoint(defaultGrafanaName, t.GetNamespace())

	return nil
}

func (r *Controller) importDashboards(ctx context.Context, t *v1alpha1.Scenario, spec *v1alpha1.ServiceSpec, telemetryAgents []string) error {
	imported := make(map[string]struct{})

	for _, agentRef := range telemetryAgents {
		// Every Telemetry agent must be accompanied by a configMap that contains the visualization dashboards.
		// The dashboards are expected to be named {{.TelemetryAgentName}}.config
		var dashboards corev1.ConfigMap
		{
			key := client.ObjectKey{
				Namespace: t.GetNamespace(),
				Name:      agentRef + ".config",
			}

			if err := r.GetClient().Get(ctx, key, &dashboards); err != nil {
				return errors.Wrapf(err, "configmap '%s' is missing", key)
			}

			// avoid duplicates that may be caused when multiple agents share the same dashboard
			if _, exists := imported[dashboards.GetName()]; exists {
				continue
			}

			imported[dashboards.GetName()] = struct{}{}
		}

		// The  visualizations Dashboards should be loaded to Grafana.
		{
			// create a Pod volume from the config map
			volumeName := fmt.Sprintf("vol-%d", len(spec.Volumes))
			spec.Volumes = append(spec.Volumes, corev1.Volume{
				Name: volumeName,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{Name: dashboards.GetName()},
					},
				},
			})

			// mount the Pod volume to the main Grafana container.
			if len(spec.Containers) != 1 {
				return errors.Errorf("Grafana expected a single '%s' but found '%d' containers",
					v1alpha1.MainContainerName, len(spec.Containers))
			}
			mainContainer := &spec.Containers[0]

			for file := range dashboards.Data {
				mainContainer.VolumeMounts = append(mainContainer.VolumeMounts, corev1.VolumeMount{
					Name:      volumeName, // Name of a Volume.
					ReadOnly:  true,
					MountPath: filepath.Join(defaultPathToGrafanaDashboards, file), // Path within the container
					SubPath:   file,                                                //  Path within the volume
				})

				r.Logger.Info("LoadDashboard", "obj", client.ObjectKeyFromObject(&dashboards), "file", file)
			}
		}
	}

	return nil
}

// ListTelemetryAgents iterates the referenced services (directly via Service or indirectly via Cluster) and list
// all telemetry dashboards that need to be imported
func (r *Controller) ListTelemetryAgents(ctx context.Context, scenario *v1alpha1.Scenario) ([]string, error) {
	dedup := make(map[string]struct{})

	for _, action := range scenario.Spec.Actions {
		var fromTemplate *v1alpha1.GenerateObjectFromTemplate

		// only Services and Clusters may container Telemetry Agents.
		switch action.ActionType {
		case v1alpha1.ActionService:
			fromTemplate = action.Service
		case v1alpha1.ActionCluster:
			fromTemplate = &action.Cluster.GenerateObjectFromTemplate
		default:
			continue
		}

		spec, err := serviceutils.GetServiceSpec(ctx, r.GetClient(), scenario, *fromTemplate)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot retrieve service spec")
		}

		// store everything on a map to avoid duplicates
		for _, dashboard := range spec.Decorators.Telemetry {
			dedup[dashboard] = struct{}{}
		}
	}

	return structure.SortedMapKeys(dedup), nil
}

// connectToGrafana creates a dedicated link between the scenario controller and the Grafana service.
// The link must be destroyed if the scenario is deleted, since any new instance will change the ip of Grafana.
func (r *Controller) connectToGrafana(ctx context.Context, t *v1alpha1.Scenario) error {
	if t.Status.GrafanaEndpoint == "" {
		r.Logger.Info("The Grafana endpoint is empty. Skip telemetry.", "scenario", t.GetName())
		return nil
	}

	// if a client exists, return the client directly.
	if c := grafana.GetClientFor(t); c != nil {
		return nil
	}

	// otherwise, re-create a client.
	// this condition captures both the cases:
	// 1) this is the first time we create a client to the controller
	// 2) the controller has been restarted and lost all of the create controllers.

	var endpoint string

	if configuration.Global.DeveloperMode {
		/* If in developer mode, the operator runs outside the cluster, and will reach Grafana via the ingress */
		endpoint = common.ExternalEndpoint(defaultGrafanaName, t.GetNamespace())
	} else {
		/* If the operator runs within the cluster, it will reach Grafana via the service */
		endpoint = common.InternalEndpoint(defaultGrafanaName, t.GetNamespace(), GrafanaPort)
	}

	_, err := grafana.New(ctx,
		grafana.WithHTTP(endpoint),   // Connect to ...
		grafana.WithRegisterFor(t),   // Used by grafana.GetFrisbeeClient(), grafana.ClientExistsFor(), ...
		grafana.WithLogger(r.Logger), // Log info
		grafana.WithNotifications(WebhookURL),
	)

	return err
}

var gracefulShutDown = 30 * time.Second

var WebhookURL string

var startWebhookOnce sync.Once

const alertingWebhook = "alerting-service"

// CreateWebhookServer  creates a Webhook for listening for events from Grafana *
func (r *Controller) CreateWebhookServer(ctx context.Context, alertingPort int) error {
	WebhookURL = fmt.Sprintf("http://%s:%d", alertingWebhook, alertingPort)

	r.Logger.Info("StartWebhook", "URL", WebhookURL)

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

	idleConnectionsClosed := make(chan error)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			idleConnectionsClosed <- err
		}
	}()

	go func() {
		select {
		case <-ctx.Done():
			r.Logger.Info("Shutdown signal received, waiting for webhook server to finish")

		case err := <-idleConnectionsClosed:
			r.Logger.Error(err, "Shutting down the webhook server")
		}

		// need a new background context for the graceful shutdown. the ctx is already cancelled.
		gracefulTimeout, cancel := context.WithTimeout(context.Background(), gracefulShutDown)
		defer cancel()

		if err := srv.Shutdown(gracefulTimeout); err != nil {
			r.Logger.Error(err, "shutting down the webhook server")
		}
		close(idleConnectionsClosed)
	}()

	return nil
}
