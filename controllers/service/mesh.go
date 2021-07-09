package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common/selector/service"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
	// "k8s.io/apimachinery/pkg/labels"
)

func (r *Reconciler) discoverMesh(ctx context.Context, obj *v1alpha1.Service) error {
	if obj.Spec.Mesh == nil {
		return nil
	}

	// handle inputs
	for i :=0; i < len(obj.Spec.Mesh.Inputs); i++ {
		in := &obj.Spec.Mesh.Inputs[i]

		switch strings.ToLower(in.Type) {
		case "direct":
			direct := &Direct{}
			if err := direct.Input(r, ctx, obj, in); err != nil {
				return errors.Wrapf(err, "direct input error")

			}
		default:
			return errors.New("invalid type")
		}
	}

	// handle outputs
	for i :=0; i < len(obj.Spec.Mesh.Outputs); i++ {
		out := &obj.Spec.Mesh.Outputs[i]

		switch strings.ToLower(out.Type) {
		case "direct":
			direct := &Direct{}
			if err := direct.Output(r, ctx, obj, out); err != nil {
				return errors.Wrapf(err, "direct input error")
			}

		default:
			return errors.New("invalid type")
		}
	}

	return nil
}

type Direct struct {}

func (d *Direct) Input(r *Reconciler, ctx context.Context, obj *v1alpha1.Service, port *v1alpha1.Port) error {
	// make port discoverable
	if port.Labels != nil {
		obj.SetLabels(labels.Merge(obj.GetLabels(), port.Labels))
	}

	return nil
}

func (d *Direct) Output(r *Reconciler, ctx context.Context, obj *v1alpha1.Service, port *v1alpha1.Port) error {
	// make port discoverable
	if port.Labels != nil {
		obj.SetLabels(labels.Merge(obj.GetLabels(), port.Labels))
	}

	// find dependent services
	if port.Selector.Mode != v1alpha1.OneMode {
		return errors.New("Direct protocol supports only \"OneMode\" selector")
	}

	matches := service.Select(ctx, port.Selector)
	if len(matches) != 1 {
		return errors.Errorf("expected 1 server, but got multiple (%s)", matches.String())
	}

	match := matches[0]

	// create annotations
	obj.SetAnnotations(map[string]string{
		annotate("outputs", port.Name, "dst"):  match.GetName(),
		annotate("outputs", port.Name, "port"): "5201",
	})

	logrus.Warn("Annotations ", obj.GetAnnotations())

	return nil // nothing to do here
}

func annotate(direction, name, property string) string {
	return fmt.Sprintf("mesh.%s.%s.%s", direction, name, property)
}

/*
func tcpInput(obj *v1alpha1.Service) error {

	if len(obj.Spec.Container.Ports) == 0 {
		return nil
	}

	// use annotation to convey the discoverable information
	annotation := "mesh.tcpTraffic.port"
	for _, port := range obj.Spec.Container.Ports {
		if metav1.HasAnnotation(obj.ObjectMeta, annotation) {
			return errors.New("annotation already exists")
		}

		metav1.SetMetaDataAnnotation(&obj.ObjectMeta, annotation, fmt.Sprint(port.ContainerPort))
	}

	// use labels to public the offered input

	return nil
}

*/

/*

func outputs(ctx context.Context, obj *v1alpha1.Service) error {
	if obj.Spec.Mesh.Demands == nil {
		return nil
	}

	corev1.EnvVar{}

	for _, port := range obj.Spec.Mesh.Demands {
		protocol, impl := port.GetProtocol()

		switch v:= impl.(type) {
		case *v1alpha1.File:


		case *v1alpha1.Direct:
			if handleTCP(ctx, v)
		}

	}
}




*/
