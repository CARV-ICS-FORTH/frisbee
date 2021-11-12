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

package utils

import (
	"context"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func MergeAnnotation(obj metav1.Object, add map[string]string) {
	old := obj.GetAnnotations()

	if old == nil {
		obj.SetAnnotations(add)

		return
	}

	for key, value := range add {
		(old)[key] = value
	}

	obj.SetAnnotations(old)
}

func PatchAnnotation(ctx context.Context, r Reconciler, cr client.Object, toAdd map[string]string) error {
	// For more examples see:
	// https://golang.hotexamples.com/examples/k8s.io.kubernetes.pkg.client.unversioned/Client/Patch/golang-client-patch-method-examples.html
	patchStruct := struct {
		Metadata struct {
			Annotations map[string]string `json:"annotations"`
		} `json:"metadata"`
	}{}

	patchStruct.Metadata.Annotations = toAdd

	patchJSON, err := json.Marshal(patchStruct)
	if err != nil {
		return errors.Wrap(err, "cannot marshal patch")
	}

	patch := client.RawPatch(types.MergePatchType, patchJSON)

	err = r.GetClient().Patch(ctx, cr, patch)

	return errors.Wrapf(err, "patching %s has failed", cr.GetName())
}
