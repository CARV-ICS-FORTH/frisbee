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

package service

import (
	"fmt"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/pkg/lifecycle"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// updateLifecycle returns the update lifecycle of the cluster.
func (r *Controller) updateLifecycle(service *v1alpha1.Service) bool {
	// Skip any CR which are already completed, or uninitialized.
	if service.Status.Phase.Is(v1alpha1.PhaseUninitialized, v1alpha1.PhaseSuccess, v1alpha1.PhaseFailed) {
		return false
	}

	return lifecycle.SingleJob(r.view, &service.Status.Lifecycle)
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
		// Termination rules. Note the evaluation of "Main" and "Sidecars" containers do not follow any ordering.
		// It is equally possible for a "Sidecar" to be evaluated before and after the "Main" container.
		//
		// --  "Main" container is in terminal state --
		// In this case, the entire job is complete, regardless of the state of sidecar containers.
		// The job's completion status (Success or Failed) depends on the exit code of the main container.
		//
		// -- "Sidecar" container is in terminal state. --
		// This captures the condition in which a sidecar container is complete before the main container.
		// In this case, the result depends on the status of the main container.
		// 1) If the main container is in terminal state, the result follows the conditions of "Main in terminal state".
		// 2) Otherwise, if the sidecar has failed, the result is failure.
		// 3) if the sidecar is successful, the status remains running.
		var failedSidecar *v1alpha1.Lifecycle

		for _, container := range pod.Status.ContainerStatuses {
			// the container is still running
			if container.State.Terminated == nil {
				continue
			}

			if container.Name == v1alpha1.MainContainerName {
				// main has failed
				if container.State.Terminated.ExitCode != 0 {
					return v1alpha1.Lifecycle{
						Phase:   v1alpha1.PhaseFailed,
						Reason:  container.State.Terminated.Reason,
						Message: container.State.Terminated.Message,
					}
				}

				// main is successful
				return v1alpha1.Lifecycle{
					Phase:   v1alpha1.PhaseSuccess,
					Reason:  container.State.Terminated.Reason,
					Message: container.State.Terminated.Message,
				}
			}

			// sidecar has failed. cache the result. if main is complete, it has precedence.
			// if main is still running, the error will be returned at the of the loop.
			if container.State.Terminated.ExitCode != 0 {
				failedSidecar = &v1alpha1.Lifecycle{
					Phase:   v1alpha1.PhaseFailed,
					Reason:  container.State.Terminated.Reason,
					Message: container.State.Terminated.Message,
				}
			}
		}

		// lazy failure, in order to give precedence to "main" rules.
		if failedSidecar != nil {
			return *failedSidecar
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
		// A usual source for empty reason is invalid container parameters
		reason := pod.Status.Reason
		if reason == "" {
			reason = "ContainerError"
		}

		message := pod.Status.Message
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
