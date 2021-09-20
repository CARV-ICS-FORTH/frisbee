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

package lifecycle

import (
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	cachetypes "k8s.io/client-go/tools/cache"
)

/******************************************************
			Lifecycle Filters
******************************************************/

type FilterFunc func(obj interface{}) bool

type Parent interface {
	GetUID() types.UID
	GetName() string
}

// FilterByParent applies the provided FilterByParent to all events coming in, and decides which events will be handled
// by this controller. It does this by looking at the objects metadata.ownerReferences field for an
// appropriate OwnerReference. It then enqueues that Foo resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped. If the parent is empty, the object is passed
// as if it belongs to this parent.
func FilterByParent(parent Parent) FilterFunc {
	if len(parent.GetUID()) == 0 {
		panic("invalid parent UID")
	}

	return func(obj interface{}) bool {
		if obj == nil {
			return false
		}

		object, ok := obj.(metav1.Object)
		if !ok {
			// an object was deleted but the watch deletion event was missed while disconnected from apiserver.
			// In this case we don't know the final "resting" state of the object,
			// so there's a chance the included `Obj` is stale.
			tombstone, ok := obj.(cachetypes.DeletedFinalStateUnknown)
			if !ok {
				runtimeutil.HandleError(errors.New("error decoding object, invalid type"))

				return false
			}

			object, ok = tombstone.Obj.(metav1.Object)
			if !ok {
				runtimeutil.HandleError(errors.New("error decoding object tombstone, invalid type"))

				return false
			}
		}

		// Update locate view of the dependent services
		for _, owner := range object.GetOwnerReferences() {
			if owner.UID == parent.GetUID() {
				return true
			}
		}

		// TODO: use it for debugging is you believe that you get more messages than expected
		// logrus.Warnf("Mismatch parent %s for object %s", parent.GetName(), object.GetName())

		return false
	}
}

// FilterByNames excludes any object that is not on the list
func FilterByNames(nameList ...string) FilterFunc {
	if len(nameList) == 0 {
		panic("empty namelist")
	}

	// convert array to map for easier lookup
	names := make(map[string]struct{}, len(nameList))

	for _, name := range nameList {
		names[name] = struct{}{}
	}

	return func(obj interface{}) bool {
		object, ok := obj.(metav1.Object)
		if !ok {
			return false
		}

		if _, ok := names[object.GetName()]; ok {
			return true
		}

		return false
	}
}
