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
	"context"
	"reflect"
	"strconv"
	"strings"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/carv-ics-forth/frisbee/controllers/common/configuration"
	"github.com/carv-ics-forth/frisbee/controllers/common/labelling"
	serviceutils "github.com/carv-ics-forth/frisbee/controllers/service/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
)

func (r *Controller) runJob(ctx context.Context, cr *v1alpha1.Service) error {
	setDefaultValues(cr)

	if err := handleRequirements(ctx, r, cr); err != nil {
		return errors.Wrapf(err, "requirements error")
	}

	if err := decoratePod(ctx, r, cr); err != nil {
		return errors.Wrapf(err, "decorator error")
	}

	discovery, err := constructDiscoveryService(cr)
	if err != nil {
		return errors.Wrapf(err, "DNS service error")
	}

	if err := common.Create(ctx, r, cr, discovery); err != nil {
		return errors.Wrapf(err, "cannot create discovery service")
	}

	// finally, create the pod
	var pod corev1.Pod

	pod.SetName(cr.GetName())

	// make labels visible to the pod
	labelling.Propagate(&pod, cr)
	pod.SetAnnotations(cr.GetAnnotations())

	cr.Spec.PodSpec.DeepCopyInto(&pod.Spec)

	if err := common.Create(ctx, r, cr, &pod); err != nil {
		return errors.Wrapf(err, "cannot create pod")
	}

	return nil
}

func setDefaultValues(cr *v1alpha1.Service) {
	cr.Spec.RestartPolicy = corev1.RestartPolicyNever
}

var pathType = netv1.PathTypePrefix

func handleRequirements(ctx context.Context, r *Controller, cr *v1alpha1.Service) error {
	if cr.Spec.Requirements == nil {
		return nil
	}

	// Volume
	if req := cr.Spec.Requirements.EphemeralVolume; req != nil {
		var pvc corev1.PersistentVolumeClaim

		pvc.SetName(cr.GetName())
		req.Spec.DeepCopyInto(&pvc.Spec)

		if err := common.Create(ctx, r, cr, &pvc); err != nil {
			return errors.Wrapf(err, "cannot create pvc")
		}

		// auto-mount the created pvc.
		volume := corev1.Volume{
			Name: req.Name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.GetName(),
					ReadOnly:  false,
				},
			},
		}

		cr.Spec.Volumes = append(cr.Spec.Volumes, volume)
	}

	// Ingress
	if req := cr.Spec.Requirements.Ingress; req != nil {
		var ingress netv1.Ingress

		ingressClassName := configuration.Global.IngressClassName

		ingress.SetName(cr.GetName())
		ingress.Spec = netv1.IngressSpec{
			IngressClassName: &ingressClassName,
			Rules: []netv1.IngressRule{
				{
					Host: common.ExternalEndpoint(cr.GetName(), cr.GetNamespace()),
					IngressRuleValue: netv1.IngressRuleValue{
						HTTP: &netv1.HTTPIngressRuleValue{
							Paths: []netv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: netv1.IngressBackend{
										Service: &netv1.IngressServiceBackend{
											Name: cr.GetName(),
											Port: req.Service.Port,
										},
									},
								},
							},
						},
					},
				},
			},
		}

		if err := common.Create(ctx, r, cr, &ingress); err != nil {
			return errors.Wrapf(err, "cannot create ingress")
		}
	}

	return nil
}

func setField(cr *v1alpha1.Service, val v1alpha1.SetField) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("cannot set field [%s]. err: %s", val.Field, r)
		}
	}()

	fieldRef := reflect.ValueOf(&cr.Spec).Elem()

	index := func(v reflect.Value, idx string) reflect.Value {
		if i, err := strconv.Atoi(idx); err == nil {
			return v.Index(i)
		}

		// reflect.Value.FieldByName cannot be used on map Value
		if v.Kind() == reflect.Map {
			return reflect.Indirect(v)
		}

		return reflect.Indirect(v).FieldByName(idx)
	}

	for _, s := range strings.Split(val.Field, ".") {
		fieldRef = index(fieldRef, s)
	}

	var conv interface{} = val.Value

	// Convert src value to something that may fit to the dst.
	switch fieldRef.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		toInt, err := strconv.Atoi(val.Value)
		if err != nil {
			return errors.Wrapf(err, "convert to Int error")
		}

		conv = toInt

	case reflect.Bool:
		toBool, err := strconv.ParseBool(val.Value)
		if err != nil {
			return errors.Wrapf(err, "convert to Bool error")
		}

		conv = toBool

	case reflect.Map:
		// TODO: Needs to be improved because the map can be of various types
		logrus.Warn("THIS FUNCTION IS NOT WORKING, BUT WE DO NOT WANT TO FAIL EITHER")
		return nil

	default:
		conv = val.Value
	}

	fieldRef.Set(reflect.ValueOf(conv).Convert(fieldRef.Type()))

	return nil
}

func decoratePod(ctx context.Context, r *Controller, cr *v1alpha1.Service) error {
	if cr.Spec.Decorators == nil {
		return nil
	}

	// set labels
	if req := cr.Spec.Decorators.Labels; req != nil {
		cr.SetLabels(labels.Merge(cr.GetLabels(), req))
	}

	// set annotations
	if req := cr.Spec.Decorators.Annotations; req != nil {
		cr.SetAnnotations(labels.Merge(cr.GetAnnotations(), req))
	}

	// set dynamically evaluated fields
	if req := cr.Spec.Decorators.SetFields; req != nil {
		for _, val := range req {
			if err := setField(cr, val); err != nil {
				return errors.Wrapf(err, "cannot set field [%v]", val)
			}
		}
	}

	// set resources, to the first container only
	if req := cr.Spec.Decorators.Resources; req != nil {
		if len(cr.Spec.Containers) != 1 {
			return errors.New("Decoration resources are not applicable for multiple containers")
		}

		resources := make(map[corev1.ResourceName]resource.Quantity)

		if len(req.CPU) > 0 {
			resources[corev1.ResourceCPU] = resource.MustParse(req.CPU)
		}

		if len(req.Memory) > 0 {
			resources[corev1.ResourceMemory] = resource.MustParse(req.Memory)
		}

		cr.Spec.Containers[0].Resources = corev1.ResourceRequirements{
			Limits:   resources,
			Requests: resources,
		}
	}

	if len(cr.Spec.Decorators.Telemetry) > 0 {
		//  The sidecar makes use of the shareProcessNamespace option to access the host cgroup metrics.
		share := true
		cr.Spec.ShareProcessNamespace = &share
	}

	// import telemetry agents
	if req := cr.Spec.Decorators.Telemetry; req != nil {
		// import dashboards for monitoring agents to the service
		for _, monRef := range req {
			monSpec, err := serviceutils.GetServiceSpec(ctx, r.GetClient(), cr, v1alpha1.GenerateFromTemplate{TemplateRef: monRef})
			if err != nil {
				return errors.Wrapf(err, "cannot get monitor")
			}

			if len(monSpec.Containers) != 1 {
				return errors.Wrapf(err, "telemetry sidecar '%s' expected 1 container but got %d",
					monRef, len(monSpec.Containers))
			}

			cr.Spec.Containers = append(cr.Spec.Containers, monSpec.Containers[0])
			cr.Spec.Volumes = append(cr.Spec.Volumes, monSpec.Volumes...)
			cr.Spec.Volumes = append(cr.Spec.Volumes, monSpec.Volumes...)
		}
	}

	return nil
}

func constructDiscoveryService(cr *v1alpha1.Service) (*corev1.Service, error) {
	// register ports from containers and sidecars
	var allPorts []corev1.ServicePort

	for ci, container := range cr.Spec.Containers {
		for pi, port := range container.Ports {
			if port.ContainerPort == 0 {
				return nil, errors.Errorf("port is 0 for container[%d].port[%d]", ci, pi)
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

	var kubeService corev1.Service

	kubeService.SetName(cr.GetName())

	// make labels visibile to the dns service
	labelling.Propagate(&kubeService, cr)

	kubeService.Spec.Ports = allPorts
	kubeService.Spec.ClusterIP = clusterIP

	// bind service to the pod
	kubeService.Spec.Selector = labelling.GetInstance(cr)

	return &kubeService, nil
}
