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

package call

import (
	"context"
	"fmt"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/carv-ics-forth/frisbee/controllers/common/labelling"
	"github.com/carv-ics-forth/frisbee/pkg/structure"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *Controller) constructJobSpecList(ctx context.Context, cr *v1alpha1.Call) ([]v1alpha1.Callable, error) {
	specs := make([]v1alpha1.Callable, 0, len(cr.Spec.Services))

	for _, serviceName := range cr.Spec.Services {
		// get service spec
		var service v1alpha1.Service

		key := client.ObjectKey{
			Namespace: cr.GetNamespace(),
			Name:      serviceName,
		}

		if err := r.GetClient().Get(ctx, key, &service); err != nil {
			return nil, errors.Wrapf(err, "cannot get info for service %s", serviceName)
		}

		// find callable
		callable, ok := service.Spec.Callables[cr.Spec.Callable]
		if !ok {
			return nil, errors.Errorf("cannot find callable '%s' on service '%s'. Available: %s",
				cr.Spec.Callable, serviceName, structure.MapKeys(service.Spec.Callables))
		}

		specs = append(specs, callable)
	}

	return specs, nil
}

func (r *Controller) callJob(ctx context.Context, cr *v1alpha1.Call, i int) error {
	var mock v1alpha1.VirtualObject
	// Call normally does not return anything. This however would break all the pipeline for
	// managing dependencies between jobs. For that, we return a dummy virtual object without dedicated controller.
	// Delete normally does not return anything. This however would break all the pipeline for
	// managing dependencies between jobs. For that, we return a dummy virtual object without dedicated controller.
	// FIXME: if the call fails, this object will be re-created, and the call will failed with an "existing object" error.
	{
		mock.SetGroupVersionKind(v1alpha1.GroupVersion.WithKind("VirtualObject"))
		mock.SetName(fmt.Sprintf("%s-%d", cr.GetName(), i))

		labelling.Propagate(&mock, cr)

		if err := common.Create(ctx, r, cr, &mock); err != nil {
			return errors.Wrapf(err, "cannot create virtual object")
		}
	}

	{ // Perform the actual job
		serviceName := cr.Spec.Services[i]
		callable := cr.Status.QueuedJobs[i]

		r.Info("===> Synchronous call", "caller", cr.GetName(), "service", serviceName)
		defer r.Info("<=== Synchronous call", "caller", cr.GetName(), "service", serviceName)

		pod := types.NamespacedName{
			Namespace: cr.GetNamespace(),
			Name:      serviceName,
		}

		// Do some hacks to abort the call if the main context is cancelled.
		quit := make(chan error, 1)

		go func() {
			defer close(quit)

			res, err := r.executor.Exec(pod, callable.Container, callable.Command, true)
			if err != nil {
				quit <- errors.Wrapf(err, "command execution failed. Out: %s, Err: %s", res.Stdout, res.Stderr)
			}

			r.Logger.Info("Call Output", "stdout", res.Stdout, "stderr", res.Stderr)
		}()

		select {
		case <-ctx.Done():
			return errors.Wrapf(ctx.Err(), "cancel operation")
		case err := <-quit:
			if err != nil {
				return errors.Wrapf(err, "call failed")
			}
		}
	}

	{ // update the status of the mockup. This will be captured by the lifecycle.
		mock.SetReconcileStatus(v1alpha1.Lifecycle{
			Phase:   v1alpha1.PhaseSuccess,
			Reason:  "AllJobsCalled",
			Message: "Job is called ",
		})

		if err := common.UpdateStatus(ctx, r, &mock); err != nil {
			return errors.Wrapf(err, "cannot update job status")
		}
	}

	return nil
}
