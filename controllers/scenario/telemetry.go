/*
Copyright 2021-2023 ICS-FORTH.

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
	"sync"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	scenarioutils "github.com/carv-ics-forth/frisbee/controllers/scenario/utils"
	serviceutils "github.com/carv-ics-forth/frisbee/controllers/service/utils"
	"github.com/carv-ics-forth/frisbee/pkg/configuration"
	"github.com/carv-ics-forth/frisbee/pkg/grafana"
	"github.com/carv-ics-forth/frisbee/pkg/structure"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
)

// {{{ Internal types

func (r *Controller) StartTelemetry(ctx context.Context, scenario *v1alpha1.Scenario) error {
	// the filebrowser makes sense only if test data are enabled.
	if scenario.Spec.TestData != nil {
		if err := scenarioutils.DeployDataviewer(ctx, r, scenario); err != nil {
			return errors.Wrapf(err, "cannot provision testdata")
		}
	}

	// there is no need to import the stack of the is no dashboard.
	telemetryAgents, err := r.ListTelemetryAgents(ctx, scenario)
	if err != nil {
		return errors.Wrapf(err, "importing dashboards")
	}

	if len(telemetryAgents) > 0 {
		if err := scenarioutils.DeployPrometheus(ctx, r, scenario); err != nil {
			return errors.Wrapf(err, "prometheus error")
		}

		if err := scenarioutils.DeployGrafana(ctx, r, scenario, telemetryAgents); err != nil {
			return errors.Wrapf(err, "grafana error")
		}
	}

	return nil
}

// StopTelemetry removes the annotations from the target object, removes the Alert from Grafana, and deleted the
// client for the specific scenario.
func (r *Controller) StopTelemetry(scenario *v1alpha1.Scenario) {
	// If the resource is not initialized, then there is not registered telemetry client.
	if meta.IsStatusConditionTrue(scenario.Status.Conditions, v1alpha1.ConditionCRInitialized.String()) {
		grafana.DeleteClientFor(scenario)
	}
}

// ListTelemetryAgents iterates the referenced services (directly via Service or indirectly via Cluster) and list
// all telemetry dashboards that need to be imported.
func (r *Controller) ListTelemetryAgents(ctx context.Context, scenario *v1alpha1.Scenario) ([]string, error) {
	dedup := make(map[string]struct{})

	for _, action := range scenario.Spec.Actions {
		var fromTemplate *v1alpha1.GenerateObjectFromTemplate

		// only Service and Cluster Templates may container Telemetry Agents.
		switch action.ActionType {
		case v1alpha1.ActionService:
			fromTemplate = action.Service
		case v1alpha1.ActionCluster:
			fromTemplate = &action.Cluster.GenerateObjectFromTemplate
		default:
			continue
		}

		// get the spec from instances, not directly from the template.
		// this allows us to support conditional includes.
		specs, err := serviceutils.GetServiceSpecList(ctx, r.GetClient(), scenario, *fromTemplate)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot retrieve service spec")
		}

		// store everything on a map to avoid duplicates.
		for _, spec := range specs {
			for _, dashboard := range spec.Decorators.Telemetry {
				dedup[dashboard] = struct{}{}
			}
		}
	}

	return structure.SortedMapKeys(dedup), nil
}

// connectToGrafana creates a dedicated link between the scenario controller and the Grafana service.
// The link must be destroyed if the scenario is deleted, since any new instance will change the ip of Grafana.
func (r *Controller) connectToGrafana(ctx context.Context, scenario *v1alpha1.Scenario, notificationEndpoint string) error {
	// if a client exists, there is no need to create another one.
	if grafana.HasClientFor(scenario) {
		return nil
	}

	// otherwise, re-create a client.
	// this condition captures both the cases:
	// 1) this is the first time we create a client to the controller
	// 2) the controller has been restarted and lost its state.

	var endpoint string

	if configuration.Global.DeveloperMode {
		/* If in developer mode, the operator runs outside the cluster, and will reach Grafana via the ingress */
		endpoint = common.ExternalEndpoint(common.DefaultGrafanaServiceName, scenario.GetNamespace())
	} else {
		/* If the operator runs within the cluster, it will reach Grafana via the service */
		endpoint = common.InternalEndpoint(common.DefaultGrafanaServiceName, scenario.GetNamespace(), common.DefaultGrafanaPort)
	}

	_, err := grafana.New(ctx,
		grafana.WithHTTP(endpoint),        // Connect to ...
		grafana.WithRegisterFor(scenario), // Used by grafana.GetFrisbeeClient(), grafana.ClientExistsFor(), ...
		grafana.WithLogger(r.Logger),      // Log info
		grafana.WithNotifications(notificationEndpoint),
	)

	return err
}

var startWebhookOnce sync.Once
