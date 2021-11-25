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

package utils

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	Hook     = "app.frisbee.io/hook"
	BootHook = "boot"

	// PlatformConfiguration points to a configmap that maintain information about the installation.
	PlatformConfiguration = "platform-configuration"
)

// LoadPlatformConfiguration loads the platform configuration.
// The configuration must be pre-installed in the platform-configuration config map. (see chart/platform).
func LoadPlatformConfiguration(ctx context.Context, r Reconciler) (map[string]string, error) {
	filters := []client.ListOption{
		client.MatchingLabels{Hook: BootHook},
		//	client.MatchingFields{jobOwnerKey: req.Name},
	}

	// get grafana endpoint from the platform configurations
	var configMapList corev1.ConfigMapList

	if err := r.GetClient().List(ctx, &configMapList, filters...); err != nil {
		return nil, errors.Wrapf(err, "cannot get configuration")
	}

	if len(configMapList.Items) != 1 {
		return nil, errors.Errorf("expected [%s]. Got [%v]", PlatformConfiguration, configMapList.ListMeta)
	}

	return configMapList.Items[0].Data, nil
}

func MustGetGrafanaEndpoint(config map[string]string) string {
	endpoint, exists := config["grafana-endpoint"]
	if !exists {
		panic("cannot get grafana endpoint")
	}

	return endpoint
}
