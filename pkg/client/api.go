/*
Copyright 2022 ICS-FORTH.

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
	frisbeev1alpha1 "github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/pkg/kubexec"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// check in compile time if interface is implemented
// var _ Client = (*APIClient)(nil)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(frisbeev1alpha1.AddToScheme(scheme))
}

// NewDirectAPIClient returns proxy api client.
func NewDirectAPIClient(client client.Client) APIClient {
	return APIClient{
		TestManagementClient: NewTestManagementClient(client),
		TestInspectionClient: NewTestInspectionClient(client, kubexec.NewExecutor(controllerruntime.GetConfigOrDie())),
	}
}

// APIClient struct managing proxy API Client dependencies.
type APIClient struct {
	TestManagementClient
	TestInspectionClient
	// TestSuiteClient
	// ExecutorClient
	// WebhookClient
}
