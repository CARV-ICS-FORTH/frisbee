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

package utils

import (
	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func AttachTestDataVolume(service *v1alpha1.Service, source *v1alpha1.TestdataVolume, useSubPath bool) {
	if source == nil {
		return
	}

	// add volume to the pod
	service.Spec.Volumes = append(service.Spec.Volumes, corev1.Volume{
		Name: source.Claim.ClaimName,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &source.Claim,
		},
	})

	subpath := ""
	if useSubPath && !source.GlobalNamespace {
		subpath = service.GetName()
	}

	// mount volume to initContainers
	for i := 0; i < len(service.Spec.InitContainers); i++ {
		service.Spec.InitContainers[i].VolumeMounts = append(service.Spec.InitContainers[i].VolumeMounts, corev1.VolumeMount{
			Name:             source.Claim.ClaimName, // Name of a Volume.
			ReadOnly:         source.Claim.ReadOnly,
			MountPath:        "/testdata", // Path within the container
			SubPath:          subpath,     //  Path within the volume
			MountPropagation: nil,
			SubPathExpr:      "",
		})
	}

	// mount volume to application containers
	for i := 0; i < len(service.Spec.Containers); i++ {
		service.Spec.Containers[i].VolumeMounts = append(service.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
			Name:             source.Claim.ClaimName, // Name of a Volume.
			ReadOnly:         source.Claim.ReadOnly,
			MountPath:        "/testdata", // Path within the container
			SubPath:          subpath,     //  Path within the volume
			MountPropagation: nil,
			SubPathExpr:      "",
		})
	}
}
