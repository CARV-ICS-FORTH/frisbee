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
	"github.com/carv-ics-forth/frisbee/controllers/utils"
	"github.com/go-logr/logr"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func namesOfItems(list corev1.ConfigMapList) []string {
	names := make([]string, 0, len(list.Items))

	for _, obj := range list.Items {
		names = append(names, obj.GetName())
	}

	return names
}

// Get returns the system configuration
func Get(ctx context.Context, c client.Client, logger logr.Logger) (v1alpha1.Configuration, error) {
	// 1. Discovery the configuration across the various namespaces.
	var list corev1.ConfigMapList

	if err := utils.Discover(ctx, c, &list, PlatformConfigurationName); err != nil {
		return v1alpha1.Configuration{}, errors.Wrapf(err, "cannot discover '%s'", PlatformConfigurationName)
	}

	// ensure that we have spotted only one configuration
	if len(list.Items) != 1 {
		return v1alpha1.Configuration{}, errors.Errorf("Expected a single resource for '%s' but got #%s",
			PlatformConfigurationName, namesOfItems(list))
	}

	config := list.Items[0]

	var sysConf v1alpha1.Configuration

	// 2. Parse the configuration
	decoderConfig := &mapstructure.DecoderConfig{
		DecodeHook:       nil,
		ErrorUnused:      true,
		ZeroFields:       true,
		WeaklyTypedInput: true,
		Squash:           false,
		Metadata:         nil,
		Result:           &sysConf,
		TagName:          "",
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return v1alpha1.Configuration{}, errors.Wrapf(err, "cannot create decoder")
	}

	if err := decoder.Decode(config.Data); err != nil {
		return v1alpha1.Configuration{}, errors.Wrapf(err, "decoding error")
	}

	logger.Info("Set configuration parameters",
		"source", PlatformConfigurationName,
		"parameters", sysConf,
	)

	return sysConf, nil
}

func SetGlobal(conf v1alpha1.Configuration) {
	Global = conf
}

var Global v1alpha1.Configuration
