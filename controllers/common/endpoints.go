/*
Copyright 2022 ICS-FORTH.

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
	"fmt"

	"github.com/carv-ics-forth/frisbee/controllers/common/configuration"
)

// GenerateEndpoint creates an endpoint for accessing the service.
func GenerateEndpoint(name, postfix string, port int64) string {
	// protocol://name-namespace.domain

	public := fmt.Sprintf("%s-%s.%s", name, postfix, configuration.Global.DomainName)
	internal := fmt.Sprintf("%s:%d", name, port)

	/* If in developer mode, the operator runs outside the cluster, and will reach Grafana via the ingress */
	if configuration.Global.DeveloperMode {
		return internal
	}

	/* If the operator runs within the cluster, it will reach Grafana via the service */
	return public
}
