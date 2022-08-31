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

package configuration

// References are defined separately in order to facilitate the matching between Yaml configuration (of kubernetes)
// and Go code of the controller
const (
	// PlatformConfigurationName points to a configmap that maintain information about the installation.
	PlatformConfigurationName = "system.controller.configuration"

	PrometheusTemplate = "system.telemetry.prometheus.template"

	GrafanaTemplate = "system.telemetry.grafana.template"

	DataviewerTemplate = "system.telemetry.dataviewer.template"
)
