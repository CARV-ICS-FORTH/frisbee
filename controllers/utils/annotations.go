// Licensed to FORTH/ICS under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. FORTH/ICS licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package utils

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
	AnnotationRun     = "run"
	AnnotationExit    = "exit"
	AnnotationFailure = "failure"
)

// Annotator provides a way to mark points on the graph with rich events.
// // When you hover over an annotation you can get event description and event tags.
// // The text field can include links to other systems with more detail.
type Annotator interface {
	// Add  pushes an annotation to grafana indicating that a new component has joined the experiment.
	Add(obj client.Object)

	// Delete pushes an annotation to grafana indicating that a new component has left the experiment.
	Delete(obj client.Object)
}

// ///////////////////////////////////////////
//		Point Annotator
// ///////////////////////////////////////////

type PointAnnotation struct{}

func (a *PointAnnotation) Add(obj client.Object) {
	ga := sdk.CreateAnnotationRequest{
		Time:    obj.GetCreationTimestamp().UnixMilli(),
		TimeEnd: 0,
		Tags:    []string{AnnotationRun},
		Text:    fmt.Sprintf("Job added. Kind:%s Name:%s", reflect.TypeOf(obj), obj.GetName()),
	}

	if v := Annotate; v != nil {
		v.Insert(ga)
	}
}

func (a *PointAnnotation) Delete(obj client.Object) {
	ga := sdk.CreateAnnotationRequest{
		Time:    obj.GetDeletionTimestamp().UnixMilli(),
		TimeEnd: 0,
		Tags:    []string{AnnotationExit},
		Text:    fmt.Sprintf("Job Deleted. Kind:%s Name:%s", reflect.TypeOf(obj), obj.GetName()),
	}

	if v := Annotate; v != nil {
		v.Insert(ga)
	}
}

// ///////////////////////////////////////////
//		Range Annotator
// ///////////////////////////////////////////

// RangeAnnotation uses range annotations to indicate the duration of a Chaos.
// It consists of two parts. In the first part, a failure annotation is created
// with open end. When a new value is pushed to the timeEnd channel, the annotation is updated
// accordingly. TimeEnd channel can be used as many times as wished. The client is responsible to close the channel.
type RangeAnnotation struct {
	// Currently the Annotator works for a single watched object. If we want to support more, use a map with
	// the key being the object Name.
	reqID uint
}

func (a *RangeAnnotation) Add(obj client.Object) {
	ga := sdk.CreateAnnotationRequest{
		Time:    obj.GetCreationTimestamp().UnixMilli(),
		TimeEnd: 0,
		Tags:    []string{AnnotationFailure},
		Text:    fmt.Sprintf("Job Added. Kind:%s Name:%s", reflect.TypeOf(obj), obj.GetName()),
	}

	if v := Annotate; v != nil {
		a.reqID = v.Insert(ga)
		v.Insert(ga)
	}
}

func (a *RangeAnnotation) Delete(obj client.Object) {
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
		Tags:    []string{AnnotationFailure},
		Text:    fmt.Sprintf("Job Deleted. Kind:%s Name:%s", reflect.TypeOf(obj), obj.GetName()),
	}

	if v := Annotate; v != nil {
		v.Patch(a.reqID, ga)
	}
}

// ///////////////////////////////////////////
//		Grafana Annotator
// ///////////////////////////////////////////

var AnnotationTimeout = 30 * time.Second

const (
	statusAnnotationAdded = "Annotation added"

	statusAnnotationPatched = "Annotation patched"
)

type GrafanaAnnotator struct {
	ctx context.Context

	*sdk.Client
}

// Insert inserts a new annotation to Grafana.
func (c *GrafanaAnnotator) Insert(ga sdk.CreateAnnotationRequest) (reqID uint) {
	ctx, cancel := context.WithTimeout(c.ctx, AnnotationTimeout)
	defer cancel()

	// submit
	gaResp, err := c.Client.CreateAnnotation(ctx, ga)
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

// Patch updates an existing annotation to Grafana.
func (c *GrafanaAnnotator) Patch(reqID uint, ga sdk.PatchAnnotationRequest) {
	ctx, cancel := context.WithTimeout(c.ctx, AnnotationTimeout)
	defer cancel()

	// submit
	gaResp, err := c.Client.PatchAnnotation(ctx, reqID, ga)
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
