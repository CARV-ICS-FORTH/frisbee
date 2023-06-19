/*
Copyright 2022-2023 ICS-FORTH.

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

package client

import (
	"context"
	"strings"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewTestManagementClient creates new Test client.
func NewTestManagementClient(client client.Client) TestManagementClient {
	return TestManagementClient{
		client: client,
	}
}

type TestManagementClient struct {
	client client.Client
}

// GetScenario returns single scenario by id.
func (c TestManagementClient) GetScenario(ctx context.Context, id string) (scenario *v1alpha1.Scenario, err error) {
	filters := &client.ListOptions{Namespace: id}

	var scenarios v1alpha1.ScenarioList

	if err := c.client.List(ctx, &scenarios, filters); err != nil {
		return nil, errors.Wrapf(err, "cannot list resources")
	}

	switch numItems := len(scenarios.Items); numItems {
	case 0:
		return nil, nil
	case 1:
		return &scenarios.Items[0], nil
	default:
		return nil, errors.Errorf("test '%s' has %d scenarios", id, numItems)
	}
}

// ListScenarios list all scenarios.
func (c TestManagementClient) ListScenarios(ctx context.Context, selector string) (scenarios v1alpha1.ScenarioList, err error) {
	set, err := labels.ConvertSelectorToLabelsMap(selector)
	if err != nil {
		return scenarios, errors.Wrapf(err, "invalid selector")
	}

	// find namespaces where scenarios are running
	filters := &client.ListOptions{LabelSelector: labels.SelectorFromValidatedSet(set)}

	var namespaces corev1.NamespaceList

	if err := c.client.List(ctx, &namespaces, filters); err != nil {
		return scenarios, errors.Wrapf(err, "cannot list resource")
	}

	// extract scenarios from the namespaces
	for _, namespace := range namespaces.Items {
		var localList v1alpha1.ScenarioList

		if err := c.client.List(ctx, &localList, &client.ListOptions{Namespace: namespace.GetName()}); err != nil {
			return scenarios, errors.Wrapf(err, "cannot list resources")
		}

		switch numItems := len(localList.Items); numItems {
		case 0:
			// There is a namespace but no scenario. This may happen due to a scenario being
			// externally deleted. In this case, create a dummy object just to continue with the listing.
			var dummy v1alpha1.Scenario

			dummy.SetName("----")
			dummy.SetNamespace(namespace.GetName())
			dummy.SetCreationTimestamp(namespace.GetCreationTimestamp())

			if !namespace.GetDeletionTimestamp().IsZero() {
				dummy.SetReconcileStatus(v1alpha1.Lifecycle{
					Phase:   "Terminating",
					Reason:  "NoScenario",
					Message: "No Scenario is found in namespace, and the namespace is terminating",
				})
			} else {
				dummy.SetReconcileStatus(v1alpha1.Lifecycle{
					Phase:   "----",
					Reason:  "NoScenario",
					Message: "No Scenario is found in namespace",
				})
			}

			scenarios.Items = append(scenarios.Items, dummy)

		case 1:
			if !namespace.GetDeletionTimestamp().IsZero() { // Some rewrite for output to make more sense
				localList.Items[0].Status.Phase = "Terminating"
			}

			scenarios.Items = append(scenarios.Items, localList.Items[0])
		default:
			return v1alpha1.ScenarioList{}, errors.Errorf("test '%s' has %d scenarios", namespace.GetName(), numItems)
		}
	}

	return scenarios, nil
}

// ListVirtualObjects list all virtual objects.
func (c TestManagementClient) ListVirtualObjects(ctx context.Context, namespace string, selectors ...string) (list v1alpha1.VirtualObjectList, err error) {
	var filter client.ListOptions
	filter.Namespace = namespace

	if selectors != nil {
		set, err := labels.ConvertSelectorToLabelsMap(strings.Join(selectors, ","))
		if err != nil {
			return v1alpha1.VirtualObjectList{}, errors.Wrapf(err, "invalid selector")
		}

		// find namespaces where tests are running
		filter.LabelSelector = labels.SelectorFromValidatedSet(set)
	}

	if err = c.client.List(ctx, &list, &filter); err != nil {
		return v1alpha1.VirtualObjectList{}, errors.Wrapf(err, "cannot list resources")
	}

	return list, err
}

// ListServices list all services.
func (c TestManagementClient) ListServices(ctx context.Context, namespace string, selectors ...string) (list v1alpha1.ServiceList, err error) {
	var filter client.ListOptions
	filter.Namespace = namespace

	if selectors != nil {
		set, err := labels.ConvertSelectorToLabelsMap(strings.Join(selectors, ","))
		if err != nil {
			return v1alpha1.ServiceList{}, errors.Wrapf(err, "invalid selector")
		}

		// find namespaces where tests are running
		filter.LabelSelector = labels.SelectorFromValidatedSet(set)
	}

	if err = c.client.List(ctx, &list, &filter); err != nil {
		return v1alpha1.ServiceList{}, errors.Wrapf(err, "cannot list resources")
	}

	return list, err
}
