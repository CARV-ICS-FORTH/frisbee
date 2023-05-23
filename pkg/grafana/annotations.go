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
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/grafana-tools/sdk"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"
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

	go GetClientFor(obj).AddAnnotation(annotationRequest)
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

	go GetClientFor(obj).AddAnnotation(annotationRequest)
}

// AddAnnotation inserts a new annotation to Grafana.
func (c *Client) AddAnnotation(annotationRequest sdk.CreateAnnotationRequest) (reqID uint) {
	if c == nil {
		defaultLogger.Info("NilGrafanaClient", "operation", "Set", "request", annotationRequest)

		return 0
	}

	/*---------------------------------------------------*
	 * Set the retry logic
	 *---------------------------------------------------*/
	retryCond := func(ctx context.Context) (done bool, err error) {
		response, err := c.Conn.CreateAnnotation(context.Background(), annotationRequest)
		// Retry
		if err != nil {
			defaultLogger.Info("Connection error. Retry", "annotation", annotationRequest, "Error", err.Error())

			return false, nil
		}

		// Retry
		if response.Message == nil {
			defaultLogger.Info("Empty response. Retry", "annotation", annotationRequest)

			return false, nil
		}

		// Response Status
		switch *response.Message {
		case respAnnotationAddOK:
			// OK
			reqID = *response.ID

			defaultLogger.Info("Annotation Added", "reqID", reqID, "annotation", annotationRequest)

			return true, nil
		case respAnnotationAddError, respUnauthorizedError:
			// Retry
			defaultLogger.Info("AddError. Retry", "annotation", annotationRequest, "response", response)

			return false, nil
		default:
			// Abort
			return false, errors.Errorf("unexpected response message [%s]", *response.Message)
		}
	}

	/*---------------------------------------------------*
	 * Invoke the synchronous retry mechanism
	 *---------------------------------------------------*/
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	if err := wait.ExponentialBackoffWithContext(ctx, common.DefaultBackoffForServiceEndpoint, retryCond); err != nil {
		defaultLogger.Info("AddAnnotationError", "request", annotationRequest, "err", err.Error())

		return 0
	}

	return reqID
}

// PatchAnnotation updates an existing annotation to Grafana.
func (c *Client) PatchAnnotation(reqID uint, annotationRequest sdk.PatchAnnotationRequest) {
	if c == nil {
		defaultLogger.Info("NilGrafanaClient", "operation", "Patch", "request", annotationRequest)

		return
	}

	/*---------------------------------------------------*
	 * Set the retry logic
	 *---------------------------------------------------*/
	retryCond := func(ctx context.Context) (done bool, err error) {
		response, err := c.Conn.PatchAnnotation(context.Background(), reqID, annotationRequest)
		// Retry
		if err != nil {
			defaultLogger.Info("Connection error. Retry", "reqID", reqID, "annotation", annotationRequest, "Error", err.Error())

			return false, nil
		}

		// Retry
		if response.Message == nil {
			defaultLogger.Info("Empty response. Retry", "reqID", reqID, "annotation", annotationRequest)

			return false, nil
		}

		// Response Status
		switch *response.Message {
		case respAnnotationPatchOK:
			// OK
			defaultLogger.Info("Annotation Patched", "reqID", reqID, "annotation", annotationRequest)

			return true, nil
		case respAnnotationPatchError, respUnauthorizedError:
			// Retry
			defaultLogger.Info("PatchError. Retry", "reqID", reqID, "annotation", annotationRequest, "response", response)

			return false, nil
		default:
			// Abort
			return false, errors.Errorf("unexpected response message [%s]", *response.Message)
		}
	}

	/*---------------------------------------------------*
	 * Invoke the synchronous retry mechanism
	 *---------------------------------------------------*/
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	if err := wait.ExponentialBackoffWithContext(ctx, common.DefaultBackoffForServiceEndpoint, retryCond); err != nil {
		defaultLogger.Info("PatchAnnotationError", "request", annotationRequest, "err", err.Error())
	}
}
