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

package lifecycle

import (
	"context"
	"fmt"
	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/carv-ics-forth/frisbee/pkg/structure"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// buildVirtualObject builds a new virtual object.
func buildVirtualObject(parent metav1.Object, name string) *v1alpha1.VirtualObject {
	var vobject v1alpha1.VirtualObject

	vobject.SetGroupVersionKind(v1alpha1.GroupVersion.WithKind("VirtualObject"))
	vobject.SetNamespace(parent.GetNamespace())
	vobject.SetName(name)

	v1alpha1.PropagateLabels(&vobject, parent)

	v1alpha1.SetScenarioLabel(&vobject.ObjectMeta, parent.GetName())
	v1alpha1.SetComponentLabel(&vobject.ObjectMeta, v1alpha1.ComponentSUT)

	return &vobject
}

// VirtualExecution wraps a call into a virtual object. This is used for operations that do not create external resources.
// Examples: Deletions, Calls, ...
// The behavior of VirtualExecution is practically asynchronous.
// If the callback function fails, it will be reflected in the created virtual jobs and should be captured
// by the parent's lifecycle. The VirtualExecution will return nil.
// If the VirtualExecution fails (e.g, cannot create a virtual object), it will return an error.
func VirtualExecution(ctx context.Context, r common.Reconciler, parent client.Object, jobName string,
	cb func(vobj *v1alpha1.VirtualObject) error) error {
	// Step 1. Create the object in the Kubernetes API
	vJob := buildVirtualObject(parent, jobName)

	if err := common.Create(ctx, r, parent, vJob); err != nil {
		return errors.Wrapf(err, "cannot create virtual resource for vJob '%s'", jobName)
	}

	r.GetEventRecorderFor(parent.GetName()).Event(parent, corev1.EventTypeNormal, "VExecBegin", jobName)

	// Step 2. Run the callback function with support for context cancelling
	quit := make(chan error)
	go func() {
		quit <- cb(vJob)
		close(quit)
	}()

	var jobErr error
	select {
	case <-ctx.Done():
		jobErr = ctx.Err()
	case jobErr = <-quit:
	}

	// Step 3. Update the status of the virtual job
	if jobErr != nil {
		r.GetEventRecorderFor(parent.GetName()).Event(parent, corev1.EventTypeWarning, "VExecFailed", jobName)

		vJob.Status.Lifecycle.Phase = v1alpha1.PhaseFailed
		vJob.Status.Lifecycle.Reason = "VExecFailed"
		vJob.Status.Lifecycle.Message = errors.Wrapf(jobErr, "Job failed").Error()
	} else {

		r.GetEventRecorderFor(parent.GetName()).Event(parent, corev1.EventTypeNormal, "VExecSuccess", jobName)

		vJob.Status.Lifecycle.Phase = v1alpha1.PhaseSuccess
		vJob.Status.Lifecycle.Reason = "VExecSuccess"
		vJob.Status.Lifecycle.Message = fmt.Sprintf("Job completed")
	}

	// Step 4. Append information for stored data, if any
	if len(vJob.Status.Data) > 0 {
		vJob.Status.Message = fmt.Sprintf("%s. <StoredData>: '%s'", vJob.Status.Message, structure.SortedMapKeys(vJob.Status.Data))
	}

	// Step 5. Update the status of the mockup. This will be captured by the lifecycle.
	err := common.UpdateStatus(ctx, r, vJob)
	return errors.Wrapf(err, "vexec status update error")
}
