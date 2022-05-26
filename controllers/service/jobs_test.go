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

package service

import (
	"testing"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func Test_setField(t *testing.T) {

	cr := v1alpha1.Service{
		Spec: v1alpha1.ServiceSpec{
			Requirements: &v1alpha1.Requirements{
				PVC: &v1alpha1.PVC{
					Name: "bind",
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: nil,
						Selector:    nil,
						Resources: corev1.ResourceRequirements{
							Requests: map[corev1.ResourceName]resource.Quantity{"storage": resource.MustParse("1Gi")},
						},
					},
				},
			},

			PodSpec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name: "test0",
						Ports: []corev1.ContainerPort{
							{
								Name:          "MyAwesomePort",
								ContainerPort: 0,
							},
						},
					},
				},
			},
		},
	}

	type args struct {
		cr  *v1alpha1.Service
		val v1alpha1.SetField
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test-port",
			args: args{
				cr: &cr,
				val: v1alpha1.SetField{
					Field: "PodSpec.Containers.0.Ports.0.ContainerPort",
					Value: "66",
				},
			},
			wantErr: false,
		},
		{
			name: "test-requirement",
			args: args{
				cr: &cr,
				val: v1alpha1.SetField{
					Field: "Requirements.PVC.Spec.Resources.Requests.storage",
					Value: "3Gi",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := setField(tt.args.cr, tt.args.val); (err != nil) != tt.wantErr {
				t.Errorf("setField() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
