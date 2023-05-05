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

package utils

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func InstallGrafanaDashboards(ctx context.Context, reconciler common.Reconciler, scenario *v1alpha1.Scenario, spec *v1alpha1.ServiceSpec, telemetryAgents []string) error {
	imported := make(map[string]struct{})

	for _, agentRef := range telemetryAgents {
		// Every Telemetry agent must be accompanied by a configMap that contains the visualization dashboards.
		// The dashboards are expected to be named {{.TelemetryAgentName}}.config
		var dashboards corev1.ConfigMap
		{
			key := client.ObjectKey{
				Namespace: scenario.GetNamespace(),
				Name:      agentRef + ".config",
			}

			if err := reconciler.GetClient().Get(ctx, key, &dashboards); err != nil {
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
					Name:             volumeName, // Name of a Volume.
					ReadOnly:         true,
					MountPath:        filepath.Join(common.DefaultGrafanaDashboardsPath, file), // Path within the container
					SubPath:          file,                                                     //  Path within the volume
					MountPropagation: nil,
					SubPathExpr:      "",
				})

				reconciler.Info("LoadDashboard", "obj", client.ObjectKeyFromObject(&dashboards), "file", file)
			}
		}
	}

	return nil
}
