/*
Copyright 2022-2023 ICS-FORTH.

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

const (
	ClusterScope        = ""
	FrisbeeInstallation = "frisbee"
	FrisbeeNamespace    = "frisbee"
)

const (
	FrisbeeRepo  = "https://carv-ics-forth.github.io/frisbee/charts"
	JetstackRepo = "https://charts.jetstack.io"
)

const (
	ManagedNamespace = "app.kubernetes.io/managed-by=Frisbee"
)

const (
	TestTimeout = "24h"
)

var BackoffPodCreation = wait.Backoff{
	Duration: 10 * time.Second,
	Factor:   1.5,
	Jitter:   0.1,
	Steps:    60,
}
