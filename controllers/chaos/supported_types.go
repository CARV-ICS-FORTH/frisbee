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

package chaos

import (
	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
)

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

type GenericFault = unstructured.Unstructured

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

func getRawManifest(cr *v1alpha1.Chaos, f *GenericFault) error {
	var body map[string]interface{}

	if err := yaml.Unmarshal([]byte(*cr.Spec.Raw), &body); err != nil {
		return errors.Wrapf(err, "cannot unmarshal manifest")
	}

	f.SetUnstructuredContent(body)

	f.SetName(cr.GetName())
	f.SetName(cr.GetNamespace())

	return nil
}
