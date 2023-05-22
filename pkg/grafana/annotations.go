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

package grafana

import (
	"fmt"
	"reflect"
	"time"

	"github.com/grafana-tools/sdk"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func tracingMsg(name string, kind string) string {
	return fmt.Sprintf("%s (%s)", name, kind)
}

func AnnotatePointInTime(obj client.Object, ts time.Time, tags []Tag) {
	if len(tags) == 0 {
		panic("empty tag list")
	}

	// if possible, use the native kind. Otherwise, the reflected.
	// this is needed for wrappers types such as unstructured.Unstructured.
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	if kind == "" {
		kind = reflect.TypeOf(obj).String()
	}

	if ts.IsZero() {
		defaultLogger.Info("Replace nil ts with time.Now", "obj", obj.GetName(), "kind", kind)

		ts = time.Now()
	}

	annotationRequest := sdk.CreateAnnotationRequest{
		Time:    ts.UnixMilli(),
		TimeEnd: 0,
		Tags:    tags,
		Text:    tracingMsg(obj.GetName(), kind),
	}

	GetClientFor(obj).AddAnnotation(annotationRequest)
}

func AnnotateTimerange(obj client.Object, tsStart time.Time, tsEnd time.Time, tags []Tag) {
	if len(tags) == 0 {
		panic("empty tag list")
	}

	// if possible, use the native kind. Otherwise, the reflected.
	// this is needed for wrappers types such as unstructured.Unstructured.
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	if kind == "" {
		kind = reflect.TypeOf(obj).String()
	}

	if tsStart.IsZero() {
		defaultLogger.Info("Replace nil tsStart with time.Now", "obj", obj.GetName(), "kind", kind)

		tsStart = time.Now()
	}

	if tsEnd.IsZero() {
		defaultLogger.Info("Replace nil tsEnd with time.Now", "obj", obj.GetName(), "kind", kind)

		tsEnd = time.Now()
	}

	annotationRequest := sdk.CreateAnnotationRequest{
		Time:    tsStart.UnixMilli(),
		TimeEnd: tsEnd.UnixMilli(),
		Tags:    tags,
		Text:    tracingMsg(obj.GetName(), kind),
	}

	GetClientFor(obj).AddAnnotation(annotationRequest)
}
