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

package v1alpha1

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ///////////////////////////////////////////
//		System Configuration
// ///////////////////////////////////////////

const (
	// ResourceDiscoveryLabel is used to discover Frisbee resources across different namespaces.
	ResourceDiscoveryLabel = "discover.frisbee.dev/name"
)

// ///////////////////////////////////////////
//		Resource Identification
// ///////////////////////////////////////////

type Component string

const (
	// ComponentSys is a Frisbee component that is necessary for the execution of a test (e.g, Chaos, Grafana, ...)
	ComponentSys = Component("SYS")

	// ComponentSUT is a component that belongs to the system under testing.
	ComponentSUT = Component("SUT")
)

// https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
const (
	// LabelScenario points to the scenario.
	LabelScenario = "scenario.frisbee.dev/name"

	// LabelAction points to the action this resource is part of.
	LabelAction = "scenario.frisbee.dev/action"

	// LabelCreatedBy points to the controller who created this resource. It is used for listing children resources.
	LabelCreatedBy = "scenario.frisbee.dev/created-by"

	// LabelComponent describes the role of the component within the architecture (e.g, SUT or SYS).
	// It is used to handle differently the SUT resources from the SYS resources (e.g, delete the actions but not grafana).
	LabelComponent = "scenario.frisbee.dev/component"
)

func SetScenarioLabel(obj *metav1.ObjectMeta, scenario string) {
	oldValue, exists := obj.GetLabels()[scenario]
	if !exists {
		metav1.SetMetaDataLabel(obj, LabelScenario, scenario)

		return
	}

	if oldValue == scenario {
		logrus.Warnf("Overwriting scenario '%s' on object '%s'", scenario, obj.GetName())
	} else {
		panic(errors.Errorf("setting scenario '%s' failed. obj: '%s' already has scenario '%s'",
			scenario, obj.GetName(), oldValue,
		))
	}
}

func SetActionLabel(obj *metav1.ObjectMeta, actionName string) {
	oldValue, exists := obj.GetLabels()[actionName]
	if !exists {
		metav1.SetMetaDataLabel(obj, LabelAction, actionName)

		return
	}

	if oldValue == actionName {
		logrus.Warnf("Overwriting action '%s' on object '%s'", oldValue, obj.GetName())
	} else {
		panic(errors.Errorf("setting action '%s' failed. obj: '%s' already has action '%s'",
			actionName, obj.GetName(), oldValue,
		))
	}
}

func SetComponentLabel(obj *metav1.ObjectMeta, componentType Component) {
	oldValue, exists := obj.GetLabels()[string(componentType)]
	if !exists {
		metav1.SetMetaDataLabel(obj, LabelComponent, string(componentType))

		return
	}

	if oldValue == string(componentType) {
		logrus.Warnf("Overwriting component type '%s' on object '%s'", componentType, obj.GetName())
	} else {
		panic(errors.Errorf("setting component type '%s' failed. obj: '%s' already has type '%s'",
			componentType, obj.GetName(), oldValue,
		))
	}
}

func SetCreatedByLabel(child client.Object, parent client.Object) {
	oldValue, exists := child.GetLabels()[parent.GetName()]
	if !exists {
		child.SetLabels(labels.Merge(child.GetLabels(), map[string]string{LabelCreatedBy: parent.GetName()}))

		return
	}

	if oldValue == parent.GetName() {
		logrus.Warnf("Overwriting parent '%s' on object '%s'", parent.GetName(), child.GetName())
	} else {
		panic(errors.Errorf("setting parent '%s' failed. obj: '%s' already has type '%s'",
			parent.GetName(), child.GetName(), oldValue,
		))
	}
}

func PropagateLabels(child metav1.Object, parent metav1.Object) {
	child.SetLabels(labels.Merge(child.GetLabels(), parent.GetLabels()))
}

func HasScenarioLabel(obj metav1.Object) bool {
	_, ok := obj.GetLabels()[LabelScenario]

	return ok
}

func GetScenarioLabel(obj metav1.Object) string {
	scenario, ok := obj.GetLabels()[LabelScenario]
	if !ok {
		panic(errors.Errorf("Cannot extract label '%s' from resource '%s'. Labels: %s",
			LabelScenario, obj.GetName(), obj.GetLabels()))
	}

	return scenario
}

func IsSYSComponent(obj metav1.Object) bool {
	return obj.GetLabels()[LabelComponent] == string(ComponentSys)
}

func IsSUTComponent(obj metav1.Object) bool {
	return obj.GetLabels()[LabelComponent] == string(ComponentSUT)
}

// GetCreatedByLabel returns the creator of the resource.
func GetCreatedByLabel(obj metav1.Object) map[string]string {
	creator, ok := obj.GetLabels()[LabelCreatedBy]
	if !ok {
		panic(errors.Errorf("Cannot extract label '%s' from resource '%s'. Labels: %s",
			LabelCreatedBy, obj.GetName(), obj.GetLabels()))
	}

	return map[string]string{LabelCreatedBy: creator}
}

func GetComponentLabel(obj metav1.Object) Component {
	component, ok := obj.GetLabels()[LabelComponent]
	if !ok {
		panic(errors.Errorf("Cannot extract label '%s' from resource '%s'. Labels: %s",
			LabelComponent, obj.GetName(), obj.GetLabels()))
	}

	componentType := Component(component)

	if componentType != ComponentSys && componentType != ComponentSUT {
		panic(errors.Errorf("unknown component type '%s'", component))
	}

	return componentType
}

// ///////////////////////////////////////////
//		Telemetry Agents
// ///////////////////////////////////////////

const (
	// SidecarTelemetry is an annotation's value indicating the annotation's key is a telemetry agent.
	SidecarTelemetry = "sidecar.frisbee.dev/telemetry"
)

const (
	// PrometheusDiscoverablePort is a prefix that all telemetry sidecars should use in the naming of
	// the exposed ports in order to be discoverable by Prometheus.
	PrometheusDiscoverablePort = "tel-"

	// MainContainerName  is the main application that run the service. A service can be either "Main" or "Sidecar".
	MainContainerName = "main"
)
