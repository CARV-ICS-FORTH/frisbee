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

package common

import (
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

// Prometheus Section
const (
	// DefaultPrometheusName should be a fixed name because it is used within the Grafana configuration.
	// Otherwise, we should find a way to replace the value.
	DefaultPrometheusName = "prometheus"
)

// Grafana Section
const (
	DefaultGrafanaServiceName = "grafana"

	DefaultGrafanaDashboardsPath = "/etc/grafana/provisioning/dashboards"

	DefaultGrafanaPort = int64(3000)

	DefaultAdvertisedAlertingServiceHost = "alerting-service"

	DefaultAdvertisedAlertingServicePort = "6666"
)

// DataViewer Section
const (
	// DefaultDataviewerName is the default name for the dataviewer service
	DefaultDataviewerName = "dataviewer"
)

// Communication Section

// DefaultBackoffForK8sEndpoint is the default backoff for controller-to-k8s communication.
var DefaultBackoffForK8sEndpoint = wait.Backoff{
	Duration: 1 * time.Second,
	Factor:   5,
	Jitter:   0.1,
	Steps:    3,
}

// DefaultBackoffForServiceEndpoint is the default backoff for controller-to-pod communication
var DefaultBackoffForServiceEndpoint = wait.Backoff{
	Duration: 10 * time.Second,
	Factor:   0.2,
	Jitter:   0.1,
	Steps:    6,
}
