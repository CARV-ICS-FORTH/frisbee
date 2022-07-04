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

package labelling

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
	// LabelName points to the plan
	LabelName = "plan.frisbee.dev/name"

	// LabelPartOf points to the action this resource is part of.
	LabelPartOf = "plan.frisbee.dev/action"

	// LabelCreatedBy points to the controller who created this resource
	LabelCreatedBy = "plan.frisbee.dev/created-by"

	// LabelComponent describes the role of the component within the architecture.
	// It can be SUT (for system under service) or SYS (if it's a frisbee component like Grafana).
	LabelComponent = "plan.frisbee.dev/component"

	// LabelInstance contains a unique name identifying the instance of the  resource
	LabelInstance = "plan.frisbee.dev/instance"
)

func SetPlan(obj *metav1.ObjectMeta, plan string) {
	metav1.SetMetaDataLabel(obj, LabelName, plan)
}

func SetAction(obj *metav1.ObjectMeta, action string) {
	metav1.SetMetaDataLabel(obj, LabelPartOf, action)
}

func SetComponent(obj *metav1.ObjectMeta, kind Component) {
	metav1.SetMetaDataLabel(obj, LabelComponent, string(kind))
}

func SetCreatedBy(child client.Object, parent client.Object) {
	child.SetLabels(labels.Merge(child.GetLabels(), map[string]string{LabelCreatedBy: parent.GetName()}))
}

func SetInstance(obj metav1.Object) {
	obj.SetLabels(labels.Merge(obj.GetLabels(), map[string]string{LabelInstance: obj.GetName()}))
}

func HasPlan(obj metav1.Object) bool {
	_, ok := obj.GetLabels()[LabelName]
	return ok
}

func GetPlan(obj metav1.Object) string {
	plan, ok := obj.GetLabels()[LabelName]
	if !ok {
		panic(errors.Errorf("Cannot extract label '%s' from resource '%s'. Labels: %s",
			LabelName, obj.GetName(), obj.GetLabels()))
	}

	return plan
}

// GetAction returns the name of the action the object belongs to.
func GetAction(obj metav1.Object) string {
	action, ok := obj.GetLabels()[LabelPartOf]
	if !ok {
		panic(errors.Errorf("Cannot extract label '%s' from resource '%s'. Labels: %s",
			LabelPartOf, obj.GetName(), obj.GetLabels()))
	}

	return action
}

// GetCreatedBy returns the creator of the resource.
func GetCreatedBy(obj metav1.Object) string {
	creator, ok := obj.GetLabels()[LabelCreatedBy]
	if !ok {
		panic(errors.Errorf("Cannot extract label '%s' from resource '%s'. Labels: %s",
			LabelCreatedBy, obj.GetName(), obj.GetLabels()))
	}

	return creator
}

func GetComponent(obj metav1.Object) Component {
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

func Propagate(child client.Object, parent client.Object) {
	child.SetLabels(labels.Merge(child.GetLabels(), parent.GetLabels()))
}

func GetInstance(obj metav1.Object) map[string]string {
	instance, ok := obj.GetLabels()[LabelInstance]
	if !ok {
		panic(errors.Errorf("Cannot extract label '%s' from resource '%s'. Labels: %s",
			LabelInstance, obj.GetName(), obj.GetLabels()))
	}

	return map[string]string{LabelInstance: instance}
}
