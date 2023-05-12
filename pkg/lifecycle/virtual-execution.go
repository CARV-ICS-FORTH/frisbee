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

package lifecycle

import (
	"context"
	"fmt"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/carv-ics-forth/frisbee/pkg/structure"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateVirtualJob wraps a call into a virtual object. This is used for operations that do not create external resources.
// Examples: Deletions, Calls, ...
// If the callback function fails, it will be reflected in the created virtual jobs and should be captured
// by the parent's lifecycle. The CreateVirtualJob will return nil.
// If the CreateVirtualJob fails (e.g, cannot create a virtual object), it will return an error.
func CreateVirtualJob(ctx context.Context, reconciler common.Reconciler,
	parent client.Object,
	jobName string,
	callback func(vobj *v1alpha1.VirtualObject) error,
) error {
	/*---------------------------------------------------
	 * Create a Virtual Object to host the job
	 *---------------------------------------------------*/
	var vJob v1alpha1.VirtualObject

	vJob.SetGroupVersionKind(v1alpha1.GroupVersion.WithKind("VirtualObject"))
	vJob.SetNamespace(parent.GetNamespace())
	vJob.SetName(jobName)

	// Set default metadata for the virtual object.
	v1alpha1.SetScenarioLabel(&vJob.ObjectMeta, parent.GetName())
	v1alpha1.SetComponentLabel(&vJob.ObjectMeta, v1alpha1.ComponentSUT)

	// Copy parent's metadata (defaults will be overwritten).
	v1alpha1.PropagateLabels(&vJob, parent)

	if err := common.Create(ctx, reconciler, parent, &vJob); err != nil {
		return errors.Wrapf(err, "cannot create virtual resource for vJob '%s'", jobName)
	}

	reconciler.GetEventRecorderFor(parent.GetName()).Event(parent, corev1.EventTypeNormal, "VExecBegin", jobName)

	/*---------------------------------------------------
	 * Retrieve the created virtual object
	 *---------------------------------------------------*/
	// dirty solution to get the ResourceVersion is order to avoid update failing with
	// 'Invalid value: 0x0: must be specified for an update'
	// retry to until we get information about the service.
	key := client.ObjectKeyFromObject(&vJob)

	if err := retry.OnError(common.DefaultBackoffForServiceEndpoint,
		// retry condition
		func(err error) bool {
			reconciler.Info("Retry to get info about virtualobject",
				"virtualobject", key,
				"Err", err,
			)

			return k8errors.IsNotFound(err)
		},
		// execution
		func() error {
			return reconciler.GetClient().Get(ctx, key, &vJob)
		},
		// error checking
	); err != nil {
		return errors.Wrapf(err, "failed to retrieve virtual object")
	}

	/*---------------------------------------------------
	 * Run the callback function asynchronously
	 *---------------------------------------------------*/
	go func() {
		callbackJobErr := callback(&vJob)

		// resolve the status
		if callbackJobErr != nil {
			vJob.Status.Lifecycle.Phase = v1alpha1.PhaseFailed
			vJob.Status.Lifecycle.Reason = "VExecFailed"
			vJob.Status.Lifecycle.Message = errors.Wrapf(callbackJobErr, "Job failed").Error()

			reconciler.GetEventRecorderFor(parent.GetName()).Event(parent, corev1.EventTypeWarning, "VExecFailed", jobName)
		} else {
			vJob.Status.Lifecycle.Phase = v1alpha1.PhaseSuccess
			vJob.Status.Lifecycle.Reason = "VExecSuccess"
			vJob.Status.Lifecycle.Message = "Job completed"

			reconciler.GetEventRecorderFor(parent.GetName()).Event(parent, corev1.EventTypeNormal, "VExecSuccess", jobName)
		}

		// Append information for stored data, if any
		if len(vJob.Status.Data) > 0 {
			vJob.Status.Lifecycle.Message = fmt.Sprintf("%s. <StoredData>: '%s'", vJob.Status.Message, structure.SortedMapKeys(vJob.Status.Data))
		}

		/*---------------------------------------------------
		 * Update the status of the Virtual Job
		 *---------------------------------------------------*/
		if err := common.UpdateStatus(ctx, reconciler, &vJob); err != nil {
			reconciler.GetEventRecorderFor(parent.GetName()).Event(parent, corev1.EventTypeWarning,
				"UpdateFailed", "vexec status update error")
		}
	}()

	return nil
}
