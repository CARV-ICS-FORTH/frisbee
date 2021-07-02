package workflow

import (
	"context"
	"path/filepath"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/controllers/common/selector/template"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
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

	if err := r.installPrometheus(ctx, obj); err != nil {
		return err
	}

	if err := r.installGrafana(ctx, obj); err != nil {
		return err
	}

	r.Logger.Info("Monitoring stack is ready", "packages", obj.Spec.ImportMonitors)

	return nil
}

func (r *Reconciler) installPrometheus(ctx context.Context, obj *v1alpha1.Workflow) error {
	prometheusSpec, err := template.SelectService(ctx, template.ParseRef(prometheusTemplate))
	if err != nil {
		return errors.Wrapf(err, "unable to find prometheus template")
	}

	prom := v1alpha1.Service{
		Spec: prometheusSpec,
	}

	if err := common.SetOwner(obj, &prom, "prometheus"); err != nil {
		return errors.Wrapf(err, "ownership failed %s", obj.GetName())
	}

	// Set name to match the Grafana configuration within the monitoring Template.
	prom.SetName("prometheus")

	if err := r.Create(ctx, &prom); err != nil {
		return errors.Wrapf(err, "unable to create prometheus")
	}

	r.Logger.Info("Prometheus was installed")

	return nil
}

func (r *Reconciler) installGrafana(ctx context.Context, obj *v1alpha1.Workflow) error {
	grafanaSpec, err := template.SelectService(ctx, template.ParseRef(grafanaTemplate))
	if err != nil {
		return errors.Wrapf(err, "unable to find Grafana template")
	}

	grafana := v1alpha1.Service{
		Spec: grafanaSpec,
	}

	if err := common.SetOwner(obj, &grafana, "grafana"); err != nil {
		return errors.Wrapf(err, "ownership failed %s", obj.GetName())
	}

	grafana.SetName("grafana")

	if err := r.importDashboards(ctx, obj, &grafana); err != nil {
		return errors.Wrapf(err, "unable to import dashboards")
	}

	if err := r.Create(ctx, &grafana); err != nil {
		return errors.Wrapf(err, "unable to create Grafana")
	}

	r.Logger.Info("Grafana was installed")

	return nil
}

func (r *Reconciler) importDashboards(ctx context.Context, obj *v1alpha1.Workflow, grafana *v1alpha1.Service) error {
	// create configmap
	configMap := corev1.ConfigMap{}
	{
		configMap.Data = make(map[string]string, len(obj.Spec.ImportMonitors))

		for _, monRef := range obj.Spec.ImportMonitors {
			monSpec, err := template.SelectMonitor(ctx, template.ParseRef(monRef))
			if err != nil {
				return errors.Wrapf(err, "unable to find monitoring references %s", monRef)
			}

			if monSpec == nil {
				return errors.Errorf("unable to find monitor %s", monRef)
			}

			configMap.Data[monSpec.Dashboard.File] = string(monSpec.Dashboard.Payload)
		}

		if err := common.SetOwner(obj, &configMap, "dashboards"); err != nil {
			return errors.Wrapf(err, "unable to set configmap ownership for %s", obj.GetName())
		}

		if err := r.Client.Create(ctx, &configMap); err != nil {
			return errors.Wrapf(err, "unable to create dashboard configmap for %s", obj.GetName())
		}
	}

	// create volume from the configmap
	{
		volume := corev1.Volume{}
		volume.Name = "grafana-dashboards"
		volume.VolumeSource = corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: configMap.GetName()},
			},
		}

		var volumeMounts []corev1.VolumeMount

		for file := range configMap.Data {
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      volume.Name, // Name of a Volume.
				ReadOnly:  true,
				MountPath: filepath.Join(grafanaDashboards, file), // Path within the container
				SubPath:   file,                                   //  Path within the volume
			})
		}

		grafana.Spec.Volumes = append(grafana.Spec.Volumes, volume)
		grafana.Spec.Container.VolumeMounts = append(grafana.Spec.Container.VolumeMounts, volumeMounts...)
	}

	return nil
}
