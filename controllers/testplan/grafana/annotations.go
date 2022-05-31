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
	"reflect"
	"time"

	"github.com/grafana-tools/sdk"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// DrawAs hints how to mark points on the Grafana dashboard.
	DrawAs string = "grafana.frisbee.io/draw"
	// DrawAsPoint will mark the creation and deletion of a service as distinct events.
	DrawAsPoint string = "point"
	// DrawAsRegion will draw a region starting from the creation of a service and ending to the deletion of the service.
	DrawAsRegion string = "range"
)

// ///////////////////////////////////////////
//		Grafana Annotations
// ///////////////////////////////////////////

type Tag = string

const (
	TagRun     = "run"
	TagExit    = "exit"
	TagFailure = "failure"
)

// Annotation provides a way to mark points on the graph with rich events.
// // Conditions you hover over an annotation you can get event description and event tags.
// // The text field can include links to other systems with more detail.
type Annotation interface {
	// Add  pushes an annotation to grafana indicating that a new component has joined the experiment.
	Add(obj client.Object)

	// Delete pushes an annotation to grafana indicating that a new component has left the experiment.
	Delete(obj client.Object)
}

type PointAnnotation struct{}

func (a *PointAnnotation) Add(obj client.Object) {
	ga := sdk.CreateAnnotationRequest{
		Time:    obj.GetCreationTimestamp().UnixMilli(),
		TimeEnd: 0,
		Tags:    []string{TagRun},
		Text:    fmt.Sprintf("Job added. Kind:%s Name:%s", reflect.TypeOf(obj), obj.GetName()),
	}

	if ClientExistsFor(obj) {
		_ = GetClientFor(obj).SetAnnotation(ga)
	}
}

func (a *PointAnnotation) Delete(obj client.Object) {
	ga := sdk.CreateAnnotationRequest{
		Time:    obj.GetDeletionTimestamp().UnixMilli(),
		TimeEnd: 0,
		Tags:    []string{TagExit},
		Text:    fmt.Sprintf("Job Deleted. Kind:%s Name:%s", reflect.TypeOf(obj), obj.GetName()),
	}

	if ClientExistsFor(obj) {
		_ = GetClientFor(obj).SetAnnotation(ga)
	}
}

// RangeAnnotation uses range annotations to indicate the duration of a Chaos.
// It consists of two parts. In the first part, a failure annotation is created
// with open end. Conditions a new value is pushed to the timeEnd channel, the annotation is updated
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

	ga := sdk.CreateAnnotationRequest{
		Time:    obj.GetCreationTimestamp().UnixMilli(),
		TimeEnd: 0,
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

var annotationTimeout = 30 * time.Second

const (
	statusAnnotationAdded = "Annotation added"

	statusAnnotationPatched = "Annotation patched"
)

// SetAnnotation inserts a new annotation to Grafana.
func (c *Client) SetAnnotation(ga sdk.CreateAnnotationRequest) (reqID uint) {
	ctx, cancel := context.WithTimeout(c.ctx, annotationTimeout)
	defer cancel()

	// submit
	gaResp, err := c.Conn.CreateAnnotation(ctx, ga)
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "annotation failed"))

		return
	}

	if gaResp.Message == nil {
		runtime.HandleError(errors.Wrapf(err, "empty annotation response"))
	} else if *gaResp.Message != statusAnnotationAdded {
		runtime.HandleError(errors.Wrapf(err, "expected message '%s', but got '%s'",
			statusAnnotationAdded, *gaResp.Message))
	}

	return *gaResp.ID
}

// PatchAnnotation updates an existing annotation to Grafana.
func (c *Client) PatchAnnotation(reqID uint, ga sdk.PatchAnnotationRequest) {
	ctx, cancel := context.WithTimeout(c.ctx, annotationTimeout)
	defer cancel()

	// submit
	gaResp, err := c.Conn.PatchAnnotation(ctx, reqID, ga)
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "annotation failed"))

		return
	}

	if gaResp.Message == nil {
		runtime.HandleError(errors.Wrapf(err, "empty annotation response"))
	} else if *gaResp.Message != statusAnnotationPatched {
		runtime.HandleError(errors.Wrapf(err, "expected message '%s', but got '%s'",
			statusAnnotationPatched, *gaResp.Message))
	}
}
