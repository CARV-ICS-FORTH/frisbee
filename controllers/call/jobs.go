/*
Copyright 2021-2023 ICS-FORTH.

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
	"regexp"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/call/utils"
	"github.com/carv-ics-forth/frisbee/pkg/lifecycle"
	"github.com/carv-ics-forth/frisbee/pkg/structure"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *Controller) runJob(ctx context.Context, cr *v1alpha1.Call, i int) error {
	jobName := fmt.Sprintf("%s-%d", cr.GetName(), i)
	serviceName := cr.Spec.Services[i]
	callable := cr.Status.QueuedJobs[i]

	r.Info("-> Call", "caller", cr.GetName(), "service", serviceName)
	defer r.Info("<- Call", "caller", cr.GetName(), "service", serviceName)

	pod := types.NamespacedName{
		Namespace: cr.GetNamespace(),
		Name:      serviceName,
	}

	// Call normally does not return anything. This however would break all the pipeline for
	// managing dependencies between jobs. For that, we return a dummy virtual object without dedicated controller.
	// FIXME: if the call fails, this object will be re-created, and the call will fail with an "existing object" error.
	return lifecycle.VirtualExecution(ctx, r, cr, jobName, func(task *v1alpha1.VirtualObject) error {
		res, err := r.executor.Exec(ctx, pod, callable.Container, callable.Command, true)

		defer func() {
			// Use the virtual object to store the remote execution logs.
			task.Status.Data = map[string]string{
				"info":   fmt.Sprintf("Callable '%s/%s'", serviceName, callable.Container),
				"stdout": res.Stdout,
				"stderr": res.Stderr,
			}
		}()

		r.Logger.Info("CallOutput",
			"job", client.ObjectKeyFromObject(cr),
			"stdout", res.Stdout,
			"stderr", res.Stderr,
		)

		if err != nil {
			return errors.Wrapf(err, "call [%s/%s] has failed", serviceName, callable.Container)
		}

		if cr.Spec.Expect != nil {
			r.Logger.Info("AssertCall",
				"job", client.ObjectKeyFromObject(cr),
				"expect", cr.Spec.Expect,
			)

			expect := cr.Spec.Expect[i]

			if expect.Stdout != nil {
				matchStdout, err := regexp.MatchString(*expect.Stdout, res.Stdout)
				if err != nil {
					return errors.Wrapf(err, "regex error")
				}

				if !matchStdout {
					return errors.Errorf("Mismatched stdout. Expected: '%s' but got: '%s' --", *expect.Stdout, res.Stdout)
				}
			}

			if expect.Stderr != nil {
				matchStderr, err := regexp.MatchString(*expect.Stderr, res.Stderr)
				if err != nil {
					return errors.Wrapf(err, "regex error")
				}

				if !matchStderr {
					return errors.Errorf("Mismatched stderr. Expected: '%s' but got '%s' --", *expect.Stderr, res.Stderr)
				}
			}
		}

		return nil
	})
}

func (r *Controller) constructJobSpecList(ctx context.Context, call *v1alpha1.Call) ([]v1alpha1.Callable, error) {
	specs := make([]v1alpha1.Callable, 0, len(call.Spec.Services))

	for _, serviceName := range call.Spec.Services {
		// get service spec
		var service v1alpha1.Service

		key := client.ObjectKey{
			Namespace: call.GetNamespace(),
			Name:      serviceName,
		}

		if err := r.GetClient().Get(ctx, key, &service); err != nil {
			return nil, errors.Wrapf(err, "cannot get info for service %s", serviceName)
		}

		// find callable
		callable, ok := service.Spec.Callables[call.Spec.Callable]
		if !ok {
			return nil, errors.Errorf("callable '%s/%s' not found. Available: %s",
				call.Spec.Callable, serviceName, structure.SortedMapKeys(service.Spec.Callables))
		}

		specs = append(specs, callable)
	}

	utils.SetTimeline(call)

	return specs, nil
}
