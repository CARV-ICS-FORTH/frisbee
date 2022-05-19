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

package testplan

import (
	"context"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/utils"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// PlatformConfiguration points to a configmap that maintain information about the installation.
	PlatformConfiguration = "configuration.frisbee.io"
)

// UsePlatformConfiguration loads the platform configuration.
// The configuration must be pre-installed in the platform-configuration config map. (see chart/platform).
func UsePlatformConfiguration(ctx context.Context, r utils.Reconciler, plan *v1alpha1.TestPlan) error {
	var config corev1.ConfigMap

	key := client.ObjectKey{
		Namespace: plan.GetNamespace(),
		Name:      PlatformConfiguration,
	}

	if err := r.GetClient().Get(ctx, key, &config); err != nil {
		return errors.Wrapf(err, "cannot get configuration")
	}

	decoderConfig := &mapstructure.DecoderConfig{
		DecodeHook:       nil,
		ErrorUnused:      true,
		ZeroFields:       true,
		WeaklyTypedInput: true,
		Squash:           false,
		Metadata:         nil,
		Result:           &plan.Status.Configuration,
		TagName:          "",
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return errors.Wrapf(err, "cannot create decoder")
	}

	if err := decoder.Decode(config.Data); err != nil {
		return errors.Wrapf(err, "decoding error")
	}

	/* Inherit the metadata of the configuration. This is used to automatically delete and remove the
	resources if the configuration is deleted */
	utils.AppendLabels(plan, config.GetLabels())
	utils.AppendAnnotations(plan, config.GetAnnotations())

	r.Info("Set configuration parameters",
		"source", PlatformConfiguration,
		"parameters", plan.Status.Configuration,
	)

	return nil
}
