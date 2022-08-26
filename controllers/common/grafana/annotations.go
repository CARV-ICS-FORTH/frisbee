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

package grafana

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/util/wait"
	"reflect"
	"time"

	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/grafana-tools/sdk"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Tag = string

const (
	TagRun     = "run"
	TagExit    = "exit"
	TagFailure = "failure"
)

// Annotation provides a way to mark points on the graph with rich events.
// // Event you hover over an annotation you can get event description and event tags.
// // The text field can include links to other systems with more detail.
type Annotation interface {
	// Add  pushes an annotation to grafana indicating that a new component has joined the experiment.
	Add(obj client.Object)

	// Delete pushes an annotation to grafana indicating that a new component has left the experiment.
	Delete(obj client.Object)
}

type PointAnnotation struct{}

func (a *PointAnnotation) Add(obj client.Object, tags ...Tag) {
	if tags == nil {
		tags = []Tag{TagRun}
	}

	ga := sdk.CreateAnnotationRequest{
		Time:    obj.GetCreationTimestamp().UnixMilli(),
		TimeEnd: 0,
		Tags:    tags,
		Text:    fmt.Sprintf("Job added. Kind:%s Name:%s", reflect.TypeOf(obj), obj.GetName()),
	}

	if ClientExistsFor(obj) {
		_ = GetClientFor(obj).SetAnnotation(ga)
	}
}

func (a *PointAnnotation) Delete(obj client.Object, tags ...Tag) {
	if tags == nil {
		tags = []Tag{TagExit}
	}

	delTime := obj.GetDeletionTimestamp()
	if delTime == nil {
		delTime = &metav1.Time{Time: time.Now()}
	}

	ga := sdk.CreateAnnotationRequest{
		Time:    delTime.UnixMilli(),
		TimeEnd: 0,
		Tags:    tags,
		Text:    fmt.Sprintf("Job Deleted. Kind:%s Name:%s", reflect.TypeOf(obj), obj.GetName()),
	}

	if ClientExistsFor(obj) {
		_ = GetClientFor(obj).SetAnnotation(ga)
	}
}

// RangeAnnotation uses range annotations to indicate the duration of a Chaos.
// It consists of two parts. In the first part, a failure annotation is created
// with open end. Event a new value is pushed to the timeEnd channel, the annotation is updated
// accordingly. TimeEnd channel can be used as many times as wished. The client is responsible to close the channel.
type RangeAnnotation struct {
	// Currently the Annotator works for a single watched object. If we want to support more, use a map with
	// the key being the object Name.
	reqID uint
}

func (a *RangeAnnotation) Add(obj client.Object, tags ...Tag) {
	if tags == nil {
		tags = []Tag{TagRun}
	}

	// In order to make the annotation open-ended, I added a date in the future.
	// The date (January 19, 2038), is the last date that can be described by  32-bit Unix/Linux-based systems.
	// FIXME: Random guy in the future, thanks for maintaining this code.
	ga := sdk.CreateAnnotationRequest{
		Time:    obj.GetCreationTimestamp().UnixMilli(),
		TimeEnd: time.Date(2039, 1, 19, 14, 30, 45, 100, time.Local).UnixMilli(),
		Tags:    tags,
		Text:    fmt.Sprintf("Job Added. Kind:%s Name:%s", reflect.TypeOf(obj), obj.GetName()),
	}

	if ClientExistsFor(obj) {
		a.reqID = GetClientFor(obj).SetAnnotation(ga)
	}
}

func (a *RangeAnnotation) Delete(obj client.Object, tags ...Tag) {
	if tags == nil {
		tags = []Tag{TagExit}
	}

	timeStart := obj.GetCreationTimestamp()
	timeEnd := obj.GetDeletionTimestamp()

	// these nuances are caused by chaos partition events whose deletion time is the same as creation time.
	// the result is that the ending line overlaps with the starting line, thus giving the feel of a period.
	if timeEnd.IsZero() || timeEnd.Equal(&timeStart) {
		timeEnd = &metav1.Time{Time: time.Now()}
	}

	ga := sdk.PatchAnnotationRequest{
		Time:    0,
		TimeEnd: timeEnd.UnixMilli(),
		Tags:    tags,
		Text:    fmt.Sprintf("Job Deleted. Kind:%s Name:%s", reflect.TypeOf(obj), obj.GetName()),
	}

	if ClientExistsFor(obj) {
		GetClientFor(obj).PatchAnnotation(a.reqID, ga)
	}
}

// ///////////////////////////////////////////
//		Grafana Annotator
// ///////////////////////////////////////////

const (
	statusAnnotationAdded = "Annotation added"

	statusAnnotationPatched = "Annotation patched"
)

// SetAnnotation inserts a new annotation to Grafana.
func (c *Client) SetAnnotation(ga sdk.CreateAnnotationRequest) (reqID uint) {
	ctx := context.Background()

	// retry until Grafana is ready to receive annotations.
	if err := wait.ExponentialBackoff(common.BackoffForServiceEndpoint, func() (done bool, err error) {
		gaResp, err := c.Conn.CreateAnnotation(ctx, ga)
		switch {
		case err != nil: // API connection error. Just retry
			return false, nil
		case gaResp.Message == nil: // Server error. Abort
			return false, errors.Errorf("empty annotation response")
		case *gaResp.Message != statusAnnotationAdded: // Unexpected response
			return false, errors.Errorf("expected message '%s', but got '%s'", statusAnnotationAdded, *gaResp.Message)
		default: // Done
			reqID = *gaResp.ID
			return true, nil
		}
	}); err != nil {
		c.logger.Error(err, "AnnotationError", "operation", "Set", "request", ga)
	}

	return reqID
}

// PatchAnnotation updates an existing annotation to Grafana.
func (c *Client) PatchAnnotation(reqID uint, ga sdk.PatchAnnotationRequest) {
	ctx := context.Background()

	// retry until Grafana is ready to receive annotations.
	if err := wait.ExponentialBackoffWithContext(ctx, common.BackoffForServiceEndpoint, func() (done bool, err error) {
		gaResp, err := c.Conn.PatchAnnotation(ctx, reqID, ga)
		switch {
		case err != nil: // API connection error. Just retry
			return false, nil
		case gaResp.Message == nil: // Server error. Abort
			return false, errors.Errorf("empty annotation response")
		case *gaResp.Message != statusAnnotationPatched: // Unexpected response
			return false, errors.Errorf("expected message '%s', but got '%s'", statusAnnotationPatched, *gaResp.Message)
		default: // Done
			return true, nil
		}
	}); err != nil {
		c.logger.Error(err, "AnnotationError", "operation", "Patch", "request", ga)
	}
}
