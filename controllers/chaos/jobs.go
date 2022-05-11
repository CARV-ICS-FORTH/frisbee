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

package chaos

import (
	"context"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type Fault = unstructured.Unstructured

type chaoHandler interface {
	GetFault() Fault

	Inject(ctx context.Context, r *Controller) error
}

func dispatch(chaos *v1alpha1.Chaos) chaoHandler {
	switch chaos.Spec.Type {
	case v1alpha1.FaultRaw:
		return &rawHandler{cr: chaos}
	default:
		panic("should never happen")
	}
}

/*
const (
	TypeAWSChaos TemplateType = "AWSChaos"
	TypeAzureChaos TemplateType = "AzureChaos"
	TypeBlockChaos TemplateType = "BlockChaos"
	TypeDNSChaos TemplateType = "DNSChaos"
	TypeGCPChaos TemplateType = "GCPChaos"
	TypeHTTPChaos TemplateType = "HTTPChaos"
	TypeIOChaos TemplateType = "IOChaos"
	TypeJVMChaos TemplateType = "JVMChaos"
	TypeKernelChaos TemplateType = "KernelChaos"
	TypeNetworkChaos TemplateType = "NetworkChaos"
	TypePhysicalMachineChaos TemplateType = "PhysicalMachineChaos"
	TypePodChaos TemplateType = "PodChaos"
	TypeStressChaos TemplateType = "StressChaos"
	TypeTimeChaos TemplateType = "TimeChaos"
)

*/

var (
	NetworkChaosGVK = schema.GroupVersionKind{
		Group:   "chaos-mesh.org",
		Version: "v1alpha1",
		Kind:    "NetworkChaos",
	}

	PodChaosGVK = schema.GroupVersionKind{
		Group:   "chaos-mesh.org",
		Version: "v1alpha1",
		Kind:    "PodChaos",
	}

	/*
		BlockChaosGVK = schema.GroupVersionKind{
			Group:   "chaos-mesh.org",
			Version: "v1alpha1",
			Kind:    "BlockChaos",
		}

	*/

	IOChaosGVK = schema.GroupVersionKind{
		Group:   "chaos-mesh.org",
		Version: "v1alpha1",
		Kind:    "IOChaos",
	}

	KernelChaosGVK = schema.GroupVersionKind{
		Group:   "chaos-mesh.org",
		Version: "v1alpha1",
		Kind:    "KernelChaos",
	}

	TimeChaosGVK = schema.GroupVersionKind{
		Group:   "chaos-mesh.org",
		Version: "v1alpha1",
		Kind:    "TimeChaos",
	}
)

/*
	Raw Fault Handler
*/
type rawHandler struct {
	cr *v1alpha1.Chaos
}

func (h rawHandler) GetFault() Fault {
	var f map[string]interface{}

	if err := yaml.Unmarshal([]byte(*h.cr.Spec.Raw), &f); err != nil {
		panic(err)
	}

	var fault Fault

	fault.SetUnstructuredContent(f)

	fault.SetName(h.cr.GetName())
	fault.SetNamespace(h.cr.GetNamespace())

	return fault
}

func (h rawHandler) Inject(ctx context.Context, r *Controller) error {
	fault := h.GetFault()

	logrus.Warn("INJECT FAULT ", fault)

	if err := utils.Create(ctx, r, h.cr, &fault); err != nil {
		return errors.Wrapf(err, "cannot inject fault")
	}

	return nil
}
