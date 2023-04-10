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

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func AddTelemetrySidecar(ctx context.Context, cli client.Client, service *v1alpha1.Service) error {
	if service.Spec.Decorators.Telemetry == nil {
		return nil
	}

	if len(service.Spec.Decorators.Telemetry) > 0 {
		//  The sidecar makes use of the shareProcessNamespace option to access the host cgroup metrics.
		share := true
		service.Spec.ShareProcessNamespace = &share
	}

	// import telemetry agents
	// import dashboards for monitoring agents to the service
	for _, monRef := range service.Spec.Decorators.Telemetry {
		monTemplate := v1alpha1.GenerateObjectFromTemplate{TemplateRef: monRef, MaxInstances: 1}

		monSpec, err := GetServiceSpec(ctx, cli, service, monTemplate)
		if err != nil {
			return errors.Wrapf(err, "cannot get monitor")
		}

		if len(monSpec.Containers) != 1 {
			return errors.Wrapf(err, "telemetry sidecar '%s' expected 1 container but got %d",
				monRef, len(monSpec.Containers))
		}

		service.Spec.Containers = append(service.Spec.Containers, monSpec.Containers[0])
		service.Spec.Volumes = append(service.Spec.Volumes, monSpec.Volumes...)
		service.Spec.Volumes = append(service.Spec.Volumes, monSpec.Volumes...)
	}

	return nil
}
