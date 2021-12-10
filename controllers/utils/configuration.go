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

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// PlatformConfiguration points to a configmap that maintain information about the installation.
	PlatformConfiguration = "platform"
)

var DefaultConfiguration Configuration

type Configuration struct {
	GrafanaEndpoint string

	PrometheusEndpoint string
}

func UseDefaultPlatformConfiguration(ctx context.Context, r Reconciler, namespace string) error {
	cfg, err := GetPlatformConfiguration(ctx, r, namespace)
	if err != nil {
		return err
	}

	DefaultConfiguration = cfg

	return nil
}

// GetPlatformConfiguration loads the platform configuration.
// The configuration must be pre-installed in the platform-configuration config map. (see chart/platform).
func GetPlatformConfiguration(ctx context.Context, r Reconciler, namespace string) (Configuration, error) {
	var config corev1.ConfigMap

	key := client.ObjectKey{
		Namespace: namespace,
		Name:      PlatformConfiguration,
	}

	if err := r.GetClient().Get(ctx, key, &config); err != nil {
		return Configuration{}, errors.Wrapf(err, "cannot get configuration")
	}

	var cfg Configuration

	decoderConfig := &mapstructure.DecoderConfig{
		DecodeHook:       nil,
		ErrorUnused:      true,
		ZeroFields:       true,
		WeaklyTypedInput: true,
		Squash:           false,
		Metadata:         nil,
		Result:           &cfg,
		TagName:          "",
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return Configuration{}, errors.Wrapf(err, "cannot create decoder")
	}

	if err := decoder.Decode(config.Data); err != nil {
		return Configuration{}, errors.Wrapf(err, "decoding error")
	}

	return cfg, nil
}
