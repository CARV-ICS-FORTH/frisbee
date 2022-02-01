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

package stop

import (
	"context"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getJob(cr *v1alpha1.Stop, i int) (string, *v1alpha1.GracefulStop) {
	return cr.Spec.Services[i], cr.Status.QueuedJobs[i]
}

func (r *Controller) constructJobSpecList(ctx context.Context, cr *v1alpha1.Stop) ([]*v1alpha1.GracefulStop, error) {
	specs := make([]*v1alpha1.GracefulStop, 0, len(cr.Spec.Services))

	for _, serviceName := range cr.Spec.Services {
		var service v1alpha1.Service

		key := client.ObjectKey{
			Namespace: cr.GetNamespace(),
			Name:      serviceName,
		}

		if err := r.GetClient().Get(ctx, key, &service); err != nil {
			return nil, errors.Wrapf(err, "cannot get info for service %s", serviceName)
		}

		if service.Spec.Decorators == nil && service.Spec.Decorators.GracefulStop == nil {
			return nil, errors.Errorf("service [%s] does not support graceful stopping", service.GetName())
		}

		specs = append(specs, service.Spec.Decorators.GracefulStop)
	}

	return specs, nil
}

func (r *Controller) stopJob(cr *v1alpha1.Stop, serviceName string, stop *v1alpha1.GracefulStop) error {
	pod := types.NamespacedName{
		Namespace: cr.GetNamespace(),
		Name:      serviceName,
	}

	res, err := r.executor.Exec(pod, stop.Container, stop.Command)
	if err != nil {
		return errors.Wrapf(err, "command execution failed. Out: %s, Err: %s", res.Stdout.String(), res.Stderr.String())
	}

	return nil
}
