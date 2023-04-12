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

package utils

import (
	"context"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/carv-ics-forth/frisbee/pkg/configuration"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
)

var pathType = netv1.PathTypePrefix

func AddIngress(ctx context.Context, controller common.Reconciler, service *v1alpha1.Service) error {
	if service.Spec.Decorators.IngressPort == nil {
		return nil
	}

	var ingress netv1.Ingress

	ingressClassName := configuration.Global.IngressClassName

	ingress.SetName(service.GetName())
	v1alpha1.PropagateLabels(&ingress, service)

	ingress.Spec = netv1.IngressSpec{
		IngressClassName: &ingressClassName,
		Rules: []netv1.IngressRule{
			{
				Host: common.ExternalEndpoint(service.GetName(), service.GetNamespace()),
				IngressRuleValue: netv1.IngressRuleValue{
					HTTP: &netv1.HTTPIngressRuleValue{
						Paths: []netv1.HTTPIngressPath{
							{
								Path:     "/",
								PathType: &pathType,
								Backend: netv1.IngressBackend{
									Service: &netv1.IngressServiceBackend{
										Name: service.GetName(),
										Port: *service.Spec.Decorators.IngressPort,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return common.Create(ctx, controller, service, &ingress)
}

func AddDNSService(ctx context.Context, controller common.Reconciler, service *v1alpha1.Service) error {
	// register ports from containers and sidecars
	var allPorts []corev1.ServicePort

	for ci, container := range service.Spec.Containers {
		for pi, port := range container.Ports {
			if port.ContainerPort == 0 {
				return errors.Errorf("port is 0 for container[%d].port[%d]", ci, pi)
			}

			allPorts = append(allPorts, corev1.ServicePort{
				Name: port.Name,
				Port: port.ContainerPort,
			})
		}
	}

	// clusterIP should be specified only with ports
	var clusterIP string

	if len(allPorts) == 0 {
		clusterIP = "None"
	}

	var k8sService corev1.Service

	k8sService.SetName(service.GetName())

	// make labels visible to the dns service
	v1alpha1.PropagateLabels(&k8sService, service)

	k8sService.Spec.Ports = allPorts
	k8sService.Spec.ClusterIP = clusterIP

	// select pods that are created by the same v1alpha1.Service as this corev1.Service
	k8sService.Spec.Selector = map[string]string{
		v1alpha1.LabelCreatedBy: service.GetName(),
	}

	return common.Create(ctx, controller, service, &k8sService)
}
