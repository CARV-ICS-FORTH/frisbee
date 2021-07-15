package workflow

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/controllers/common/selector/template"
	"github.com/grafana-tools/sdk"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
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

	var prometheus v1alpha1.Service
	{ // Install Prometheus
		if err := r.installPrometheus(ctx, obj, &prometheus); err != nil {
			return errors.Wrapf(err, "prometheus error")
		}
	}

	var grafana v1alpha1.Service
	{ // Install Grafana
		if err := r.installGrafana(ctx, obj, &grafana); err != nil {
			return errors.Wrapf(err, "grafana error")
		}
	}

	{ // Make Prometheus and Grafana accessible from outside the Cluster
		if err := r.installIngress(ctx, obj, &prometheus, &grafana); err != nil {
			return errors.Wrapf(err, "ingress error")
		}
	}

	{ // wait for prometheus and  grafana to start running and then proceed with the experiment
		if err := common.GetLifecycle(ctx, obj.GetUID(), &prometheus, grafana.GetName()).
			Expect(v1alpha1.PhaseRunning); err != nil {
			return errors.Wrapf(err, "grafana is not running")
		}

		if err := common.GetLifecycle(ctx, obj.GetUID(), &grafana, grafana.GetName()).
			Expect(v1alpha1.PhaseRunning); err != nil {
			return errors.Wrapf(err, "grafana is not running")
		}
	}

	{ // use Grafana client for sending annotations
		apiURI := fmt.Sprintf("http://%s", virtualhost(grafana.GetName(), obj.Spec.Ingress))

		if err := newAnnotationClient(ctx, apiURI); err != nil {
			return errors.Wrapf(err, "cannot create annotations client")
		}

		r.Logger.Info("Monitoring stack is ready",
			"packages", obj.Spec.ImportMonitors,
			"grafana", apiURI,
		)
	}

	return nil
}

func (r *Reconciler) installPrometheus(ctx context.Context, obj *v1alpha1.Workflow, prom *v1alpha1.Service) error {
	prometheusSpec := template.SelectService(ctx, template.ParseRef(obj.GetNamespace(), prometheusTemplate))
	if prometheusSpec == nil {
		return errors.New("cannot find template")
	}

	prometheusSpec.DeepCopyInto(&prom.Spec)

	if err := common.SetOwner(obj, prom, "prometheus"); err != nil {
		return errors.Wrapf(err, "ownership error")
	}

	if err := r.Create(ctx, prom); err != nil {
		return errors.Wrapf(err, "request error")
	}

	r.Logger.Info("Prometheus was installed")

	return nil
}

func (r *Reconciler) installGrafana(ctx context.Context, obj *v1alpha1.Workflow, grafana *v1alpha1.Service) error {
	grafanaSpec := template.SelectService(ctx, template.ParseRef(obj.GetNamespace(), grafanaTemplate))
	if grafanaSpec == nil {
		return errors.New("cannot fubd template")
	}

	grafanaSpec.DeepCopyInto(&grafana.Spec)

	if err := common.SetOwner(obj, grafana, "grafana"); err != nil {
		return errors.Wrapf(err, "ownership error")
	}

	if err := r.importDashboards(ctx, obj, grafana); err != nil {
		return errors.Wrapf(err, "unable to import dashboards")
	}

	if err := r.Client.Create(ctx, grafana); err != nil {
		return errors.Wrapf(err, "request error")
	}

	r.Logger.Info("Grafana was installed")

	return nil
}

func (r *Reconciler) importDashboards(ctx context.Context, obj *v1alpha1.Workflow, grafana *v1alpha1.Service) error {
	// FIXME: https://github.com/kubernetes/kubernetes/pull/63362#issuecomment-386631005

	// create configmap
	configMap := corev1.ConfigMap{}
	{
		configMap.Data = make(map[string]string, len(obj.Spec.ImportMonitors))

		for _, monRef := range obj.Spec.ImportMonitors {
			monSpec := template.SelectMonitor(ctx, template.ParseRef(obj.GetNamespace(), monRef))
			if monSpec == nil {
				return errors.Errorf("monitor failed")
			}

			configMap.Data[monSpec.Dashboard.File] = monSpec.Dashboard.Payload
		}

		if err := common.SetOwner(obj, &configMap, "dashboards"); err != nil {
			return errors.Wrapf(err, "ownership failed")
		}

		if err := r.Client.Create(ctx, &configMap); err != nil {
			return errors.Wrapf(err, "configmap failed")
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

func (r *Reconciler) installIngress(ctx context.Context, obj *v1alpha1.Workflow, services ...*v1alpha1.Service) error {
	if obj.Spec.Ingress == "" || len(services) == 0 {
		return nil
	}

	var ingress netv1.Ingress

	if err := common.SetOwner(obj, &ingress, ""); err != nil {
		return errors.Wrapf(err, "ownership failed")
	}

	pathtype := netv1.PathTypePrefix

	rules := make([]netv1.IngressRule, 0, len(services))

	for _, service := range services {
		port := service.Spec.Container.Ports[0]

		rule := netv1.IngressRule{
			Host: virtualhost(service.GetName(), obj.Spec.Ingress),
			IngressRuleValue: netv1.IngressRuleValue{
				HTTP: &netv1.HTTPIngressRuleValue{
					Paths: []netv1.HTTPIngressPath{
						{
							Path:     "/",
							PathType: &pathtype,
							Backend: netv1.IngressBackend{
								Service: &netv1.IngressServiceBackend{
									Name: service.GetName(),
									Port: netv1.ServiceBackendPort{Number: port.ContainerPort},
								},
							},
						},
					},
				},
			},
		}

		rules = append(rules, rule)
	}

	ingress.Spec.Rules = rules

	if err := r.Client.Create(ctx, &ingress); err != nil {
		return errors.Wrapf(err, "unable to create ingress")
	}

	return nil
}

func virtualhost(serviceName, ingress string) string {
	return fmt.Sprintf("%s.%s", serviceName, ingress)
}

//////////////////////////////////////////////////////
// 			Mock Observability
//////////////////////////////////////////////////////

var annotationTimeout = 30 * time.Second

const (
	// grafana specific.
	statusAnnotationAdded   = "Annotation added"
	statusAnnotationPatched = "Annotation patched"
)

type AnnotationClient struct {
	ctx    context.Context
	apiURI string

	*sdk.Client
}

func newAnnotationClient(ctx context.Context, apiURI string) error {
	client, err := sdk.NewClient(apiURI, "", sdk.DefaultHTTPClient)
	if err != nil {
		return err
	}

	common.EnableAnnotations(ctx, &AnnotationClient{ctx: ctx, Client: client, apiURI: apiURI})

	return nil
}

func (c *AnnotationClient) Add(ga sdk.CreateAnnotationRequest) {
	ctx, cancel := context.WithTimeout(c.ctx, annotationTimeout)
	defer cancel()

	// submit
	gaResp, err := c.Client.CreateAnnotation(ctx, ga)
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "annotation failed"))

		return
	}

	// validate
	switch {
	case gaResp.Message == nil:
		runtime.HandleError(errors.Wrapf(err, "empty annotation response"))

	case *gaResp.Message == string(statusAnnotationAdded):
		// valid
		return

	default:
		runtime.HandleError(errors.Wrapf(err,
			"unexpected annotation response. expected %s but got %s", statusAnnotationAdded, *gaResp.Message,
		))
	}
}
