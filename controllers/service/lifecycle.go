// Licensed to FORTH/ICS under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. FORTH/ICS licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package service

import (
	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

// calculateLifecycle returns the update lifecycle of the cluster.
func calculateLifecycle(cr *v1alpha1.Service, pod *corev1.Pod) v1alpha1.Lifecycle {
	status := cr.Status

	if status.Phase == v1alpha1.PhaseUninitialized ||
		status.Phase == v1alpha1.PhaseSuccess ||
		status.Phase == v1alpha1.PhaseFailed {
		return status.Lifecycle
	}

	if !pod.CreationTimestamp.IsZero() {
		return convertLifecycle(pod)
	}

	return v1alpha1.Lifecycle{
		Phase:   v1alpha1.PhaseFailed,
		Reason:  "PodDeletion",
		Message: "The Pod is empty but scheduled.",
	}
}

// convertLifecycle translates the Pod's Lifecycle to Frisbee Lifecycle.
func convertLifecycle(pod *corev1.Pod) v1alpha1.Lifecycle {
	switch pod.Status.Phase {
	case corev1.PodPending:
		return v1alpha1.Lifecycle{
			Phase:   v1alpha1.PhasePending,
			Reason:  pod.Status.Reason,
			Message: pod.Status.Message,
		}

	case corev1.PodRunning:
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

	case corev1.PodUnknown:
		return v1alpha1.Lifecycle{
			Phase:   v1alpha1.PhaseFailed,
			Reason:  "unknown state",
			Message: pod.Status.Message,
		}
	default:
		logrus.Warn("DEBUG ", pod)

		panic("unhandled lifecycle condition")
	}
}
