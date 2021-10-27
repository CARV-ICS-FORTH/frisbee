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

package utils

import (
	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// SetOwner is a helper method to make sure the given object contains an object reference to the object provided.
// This behavior is instrumental to guarantee correct garbage collection of resources especially when resources
// control other resources in a multilevel hierarchy (think deployment-> repilcaset->pod).
func SetOwner(r Reconciler, parent, child metav1.Object) {
	child.SetNamespace(parent.GetNamespace())

	// SetControllerReference sets owner as a Controller OwnerReference on controlled.
	// This is used for garbage collection of the controlled object and for
	// reconciling the owner object on changes to controlled (with a Watch + EnqueueRequestForOwner).
	// Since only one OwnerReference can be a controller, it returns an error if
	// there is another OwnerReference with Controller flag set.
	if err := controllerutil.SetControllerReference(parent, child, r.GetClient().Scheme()); err != nil {
		panic(errors.Wrapf(err, "set controller reference"))
	}

	/*
		if err := controllerutil.SetOwnerReference(parent, child, Globals.Client.Scheme()); err != nil {
			panic(errors.Wrapf(err, "set owner reference"))
		}
	*/

	// owner labels are used by the selectors.
	// workflow labels are used to select only objects that belong to this experiment.
	// used to narrow down the scope of fault injection in a common namespace
	child.SetLabels(labels.Merge(child.GetLabels(), map[string]string{
		v1alpha1.LabelManagedBy:    parent.GetName(),
		v1alpha1.BelongsToWorkflow: parent.GetLabels()[v1alpha1.BelongsToWorkflow],
	}))
}

// IsManagedByThisController returns true if the object is managed by the specified controller.
// If it is managed by another controller, or no controller is being resolved, it returns false.
func IsManagedByThisController(obj metav1.Object, controller schema.GroupVersionKind) bool {
	owner := metav1.GetControllerOf(obj)
	if owner == nil {
		return false
	}

	if owner.APIVersion != controller.GroupVersion().String() ||
		owner.Kind != controller.Kind {
		return false
	}

	return true
}
