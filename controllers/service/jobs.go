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
	"context"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	serviceutils "github.com/carv-ics-forth/frisbee/controllers/service/utils"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func (r *Controller) runJob(ctx context.Context, service *v1alpha1.Service) error {
	setDefaultValues(service)

	if err := decoratePod(ctx, r, service); err != nil {
		return errors.Wrapf(err, "cannot set pod decorators")
	}

	if err := serviceutils.AddDNSService(ctx, r, service); err != nil {
		return errors.Wrapf(err, "failed to add dns server")
	}

	// finally, create the pod
	var pod corev1.Pod

	pod.SetName(service.GetName())
	v1alpha1.PropagateLabels(&pod, service)
	pod.SetAnnotations(service.GetAnnotations())

	service.Spec.PodSpec.DeepCopyInto(&pod.Spec)

	if err := common.Create(ctx, r, service, &pod); err != nil {
		return errors.Wrapf(err, "cannot create pod")
	}

	return nil
}

func setDefaultValues(service *v1alpha1.Service) {
	// Set the restart policy
	service.Spec.RestartPolicy = corev1.RestartPolicyNever

	// Set the pre/post execution hooks
	/*
		for i := 0; i < len(service.Spec.Containers); i++ {
			// Use this for the telemetry sidecar to be able to enter the cgroup of the main container

				cr.Spec.Containers[i].Lifecycle = &corev1.Lifecycle{
					PostStart: &corev1.Handler{
						Exec: &corev1.ExecAction{
							Command: []string{
								// "/bin/sh", "-c", "|", "cut -d ' ' -f 4 /proc/self/stats > /dev/shm/app",
							},
						},
					},
					PreStop: nil,
				}
		}
	*/
}

func decoratePod(ctx context.Context, controller *Controller, service *v1alpha1.Service) error {
	// set labels
	if req := service.Spec.Decorators.Labels; req != nil {
		service.SetLabels(labels.Merge(service.GetLabels(), req))
	}

	// set annotations
	if req := service.Spec.Decorators.Annotations; req != nil {
		service.SetAnnotations(labels.Merge(service.GetAnnotations(), req))
	}

	// set dynamically evaluated fields
	if req := service.Spec.Decorators.SetFields; req != nil {
		for _, val := range req {
			if err := serviceutils.SetField(service, val); err != nil {
				return errors.Wrapf(err, "cannot set field [%v]", val)
			}
		}
	}

	if err := serviceutils.AddTelemetrySidecar(ctx, controller.GetClient(), service); err != nil {
		return errors.Wrapf(err, "failed to add telemetry")
	}

	if err := serviceutils.AddIngress(ctx, controller, service); err != nil {
		return errors.Wrapf(err, "failed to add ingress")
	}

	return nil
}
