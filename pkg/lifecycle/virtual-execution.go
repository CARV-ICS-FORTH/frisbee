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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateVirtualJob wraps a call into a virtual object. This is used for operations that do not create external resources.
// Examples: Deletions, Calls, ...
// The behavior of CreateVirtualJob is practically asynchronous.
// If the callback function fails, it will be reflected in the created virtual jobs and should be captured
// by the parent's lifecycle. The CreateVirtualJob will return nil.
// If the CreateVirtualJob fails (e.g, cannot create a virtual object), it will return an error.
func CreateVirtualJob(ctx context.Context, reconciler common.Reconciler,
	parent client.Object,
	jobName string,
	callable func(vobj *v1alpha1.VirtualObject) error,
) error {
	/*---------------------------------------------------
	 * Create a Virtual Object to host the job
	 *---------------------------------------------------*/
	var vJob v1alpha1.VirtualObject

	vJob.SetGroupVersionKind(v1alpha1.GroupVersion.WithKind("VirtualObject"))
	vJob.SetNamespace(parent.GetNamespace())
	vJob.SetName(jobName)
	v1alpha1.PropagateLabels(&vJob, parent)

	if err := common.Create(ctx, reconciler, parent, &vJob); err != nil {
		return errors.Wrapf(err, "cannot create virtual resource for vJob '%s'", jobName)
	}

	reconciler.GetEventRecorderFor(parent.GetName()).Event(parent, corev1.EventTypeNormal, "VExecBegin", jobName)

	/*---------------------------------------------------
	 * Run the callback function
	 *---------------------------------------------------*/
	quit := make(chan error)

	// Trick to support context cancelling
	// Nonetheless, the call is performed synchronously
	go func() {
		quit <- callable(&vJob)
		close(quit)
	}()

	var callbackJobErr error
	select {
	case <-ctx.Done():
		callbackJobErr = ctx.Err()
	case callbackJobErr = <-quit:
	}

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

	/*---------------------------------------------------
	 * Update the status of Virtual Job
	 *---------------------------------------------------*/
	// Append information for stored data, if any
	if len(vJob.Status.Data) > 0 {
		vJob.Status.Message = fmt.Sprintf("%s. <StoredData>: '%s'", vJob.Status.Message, structure.SortedMapKeys(vJob.Status.Data))
	}

	// remove it to avoid 'metadata.resourceVersion: Invalid value: 0x0: must be specified for an update'
	// this happens because the last configuration was on create, which does not specify any version
	delete(vJob.Annotations, "kubectl.kubernetes.io/last-applied-configuration")

	if err := common.UpdateStatus(ctx, reconciler, &vJob); err != nil {
		return errors.Wrapf(err, "vexec status update error")
	}

	return nil
}
