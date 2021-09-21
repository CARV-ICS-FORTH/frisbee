// Licensed to FORTH/ICS under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. FORTH/ICS licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package cluster

import (
	"context"
	"fmt"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/controllers/common/selector/service"
	"github.com/fnikolai/frisbee/controllers/template/helpers"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (r *Reconciler) prepare(ctx context.Context, cluster *v1alpha1.Cluster) error {

	// all inputs are explicitly defined. no instances were given
	if cluster.Spec.Instances == 0 {
		if len(cluster.Spec.Inputs) == 0 {
			return errors.New("at least one of instances || inputs must be defined")
		}

		cluster.Spec.Instances = len(cluster.Spec.Inputs)
	}

	scheme := helpers.SelectServiceTemplate(ctx, helpers.ParseRef(cluster.GetNamespace(), cluster.Spec.TemplateRef))

	// cache the results of macro as to avoid asking the Kubernetes API. This, however, is only applicable
	// to the level of a cluster, because different groups may be created in different moments
	// throughout the experiment,  thus yielding different results.
	lookupCache := make(map[string]v1alpha1.SList)

	for i := 0; i < cluster.Spec.Instances; i++ {
		switch len(cluster.Spec.Inputs) {
		case 0:
			// no inputs
		case 1:
			// use a common set of inputs for all instances
			if err := inputs2Env(ctx, scheme.Inputs.Parameters, cluster.Spec.Inputs[0], lookupCache); err != nil {
				return errors.Wrapf(err, "macro expansion failed")
			}

		default:
			// use a different set of inputs for every instance
			if err := inputs2Env(ctx, scheme.Inputs.Parameters, cluster.Spec.Inputs[i], lookupCache); err != nil {
				return errors.Wrapf(err, "macro expansion failed")
			}
		}

		var instance v1alpha1.Service

		common.SetOwner(cluster, &instance)
		instance.SetName(generateName(cluster, i))

		// set the service specification
		spec, err := helpers.GenerateServiceSpec(scheme)
		if err != nil {
			return errors.Wrapf(err, "scheme to instance")
		}

		spec.DeepCopyInto(&instance.Spec)

		// add service to the expected list
		cluster.Status.ExpectedServices = append(cluster.Status.ExpectedServices, instance)

		logrus.Warn("Generate scheme", scheme.Inputs)
	}

	return nil
}

// if there is only one instance, it will be named after the group. otherwise, the instances will be named
// as Master-0, Master-1, ...
func generateName(group *v1alpha1.Cluster, i int) string {
	if group.Spec.Instances == 1 {
		return group.GetName()
	}

	return fmt.Sprintf("%s-%d", group.GetName(), i)
}

func inputs2Env(ctx context.Context, dst, src map[string]string, cache map[string]v1alpha1.SList) error {
	for key := range dst {
		// if there is no user-given value, use the default.
		value, ok := src[key]
		if !ok {
			continue
		}

		// if the value is not a macro, write it directly to the inputs
		if !service.IsMacro(value) {
			dst[key] = value
		} else { // expand macro
			services, ok := cache[value]
			if !ok {
				services = service.Select(ctx, &v1alpha1.ServiceSelector{Macro: &value})

				if len(services) == 0 {
					return errors.Errorf("macro %s yields no services", value)
				}

				cache[value] = services
			}

			dst[key] = services.ToString()
		}
	}

	return nil
}
