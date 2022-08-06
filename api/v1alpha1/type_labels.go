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

package v1alpha1

import (
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Component string

const (
	// ComponentSys is a component that belongs to Frisbee. Such components can be excluded from Chaos events.
	ComponentSys = Component("SYS")

	// ComponentSUT is a component that belongs to the system under testing
	ComponentSUT = Component("SUT")
)

const (
	// ResourceDiscoveryLabel is used to discover frisbee resources across different namespaces
	ResourceDiscoveryLabel = "discover.frisbee.dev/name"
)

// https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
const (
	// LabelName points to the scenario
	LabelName = "scenario.frisbee.dev/name"

	// LabelAction points to the action this resource is part of.
	LabelAction = "scenario.frisbee.dev/action"

	// LabelCreatedBy points to the controller who created this resource
	LabelCreatedBy = "scenario.frisbee.dev/created-by"

	// LabelComponent describes the role of the component within the architecture.
	// It can be SUT (for system under service) or SYS (if it's a frisbee component like Grafana).
	LabelComponent = "scenario.frisbee.dev/component"

	// LabelInstance contains a unique name identifying the instance of the  resource
	LabelInstance = "scenario.frisbee.dev/instance"
)

func SetScenarioLabel(obj *metav1.ObjectMeta, scenario string) {
	metav1.SetMetaDataLabel(obj, LabelName, scenario)
}

func SetActionLabel(obj *metav1.ObjectMeta, action string) {
	metav1.SetMetaDataLabel(obj, LabelAction, action)
}

func SetComponentLabel(obj *metav1.ObjectMeta, kind Component) {
	metav1.SetMetaDataLabel(obj, LabelComponent, string(kind))
}

func SetCreatedByLabel(child client.Object, parent client.Object) {
	child.SetLabels(labels.Merge(child.GetLabels(), map[string]string{LabelCreatedBy: parent.GetName()}))
}

func SetInstanceLabel(obj metav1.Object) {
	obj.SetLabels(labels.Merge(obj.GetLabels(), map[string]string{LabelInstance: obj.GetName()}))
}

func HasScenarioLabel(obj metav1.Object) bool {
	_, ok := obj.GetLabels()[LabelName]
	return ok
}

func GetScenarioLabel(obj metav1.Object) string {
	scenario, ok := obj.GetLabels()[LabelName]
	if !ok {
		panic(errors.Errorf("Cannot extract label '%s' from resource '%s'. Labels: %s",
			LabelName, obj.GetName(), obj.GetLabels()))
	}

	return scenario
}

// GetActionLabel returns the name of the action the object belongs to.
func GetActionLabel(obj metav1.Object) string {
	action, ok := obj.GetLabels()[LabelAction]
	if !ok {
		panic(errors.Errorf("Cannot extract label '%s' from resource '%s'. Labels: %s",
			LabelAction, obj.GetName(), obj.GetLabels()))
	}

	return action
}

// GetCreatedByLabel returns the creator of the resource.
func GetCreatedByLabel(obj metav1.Object) string {
	creator, ok := obj.GetLabels()[LabelCreatedBy]
	if !ok {
		panic(errors.Errorf("Cannot extract label '%s' from resource '%s'. Labels: %s",
			LabelCreatedBy, obj.GetName(), obj.GetLabels()))
	}

	return creator
}

func GetComponentLabel(obj metav1.Object) Component {
	component, ok := obj.GetLabels()[LabelComponent]
	if !ok {
		panic(errors.Errorf("Cannot extract label '%s' from resource '%s'. Labels: %s",
			LabelComponent, obj.GetName(), obj.GetLabels()))
	}

	v := Component(component)

	if v != ComponentSys && v != ComponentSUT {
		panic(errors.Errorf("unknown component type '%s'", component))
	}

	return v
}

func GetInstanceLabel(obj metav1.Object) map[string]string {
	instance, ok := obj.GetLabels()[LabelInstance]
	if !ok {
		panic(errors.Errorf("Cannot extract label '%s' from resource '%s'. Labels: %s",
			LabelInstance, obj.GetName(), obj.GetLabels()))
	}

	return map[string]string{LabelInstance: instance}
}

func PropagateLabels(child metav1.Object, parent metav1.Object) {
	child.SetLabels(labels.Merge(child.GetLabels(), parent.GetLabels()))
}

/////////////////////////////////////////////
//		Telemetry Agents
/////////////////////////////////////////////

const (
	// SidecarTelemetry is an annotation's value indicating the annotation's key is a telemetry agent.
	SidecarTelemetry = "sidecar.frisbee.dev/telemetry"
)

const (
	// PrometheusDiscoverablePort is a prefix that all telemetry sidecars should use in the naming of
	// the exposed ports in order to be discoverable by Prometheus.
	PrometheusDiscoverablePort = "tel-"

	// MainAppContainerName  is the main application that run the service. A service can be either "Main" or "Sidecar".
	MainAppContainerName = "app"
)

/////////////////////////////////////////////
//		Grafana Visualization
/////////////////////////////////////////////

const (
	// DrawAs hints how to mark points on the Grafana dashboard.
	DrawAs string = "grafana.frisbee.dev/draw"
	// DrawAsPoint will mark the creation and deletion of a service as distinct events.
	DrawAsPoint string = "point"
	// DrawAsRegion will draw a region starting from the creation of a service and ending to the deletion of the service.
	DrawAsRegion string = "range"
)

func DrawRegion(obj metav1.Object) bool {
	return obj.GetAnnotations()[DrawAs] == DrawAsRegion
}

func DrawPoint(obj metav1.Object) bool {
	return obj.GetAnnotations()[DrawAs] == DrawAsPoint
}
