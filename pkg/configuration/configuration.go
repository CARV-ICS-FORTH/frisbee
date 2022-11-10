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

import (
	"context"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Configuration is the programmatic equivalent of charts/platform/configuration.
type Configuration struct {
	DeveloperMode bool `json:"developerMode"`

	Namespace string `json:"namespace"`

	DomainName string `json:"domainName"`

	IngressClassName string `json:"ingressClassName"`

	ControllerName string `json:"controllerName"`
}

func (c Configuration) Validate() error {
	switch {
	case c.Namespace == "":
		return errors.Errorf("Configuration.Namespace is empty")

	case c.DomainName == "":
		return errors.Errorf("Configuration.DomainName is empty")

	case c.IngressClassName == "":
		return errors.Errorf("Configuration.IngressClassName is empty")

	case c.ControllerName == "":
		return errors.Errorf("Configuration.ControllerName is empty")
	default:
		return nil
	}
}

func namesOfItems(list corev1.ConfigMapList) []string {
	names := make([]string, 0, len(list.Items))

	for _, obj := range list.Items {
		names = append(names, obj.GetName())
	}

	return names
}

// Get returns the system configuration.
func Get(ctx context.Context, cli client.Client, logger logr.Logger) (Configuration, error) {
	// 1. Discovery the configuration across the various namespaces.
	var list corev1.ConfigMapList

	// find the platform configuration (which may reside on a different namespace)
	filters := []client.ListOption{
		client.MatchingLabels{v1alpha1.ResourceDiscoveryLabel: PlatformConfigurationName},
	}

	if err := cli.List(ctx, &list, filters...); err != nil {
		return Configuration{}, errors.Wrapf(err, "cannot discover '%s'", PlatformConfigurationName)
	}

	// ensure that we have spotted only one configuration
	if len(list.Items) != 1 {
		return Configuration{}, errors.Errorf("Expected a single resource for '%s' but got #%s",
			PlatformConfigurationName, namesOfItems(list))
	}

	config := list.Items[0]

	var sysConf Configuration

	// 2. Parse the configuration
	decoderConfig := &mapstructure.DecoderConfig{
		DecodeHook:           nil,
		ErrorUnused:          true,
		ErrorUnset:           true,
		ZeroFields:           true,
		WeaklyTypedInput:     true,
		Squash:               false,
		Metadata:             nil,
		Result:               &sysConf,
		TagName:              "",
		IgnoreUntaggedFields: false,
		MatchName:            nil,
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return Configuration{}, errors.Wrapf(err, "cannot create decoder")
	}

	if err := decoder.Decode(config.Data); err != nil {
		return Configuration{}, errors.Wrapf(err, "decoding error")
	}

	logger.Info("LoadGlobalConf",
		"config", PlatformConfigurationName,
		"parameters", sysConf,
	)

	if err := sysConf.Validate(); err != nil {
		return Configuration{}, err
	}

	return sysConf, nil
}

func SetGlobal(conf Configuration) {
	Global = conf
}

var Global Configuration
