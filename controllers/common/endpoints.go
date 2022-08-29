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
	"time"

	"github.com/carv-ics-forth/frisbee/pkg/configuration"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var BackoffForK8sEndpoint = wait.Backoff{
	Duration: 1 * time.Second,
	Factor:   5,
	Jitter:   0.1,
	Steps:    3,
}

var BackoffForServiceEndpoint = wait.Backoff{
	Duration: 5 * time.Second,
	Factor:   5,
	Jitter:   0.1,
	Steps:    3,
}

// InternalEndpoint creates an endpoint for accessing the service within the cluster.
func InternalEndpoint(name string, planName string, port int64) string {
	return fmt.Sprintf("%s.%s:%d", name, planName, port)
}

// ExternalEndpoint creates an endpoint for accessing the service outside the cluster.
func ExternalEndpoint(name, planName string) string {
	return fmt.Sprintf("%s-%s.%s", name, planName, configuration.Global.DomainName)
}

// GenerateName names the children of a given resource. if there is only one instance, it will be named after the group.
// otherwise, the instances will be named as Master-0, Master-1, ...
func GenerateName(group metav1.Object, i int, max int) string {
	if max == 1 {
		return group.GetName()
	}

	return fmt.Sprintf("%s-%d", group.GetName(), i)
}
