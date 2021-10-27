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

package cluster

import (
	"context"
	"fmt"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	thelpers "github.com/fnikolai/frisbee/controllers/template/helpers"

	"github.com/fnikolai/frisbee/controllers/utils"
	"github.com/pkg/errors"
)

func getJob(r *Controller, cluster *v1alpha1.Cluster, i int) *v1alpha1.Service {
	var instance v1alpha1.Service

	{ // metadata
		utils.SetOwner(r, cluster, &instance)
		instance.SetName(generateName(cluster, i))
	}

	{ // spec
		cluster.Status.Expected[i].DeepCopyInto(&instance.Spec)
	}

	return &instance
}

func constructJobSpecList(ctx context.Context, r *Controller, cluster *v1alpha1.Cluster) ([]v1alpha1.ServiceSpec, error) {
	var specs []v1alpha1.ServiceSpec

	// all inputs are explicitly defined. no instances were given
	if cluster.Spec.Instances == 0 {
		if len(cluster.Spec.Inputs) == 0 {
			return nil, errors.New("at least one of instances || inputs must be defined")
		}

		cluster.Spec.Instances = len(cluster.Spec.Inputs)
	}

	ts := thelpers.ParseRef(cluster.GetNamespace(), cluster.Spec.TemplateRef)

	scheme, err := thelpers.Select(ctx, r, ts)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get scheme for: %s", cluster.GetName())
	}

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
			if err := thelpers.ExpandInputs(ctx, r, cluster.GetNamespace(), &scheme, cluster.Spec.Inputs[0], lookupCache); err != nil {
				return nil, errors.Wrapf(err, "macro expansion failed")
			}

		default:
			// use a different set of inputs for every instance
			if err := thelpers.ExpandInputs(ctx, r, cluster.GetNamespace(), &scheme, cluster.Spec.Inputs[i], lookupCache); err != nil {
				return nil, errors.Wrapf(err, "macro expansion failed")
			}
		}

		genSpec, err := thelpers.GenerateSpecFromScheme(&scheme)
		if err != nil {
			return nil, errors.Wrapf(err, "scheme to generic spec")
		}

		spec, err := genSpec.ToServiceSpec()
		if err != nil {
			return nil, errors.Wrapf(err, "genSpec to Service spec")
		}

		specs = append(specs, spec)
	}

	return specs, nil
}

// if there is only one instance, it will be named after the group. otherwise, the instances will be named
// as Master-0, Master-1, ...
func generateName(group *v1alpha1.Cluster, i int) string {
	if group.Spec.Instances == 1 {
		return group.GetName()
	}

	return fmt.Sprintf("%s-%d", group.GetName(), i)
}
