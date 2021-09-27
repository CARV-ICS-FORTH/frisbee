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

package workflow

import (
	"context"
	"fmt"
	"path/filepath"

	pet "github.com/dustinkirkland/golang-petname"
	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/template/helpers"
	"github.com/fnikolai/frisbee/controllers/utils"
	"github.com/fnikolai/frisbee/controllers/utils/lifecycle"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// {{{ Internal types

const (
	// grafana specific.
	grafanaDashboards  = "/etc/grafana/provisioning/dashboards"
	prometheusTemplate = "observability/prometheus"
	grafanaTemplate    = "observability/grafana"
)

func (r *Reconciler) newMonitoringStack(ctx context.Context, obj *v1alpha1.Workflow) error {
	if len(obj.Spec.ImportMonitors) == 0 {
		return nil
	}

	logrus.Warnf("Create Prometheus")

	prometheus, err := r.installPrometheus(ctx, obj)
	if err != nil {
		return errors.Wrapf(err, "prometheus error")
	}

	logrus.Warnf("Create Grafana")

	grafana, err := r.installGrafana(ctx, obj)
	if err != nil {
		return errors.Wrapf(err, "grafana error")
	}

	logrus.Warnf("Create Ingress")

	// Make Prometheus and Grafana accessible from outside the ByCluster
	if obj.Spec.Ingress != nil {
		if err := r.installIngress(ctx, obj, prometheus, grafana); err != nil {
			return errors.Wrapf(err, "ingress error")
		}

		r.Logger.Info("Ingress is installed")

		// use the public Grafana address (via Ingress) because the controller runs outside the cluster
		grafanaPublicURI := fmt.Sprintf("http://%s", virtualhost(grafana.GetName(), obj.Spec.Ingress.Host))

		if err := utils.SetGrafana(ctx, grafanaPublicURI); err != nil {
			return errors.Wrapf(err, "grafana client error")
		}
	}

	r.Logger.Info("Monitoring stack is ready")

	return nil
}

func (r *Reconciler) installPrometheus(ctx context.Context, obj *v1alpha1.Workflow) (*v1alpha1.Service, error) {
	prom := v1alpha1.Service{}

	{ // metadata
		utils.SetOwner(obj, &prom)
		prom.SetName("prometheus")
	}

	{ // spec
		spec, err := helpers.GetServiceSpec(ctx, r, helpers.ParseRef(obj.GetNamespace(), prometheusTemplate))
		if err != nil {
			return nil, errors.Wrapf(err, "spec spec error")
		}

		spec.DeepCopyInto(&prom.Spec)
	}

	{ // deployment
		if err := r.GetClient().Create(ctx, &prom); err != nil {
			return nil, errors.Wrapf(err, "request error")
		}

		logrus.Warnf("Waiting for prometheus to become ready ...")

		if err := lifecycle.WaitRunningAndUpdate(ctx, r, &prom); err != nil {
			return nil, errors.Wrapf(err, "prometheus is not running")
		}
	}

	r.Logger.Info("Prometheus is installed")

	return &prom, nil
}

func (r *Reconciler) installGrafana(ctx context.Context, obj *v1alpha1.Workflow) (*v1alpha1.Service, error) {
	grafana := v1alpha1.Service{}

	{ // metadata
		utils.SetOwner(obj, &grafana)
		grafana.SetName("grafana")
	}

	{ // spec
		// to perform the necessary automations, we load the spec locally and push the modified version for creation.
		spec, err := helpers.GetServiceSpec(ctx, r, helpers.ParseRef(obj.GetNamespace(), grafanaTemplate))
		if err != nil {
			return nil, errors.Wrapf(err, "spec spec error")
		}

		if err := r.importDashboards(ctx, obj, &spec); err != nil {
			return nil, errors.Wrapf(err, "import dashboards")
		}

		spec.DeepCopyInto(&grafana.Spec)
	}

	{ // deployment
		if err := r.GetClient().Create(ctx, &grafana); err != nil {
			return nil, errors.Wrapf(err, "request error")
		}

		if err := lifecycle.WaitRunningAndUpdate(ctx, r, &grafana); err != nil {
			return nil, errors.Wrapf(err, "grafana is not running")
		}
	}

	r.Logger.Info("Grafana is installed")

	return &grafana, nil
}

func (r *Reconciler) importDashboards(ctx context.Context, obj *v1alpha1.Workflow, spec *v1alpha1.ServiceSpec) error {
	// iterate monitoring services
	for _, monRef := range obj.Spec.ImportMonitors {
		monSpec, err := helpers.GetMonitorSpec(ctx, r, helpers.ParseRef(obj.GetNamespace(), monRef))
		if err != nil {
			return errors.Errorf("monitor failed")
		}

		// get the configmap which contains our desired dashboard
		configMapKey := client.ObjectKey{Namespace: obj.GetNamespace(), Name: monSpec.Dashboard.FromConfigMap}
		configMap := corev1.ConfigMap{}

		if err := r.GetClient().Get(ctx, configMapKey, &configMap); err != nil {
			return errors.Wrapf(err, "cannot get configmap %s", configMapKey)
		}

		// create volume from the configmap
		volume := corev1.Volume{}
		volume.Name = pet.Name() // generate random name

		volume.VolumeSource = corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: configMap.GetName()},
			},
		}

		// create mountpoints
		mounts := make([]corev1.VolumeMount, 0, len(configMap.Data))

		for file := range configMap.Data {
			if file == monSpec.Dashboard.File {
				mounts = append(mounts, corev1.VolumeMount{
					Name:      volume.Name, // Name of a Volume.
					ReadOnly:  true,
					MountPath: filepath.Join(grafanaDashboards, file), // Path within the container
					SubPath:   file,                                   //  Path within the volume
				})
			}
		}

		// associate mounts to grafana container
		spec.Volumes = append(spec.Volumes, volume)
		spec.Container.VolumeMounts = append(spec.Container.VolumeMounts, mounts...)
	}

	return nil
}

func (r *Reconciler) installIngress(ctx context.Context, obj *v1alpha1.Workflow, services ...*v1alpha1.Service) error {
	ingress := netv1.Ingress{}

	{ // metadata
		utils.SetOwner(obj, &ingress)
		ingress.SetName("frisbee")

		if obj.Spec.Ingress.UseAmbassador {
			ingress.SetAnnotations(map[string]string{
				"kubernetes.io/ingress.class": "ambassador",
			})
		}
	}

	{ // spec
		pathtype := netv1.PathTypePrefix

		rules := make([]netv1.IngressRule, 0, len(services))

		for _, service := range services {
			// we now that prometheus and grafana have a single container
			port := service.Spec.Container.Ports[0]

			rule := netv1.IngressRule{
				Host: virtualhost(service.Name, obj.Spec.Ingress.Host),
				IngressRuleValue: netv1.IngressRuleValue{
					HTTP: &netv1.HTTPIngressRuleValue{
						Paths: []netv1.HTTPIngressPath{
							{
								Path:     "/",
								PathType: &pathtype,
								Backend: netv1.IngressBackend{
									Service: &netv1.IngressServiceBackend{
										Name: service.Name,
										Port: netv1.ServiceBackendPort{Number: port.ContainerPort},
									},
								},
							},
						},
					},
				},
			}

			rules = append(rules, rule)

			r.Logger.Info("Ingress", "host", rule.Host)
		}

		ingress.Spec.Rules = rules
	}

	{ // deployment
		if err := r.GetClient().Create(ctx, &ingress); err != nil {
			return errors.Wrapf(err, "unable to create ingress")
		}
	}

	return nil
}

func virtualhost(serviceName, ingress string) string {
	return fmt.Sprintf("%s.%s", serviceName, ingress)
}
