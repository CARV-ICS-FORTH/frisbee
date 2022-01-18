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

package cascade

import (
	"context"
	"fmt"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
)

// if there is only one instance, it will be named after the group. otherwise, the instances will be named
// as Master-0, Master-1, ...
func generateName(group *v1alpha1.Cascade, i int) string {
	if group.Spec.MaxInstances == 1 {
		return group.GetName()
	}

	return fmt.Sprintf("%s-%d", group.GetName(), i)
}

func getJob(group *v1alpha1.Cascade, i int) *v1alpha1.Chaos {
	var instance v1alpha1.Chaos

	instance.SetName(generateName(group, i))

	// modulo is needed to re-iterate the job list, required for the implementation of "Until".
	jobSpec := group.Status.QueuedJobs[i%len(group.Status.QueuedJobs)]

	jobSpec.DeepCopyInto(&instance.Spec)

	return &instance
}

func (r *Controller) constructJobSpecList(ctx context.Context, group *v1alpha1.Cascade) ([]v1alpha1.ChaosSpec, error) {
	if err := group.Spec.GenerateFromTemplate.Validate(true); err != nil {
		return nil, errors.Wrapf(err, "template validation")
	}

	specs, err := r.chaosControl.GetChaosSpecList(ctx, group.GetNamespace(), group.Spec.GenerateFromTemplate)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get specs")
	}

	return specs, nil
}
