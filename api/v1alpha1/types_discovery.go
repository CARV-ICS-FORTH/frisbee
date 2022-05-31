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

const (
	// ComponentSys is a component that belongs to Frisbee. Such components can be excluded from Chaos events.
	ComponentSys = "SYS"

	// ComponentSUT is a component that belongs to the system under testing
	ComponentSUT = "SUT"
)

// https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
const (
	// ResourceDiscoveryLabel is used to discover frisbee resources across different namespaces
	ResourceDiscoveryLabel = "discover.frisbee.io/name"

	// LabelCreatedBy points to the controller/user who created this resource
	LabelCreatedBy = "plan.frisbee.io/created-by"

	// LabelPartOfPlan points to the name of a higher level application this one is part of.
	LabelPartOfPlan = "plan.frisbee.io/part-of"

	// LabelComponent describes the role of the component within the architecture.
	// It can be SUT (for system under service) or SYS (if it's a frisbee component like Grafana).
	LabelComponent = "plan.frisbee.io/component"
)
