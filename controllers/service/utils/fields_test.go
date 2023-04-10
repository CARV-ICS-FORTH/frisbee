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

package utils_test

import (
	"testing"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	serviceutils "github.com/carv-ics-forth/frisbee/controllers/service/utils"
	corev1 "k8s.io/api/core/v1"
)

func TestSetField(t *testing.T) {
	cr := v1alpha1.Service{
		Spec: v1alpha1.ServiceSpec{
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
			name: "set-port",
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
			name: "set-volume",
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
			name: "no-such-field",
			args: args{
				cr: &cr,
				val: v1alpha1.SetField{
					Field: "Requirements.EphemeralVolume.Spec.Resources.Requests.storage",
					Value: "3Gi",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := serviceutils.SetField(tt.args.cr, tt.args.val); (err != nil) != tt.wantErr {
				t.Errorf("SetField() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
