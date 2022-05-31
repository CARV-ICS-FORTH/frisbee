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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Discover discovers a resource across different namespaces
func Discover(ctx context.Context, c client.Client, crList client.ObjectList, id string) error {
	// find the platform configuration (which may reside on a different namespace)
	filters := []client.ListOption{
		client.MatchingLabels{v1alpha1.ResourceDiscoveryLabel: id},
	}

	if err := c.List(ctx, crList, filters...); err != nil {
		return errors.Wrapf(err, "cannot list resources")
	}

	return nil
}

// Get returns the system configuration
func Get(ctx context.Context, c client.Client, logger logr.Logger) (v1alpha1.Configuration, metav1.ObjectMeta, error) {
	// 1. Discovery the configuration across the various namespaces.
	var list corev1.ConfigMapList

	if err := Discover(ctx, c, &list, PlatformConfigurationName); err != nil {
		return v1alpha1.Configuration{}, metav1.ObjectMeta{}, errors.Wrapf(err, "cannot discover '%s'", PlatformConfigurationName)
	}

	// ensure that we have spotted only one configuration
	if len(list.Items) != 1 {
		return v1alpha1.Configuration{}, metav1.ObjectMeta{}, errors.Errorf("Expected a single resource for '%s' but got #%d",
			PlatformConfigurationName, len(list.Items))
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
		return v1alpha1.Configuration{}, metav1.ObjectMeta{}, errors.Wrapf(err, "cannot create decoder")
	}

	if err := decoder.Decode(config.Data); err != nil {
		return v1alpha1.Configuration{}, metav1.ObjectMeta{}, errors.Wrapf(err, "decoding error")
	}

	logger.Info("Set configuration parameters",
		"source", PlatformConfigurationName,
		"parameters", sysConf,
	)

	return sysConf, config.ObjectMeta, nil
}

func SetGlobal(conf v1alpha1.Configuration) {
	Global = conf
}

var Global v1alpha1.Configuration
