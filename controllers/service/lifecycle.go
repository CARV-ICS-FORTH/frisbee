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

package service

import (
	"fmt"

	"github.com/carv-ics-forth/frisbee/pkg/lifecycle"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// updateLifecycle returns the update lifecycle of the cluster.
func (r *Controller) updateLifecycle(cr *v1alpha1.Service) bool {
	// Skip any CR which are already completed, or uninitialized.
	if cr.Status.Phase.Is(v1alpha1.PhaseUninitialized, v1alpha1.PhaseSuccess, v1alpha1.PhaseFailed) {
		return false
	}

	return lifecycle.SingleJob(r.view, &cr.Status.Lifecycle)
}

// convertPodLifecycle translates the Pod's Lifecycle to Frisbee Lifecycle.
func convertPodLifecycle(obj client.Object) v1alpha1.Lifecycle {
	pod := obj.(*corev1.Pod)

	if pod.CreationTimestamp.IsZero() {
		return v1alpha1.Lifecycle{
			Phase:   v1alpha1.PhaseFailed,
			Reason:  "PodDeleted",
			Message: fmt.Sprintf("Pod %s is probably killed.", pod.GetLabels()),
		}
	}

	switch pod.Status.Phase {
	case corev1.PodPending:
		return v1alpha1.Lifecycle{
			Phase:   v1alpha1.PhasePending,
			Reason:  pod.Status.Reason,
			Message: pod.Status.Message,
		}

	case corev1.PodRunning:
		// In case that the "main" container is complete, then assume that the entire service is complete.
		// Sidecars containers will be later on garbage-collected by the service controller.
		for _, container := range pod.Status.ContainerStatuses {
			if container.Name == v1alpha1.MainContainerName && container.State.Terminated != nil {
				// Following the Linux convention, we assume that is the container has exit with zero code,
				// everything has been smoothly. If the exit code is other than 0, then there is an error.
				if container.State.Terminated.ExitCode == 0 {
					return v1alpha1.Lifecycle{
						Phase:   v1alpha1.PhaseSuccess,
						Reason:  container.State.Terminated.Reason,
						Message: container.State.Terminated.Message,
					}
				} else {
					return v1alpha1.Lifecycle{
						Phase:   v1alpha1.PhaseFailed,
						Reason:  container.State.Terminated.Reason,
						Message: container.State.Terminated.Message,
					}
				}
			}

			// TODO: Should we add the case where a side-car fails before the main container?
		}

		// All containers are still running
		return v1alpha1.Lifecycle{
			Phase:   v1alpha1.PhaseRunning,
			Reason:  pod.Status.Reason,
			Message: pod.Status.Message,
		}

	case corev1.PodSucceeded:
		return v1alpha1.Lifecycle{
			Phase:   v1alpha1.PhaseSuccess,
			Reason:  pod.Status.Reason,
			Message: pod.Status.Message,
		}

	case corev1.PodFailed:
		reason := pod.Status.Reason
		message := pod.Status.Message

		// A usual source for empty reason is invalid container parameters
		if reason == "" {
			reason = "ContainerError"
		}

		if message == "" {
			message = "Check the container logs"
		}

		return v1alpha1.Lifecycle{
			Phase:   v1alpha1.PhaseFailed,
			Reason:  reason,
			Message: message,
		}

	default:
		panic("unhandled lifecycle condition")
	}
}
