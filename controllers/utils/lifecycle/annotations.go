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

package lifecycle

import (
	"fmt"
	"time"

	"github.com/fnikolai/frisbee/controllers/utils"
	"github.com/grafana-tools/sdk"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		Time: obj.GetCreationTimestamp().Unix() * 1000, // unix ts in ms
		Tags: []string{AnnotationRun},
		Text: fmt.Sprintf("Child added. Kind:%s Name:%s",
			obj.GetObjectKind().GroupVersionKind().String(), obj.GetName()),
	}

	if utils.Globals.Annotator != nil {
		utils.Globals.Annotator.Insert(ga)
	}
}

func (a *PointAnnotation) Delete(obj client.Object) {
	ts := obj.GetDeletionTimestamp()
	if ts == nil {
		ts = &metav1.Time{Time: time.Now()}
	}

	ga := sdk.CreateAnnotationRequest{
		Time: ts.Unix() * 1000, // unix ts in ms
		Tags: []string{AnnotationExit},
		Text: fmt.Sprintf("Child Deleted. Kind:%s Name:%s",
			obj.GetObjectKind().GroupVersionKind().String(), obj.GetName()),
	}

	logrus.Warn("DELETION GA ", ga)

	if utils.Globals.Annotator != nil {
		utils.Globals.Annotator.Insert(ga)
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
		Time:    obj.GetCreationTimestamp().Unix() * 1000, // unix ts in ms
		TimeEnd: 0,
		Tags:    []string{AnnotationFailure},
		Text: fmt.Sprintf("Chaos injected. Kind:%s Name:%s",
			obj.GetObjectKind().GroupVersionKind().String(), obj.GetName()),
	}

	if utils.Globals.Annotator != nil {
		a.reqID = utils.Globals.Annotator.Insert(ga)
	}
}

func (a *RangeAnnotation) Delete(obj client.Object) {

	// in some cases the deletion timestamp is nil. If so, just use the present time.
	ts := obj.GetDeletionTimestamp()
	if ts == nil {
		ts = &metav1.Time{Time: time.Now()}
	}

	ga := sdk.PatchAnnotationRequest{
		Time:    obj.GetCreationTimestamp().Unix() * 1000, // unix ts in ms
		TimeEnd: ts.Unix() * 1000,
		Tags:    []string{AnnotationFailure},
		Text: fmt.Sprintf("Chaos revoked. Kind:%s Name:%s",
			obj.GetObjectKind().GroupVersionKind().String(), obj.GetName()),
	}

	if utils.Globals.Annotator != nil {
		utils.Globals.Annotator.Patch(a.reqID, ga)
	}
}
