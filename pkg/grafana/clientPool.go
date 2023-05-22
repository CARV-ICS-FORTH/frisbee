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

package grafana

import (
	"reflect"
	"sync"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	clientsLocker sync.RWMutex
	clients       = map[types.NamespacedName]*Client{}
)

func getScenarioFromLabels(obj metav1.Object) types.NamespacedName {
	if !v1alpha1.HasScenarioLabel(obj) {
		panic(errors.Errorf("Object '%s/%s' does not have Scenario labels", obj.GetNamespace(), obj.GetName()))
	}

	// The key structure is as follows:
	// Namespaces: provides separation between test-cases
	// Scenario: Is a flag that is propagated all over the test-cases.
	return types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      v1alpha1.GetScenarioLabel(obj),
	}
}

// SetClientFor creates a new client for the given object.  It panics if it cannot parse the object's metadata,
// or if another client is already registers.
func SetClientFor(obj metav1.Object, client *Client) {
	key := getScenarioFromLabels(obj)

	clientsLocker.RLock()
	_, exists := clients[key]
	clientsLocker.RUnlock()

	if exists {
		panic(errors.Errorf("client is already registered for '%s'", key))
	}

	clientsLocker.Lock()
	clients[key] = client
	clientsLocker.Unlock()

	client.logger.Info("Set Grafana client for", "obj", key)
}

// GetClientFor returns the client with the given name. It panics if it cannot parse the object's metadata,
// if the client does not exist, or if the client is empty.
func GetClientFor(obj metav1.Object) *Client {
	if !v1alpha1.HasScenarioLabel(obj) {
		logrus.Warn("No Scenario FOR ", obj.GetName(), " type ", reflect.TypeOf(obj))

		return nil
	}

	key := getScenarioFromLabels(obj)

	clientsLocker.RLock()
	defer clientsLocker.RUnlock()

	client, exists := clients[key]
	if !exists || client == nil {
		panic("nil grafana client was found for object: " + obj.GetName())
	}

	return client
}

// HasClientFor returns whether there is a non-nil grafana client is registered for the given object.
func HasClientFor(obj metav1.Object) bool {
	if !v1alpha1.HasScenarioLabel(obj) {
		return false
	}

	key := getScenarioFromLabels(obj)

	clientsLocker.RLock()
	defer clientsLocker.RUnlock()

	client, exists := clients[key]
	if !exists || client == nil {
		return false
	}

	return true
}

// DeleteClientFor removes the client registered for the given object.
func DeleteClientFor(obj metav1.Object) {
	if !v1alpha1.HasScenarioLabel(obj) {
		return
	}

	key := getScenarioFromLabels(obj)

	clientsLocker.Lock()
	defer clientsLocker.Unlock()

	delete(clients, key)
}
