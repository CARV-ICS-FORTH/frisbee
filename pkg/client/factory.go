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
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClientType string

const (
	ClientDirect     ClientType = "terminal"
	ClientController ClientType = "controller"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(frisbeev1alpha1.AddToScheme(scheme))
}

// Options contains client options
type Options struct {
	//	Namespace string
}

// GetClient returns configured Frisbee API client.
func GetClient(clientType ClientType, options Options) (Client, error) {
	// Get the base Kubernetes client
	c, err := client.New(ctrl.GetConfigOrDie(), client.Options{Scheme: scheme})
	if err != nil {
		return nil, errors.Wrap(err, "cannot get client")
	}

	switch clientType {
	case ClientDirect:
		return NewDirectAPIClient(c, options), nil
	case ClientController:
		panic("not yet supported")
	default:
		return nil, errors.Errorf("unsupported client type '%s'", clientType)
	}
}
