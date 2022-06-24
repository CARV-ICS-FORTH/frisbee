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

package common

import (
	"fmt"

	"github.com/carv-ics-forth/frisbee/controllers/common/configuration"
)

// InternalEndpoint creates an endpoint for accessing the service within the cluster.
func InternalEndpoint(name string, planName string, port int64) string {
	return fmt.Sprintf("%s.%s:%d", name, planName, port)
}

// ExternalEndpoint creates an endpoint for accessing the service outside the cluster.
func ExternalEndpoint(name, planName string) string {
	return fmt.Sprintf("%s-%s.%s", name, planName, configuration.Global.DomainName)
}
