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

import (
	"sort"

	"github.com/carv-ics-forth/frisbee/pkg/structure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VirtualObject is a CRD without a dedicated controller. Practically, it is just an entry in the Kubernetes API
// that is used as placeholder for action like Delete and Call.
type VirtualObject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualObjectSpec   `json:"spec,omitempty"`
	Status VirtualObjectStatus `json:"status,omitempty"`
}

func (in *VirtualObject) Table() (header []string, data [][]string) {
	statusHeader, statusData := in.Status.Table()

	header = []string{"Name", "Phase"}
	header = append(header, statusHeader...)

	data = [][]string{{in.GetName(), in.Status.Phase.String()}}
	data[0] = append(data[0], statusData[0]...)

	return
}

type VirtualObjectSpec struct{}

type VirtualObjectStatus struct {
	Lifecycle `json:",inline"`

	// Data contains the configuration data.
	// Each key must consist of alphanumeric characters, '-', '_' or '.'.
	// Values with non-UTF-8 byte sequences must use the BinaryData field.
	// The keys stored in Data must not overlap with the keys in
	// the BinaryData field, this is enforced during validation process.
	// +optional
	Data map[string]string `json:"data,omitempty"`
}

func (in *VirtualObjectStatus) Table() (header []string, data [][]string) {
	values := make([]string, len(in.Data))

	for _, key := range structure.SortedMapKeys(in.Data) {
		header = append(header, key)

		// encode it to escape newlines and all that stuff that destroy the nice printing.
		encValue, _ := json.Marshal(in.Data[key])

		values = append(values, string(encValue))
	}

	data = append(data, values)

	return
}

func (in *VirtualObject) GetReconcileStatus() Lifecycle {
	return in.Status.Lifecycle
}

func (in *VirtualObject) SetReconcileStatus(lifecycle Lifecycle) {
	in.Status.Lifecycle = lifecycle
}

// +kubebuilder:object:root=true

// VirtualObjectList contains a list of Virtual Objects.
type VirtualObjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualObject `json:"items"`
}

func (in *VirtualObjectList) Table() (header []string, data [][]string) {
	// arrange in descending order (latest created goes first)
	sort.SliceStable(in.Items, func(i, j int) bool {
		tsI := in.Items[i].GetCreationTimestamp()
		tsJ := in.Items[j].GetCreationTimestamp()

		return tsI.After(tsJ.Time)
	})

	for _, vObject := range in.Items {
		vHeaders, vData := vObject.Table()

		// ignore duplicate headers
		for _, vHeader := range vHeaders {
			if !structure.ContainsStrings(header, vHeader) {
				header = append(header, vHeader)
			}
		}

		// but always add data to an existing header
		data = append(data, vData...)
	}

	return header, data
}

func init() {
	SchemeBuilder.Register(&VirtualObject{}, &VirtualObjectList{})
}
