package workflow

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/controllers/common/lifecycle"
	"github.com/fnikolai/frisbee/controllers/common/selector/template"
	"github.com/grafana-tools/sdk"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/util/retry"
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

	prometheus, err := r.installPrometheus(ctx, obj)
	if err != nil {
		return errors.Wrapf(err, "prometheus error")
	}

	grafana, err := r.installGrafana(ctx, obj)
	if err != nil {
		return errors.Wrapf(err, "grafana error")
	}

	// Make Prometheus and Grafana accessible from outside the Cluster
	if len(obj.Spec.Ingress) > 0 {
		if err := r.installIngress(ctx, obj, prometheus, grafana); err != nil {
			return errors.Wrapf(err, "ingress error")
		}

		// use the public Grafana address (via Ingress) because the controller runs outside of the cluster
		grafanaPublicURI := fmt.Sprintf("http://%s", virtualhost(grafana.GetName(), obj.Spec.Ingress))

		client, err := newGrafanaClient(ctx, grafanaPublicURI)
		if err != nil {
			return errors.Wrapf(err, "grafana clietn error")
		}

		common.EnableAnnotations(ctx, client)
	}

	r.Logger.Info("Monitoring stack is ready", "packages", obj.Spec.ImportMonitors)

	return nil
}

func (r *Reconciler) installPrometheus(ctx context.Context, obj *v1alpha1.Workflow) (*v1alpha1.Service, error) {
	var prom v1alpha1.Service

	spec := template.SelectService(ctx, template.ParseRef(obj.GetNamespace(), prometheusTemplate))
	if spec == nil {
		return nil, errors.New("cannot find template")
	}

	spec.DeepCopyInto(&prom.Spec)

	if err := common.SetOwner(obj, &prom, "prometheus"); err != nil {
		return nil, errors.Wrapf(err, "ownership error")
	}

	if err := r.Create(ctx, &prom); err != nil {
		return nil, errors.Wrapf(err, "request error")
	}

	r.Logger.Info("Wait for Prometheus to become ready")

	if err := lifecycle.WaitReady(ctx, &prom); err != nil {
		return nil, errors.Wrapf(err, "prometheus is not running")
	}

	r.Logger.Info("Prometheus was installed")

	return &prom, nil
}

func (r *Reconciler) installGrafana(ctx context.Context, obj *v1alpha1.Workflow) (*v1alpha1.Service, error) {
	var grafana v1alpha1.Service

	grafanaSpec := template.SelectService(ctx, template.ParseRef(obj.GetNamespace(), grafanaTemplate))
	if grafanaSpec == nil {
		return nil, errors.New("cannot find template")
	}

	grafanaSpec.DeepCopyInto(&grafana.Spec)

	if err := common.SetOwner(obj, &grafana, "grafana"); err != nil {
		return nil, errors.Wrapf(err, "ownership error")
	}

	if err := r.importDashboards(ctx, obj, &grafana); err != nil {
		return nil, errors.Wrapf(err, "import dashboards")
	}

	if err := r.Client.Create(ctx, &grafana); err != nil {
		return nil, errors.Wrapf(err, "request error")
	}

	r.Logger.Info("Wait for Grafana to become ready")

	if err := lifecycle.WaitReady(ctx, &grafana); err != nil {
		return nil, errors.Wrapf(err, "prometheus is not running")
	}

	r.Logger.Info("Grafana was installed")

	return &grafana, nil
}

func (r *Reconciler) importDashboards(ctx context.Context, obj *v1alpha1.Workflow, grafana *v1alpha1.Service) error {
	// create configmap
	configMap := corev1.ConfigMap{}
	configMap.Data = make(map[string]string, len(obj.Spec.ImportMonitors))
	{
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
	volume := corev1.Volume{}
	volumeMounts := make([]corev1.VolumeMount, 0, len(configMap.Data))
	{
		volume.Name = "grafana-dashboards"
		volume.VolumeSource = corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: configMap.GetName()},
			},
		}

		for file := range configMap.Data {
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      volume.Name, // Name of a Volume.
				ReadOnly:  true,
				MountPath: filepath.Join(grafanaDashboards, file), // Path within the container
				SubPath:   file,                                   //  Path within the volume
			})
		}
	}

	// associate volume with grafana
	grafana.Spec.Volumes = append(grafana.Spec.Volumes, volume)
	grafana.Spec.Container.VolumeMounts = append(grafana.Spec.Container.VolumeMounts, volumeMounts...)

	return nil
}

func (r *Reconciler) installIngress(ctx context.Context, obj *v1alpha1.Workflow, services ...*v1alpha1.Service) error {
	var ingress netv1.Ingress

	// Ingresses annotated with 'kubernetes.io/ingress.class=ambassador' will be managed by Ambassador.
	// Without annotation, the default Ingress is used.
	ingress.SetAnnotations(map[string]string{
		"kubernetes.io/ingress.class": "ambassador",
	})

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

		r.Logger.Info("Ingress", "host", rule.Host)
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

// ////////////////////////////////////////////////////
// 			Grafana Client
// ////////////////////////////////////////////////////

const (
	statusAnnotationAdded = "Annotation added"

	statusAnnotationPatched = "Annotation patched"
)

type grafanaClient struct {
	ctx context.Context

	*sdk.Client
}

func newGrafanaClient(ctx context.Context, apiURI string) (*grafanaClient, error) {
	client, err := sdk.NewClient(apiURI, "", sdk.DefaultHTTPClient)
	if err != nil {
		return nil, errors.Wrapf(err, "client error")
	}

	// retry until Grafana is ready to receive annotations.
	err = retry.OnError(common.DefaultBackoff, func(_ error) bool { return true }, func() error {
		_, err := client.GetHealth(ctx)
		return errors.Wrapf(err, "grafana health error")
	})

	if err != nil {
		return nil, errors.Wrapf(err, "grafana is unreachable")
	}

	return &grafanaClient{
		ctx:    ctx,
		Client: client,
	}, nil

}

// Insert inserts a new annotation to Grafana
func (c *grafanaClient) Insert(ga sdk.CreateAnnotationRequest) (id uint) {
	ctx, cancel := context.WithTimeout(c.ctx, common.DefaultTimeout)
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
		return *gaResp.ID

	default:
		runtime.HandleError(errors.Wrapf(err,
			"unexpected annotation response. expected %s but got %s", statusAnnotationAdded, *gaResp.Message,
		))
	}

	return 0
}

// Patch updates an existing annotation to Grafana
func (c *grafanaClient) Patch(reqID uint, ga sdk.PatchAnnotationRequest) (id uint) {
	ctx, cancel := context.WithTimeout(c.ctx, common.DefaultTimeout)
	defer cancel()

	// submit
	gaResp, err := c.Client.PatchAnnotation(ctx, id, ga)
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "annotation failed"))

		return
	}

	// validate
	switch {
	case gaResp.Message == nil:
		runtime.HandleError(errors.Wrapf(err, "empty annotation response"))

	case *gaResp.Message == string(statusAnnotationPatched):
		// valid
		return *gaResp.ID

	default:
		runtime.HandleError(errors.Wrapf(err,
			"unexpected annotation response. expected %s but got %s", statusAnnotationPatched, *gaResp.Message,
		))
	}

	return 0
}
