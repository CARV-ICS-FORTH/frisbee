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

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
)

// if there is only one instance, it will be named after the group. otherwise, the instances will be named
// as Master-0, Master-1, ...
func generateName(group *v1alpha1.Cluster, i int) string {
	if group.Spec.Instances == 1 {
		return group.GetName()
	}

	return fmt.Sprintf("%s-%d", group.GetName(), i)
}

func getJob(cluster *v1alpha1.Cluster, i int) *v1alpha1.Service {
	var instance v1alpha1.Service

	instance.SetName(generateName(cluster, i))

	cluster.Status.Expected[i].DeepCopyInto(&instance.Spec)

	return &instance
}

func (r *Controller) constructJobSpecList(ctx context.Context, cluster *v1alpha1.Cluster) ([]v1alpha1.ServiceSpec, error) {
	if err := cluster.Spec.GenerateFromTemplate.Validate(true); err != nil {
		return nil, errors.Wrapf(err, "template validation")
	}

	specs, err := r.serviceControl.GetServiceSpecList(ctx, cluster.GetNamespace(), cluster.Spec.GenerateFromTemplate)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get specs")
	}

	return specs, nil
}
