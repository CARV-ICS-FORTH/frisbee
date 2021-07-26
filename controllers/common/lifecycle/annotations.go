package lifecycle

import (
	"fmt"
	"reflect"

	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/grafana-tools/sdk"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Annotator provides a way to mark points on the graph with rich events.
// // When you hover over an annotation you can get event description and event tags.
// // The text field can include links to other systems with more detail.
type Annotator interface {
	// Add  pushes an annotation to grafana indicating that a new component has joined the experiment.
	Add(obj interface{})

	// Delete pushes an annotation to grafana indicating that a new component has left the experiment.
	Delete(obj interface{})
}

/////////////////////////////////////////////
//		Point Annotator
/////////////////////////////////////////////

type PointAnnotation struct{}

func (a *PointAnnotation) Add(obj interface{}) {
	objMeta, ok := obj.(metav1.Object)
	if !ok {
		panic("this should never happen")
	}

	ga := sdk.CreateAnnotationRequest{
		Time: objMeta.GetCreationTimestamp().Unix() * 1000, // unix ts in ms
		Tags: []string{"run"},
		Text: fmt.Sprintf("Child added. Kind:%s Name:%s", reflect.TypeOf(obj), objMeta.GetName()),
	}

	common.Common.Annotator.Insert(ga)
}

func (a *PointAnnotation) Delete(obj interface{}) {
	objMeta, ok := obj.(metav1.Object)
	if !ok {
		panic("this should never happen")
	}

	ga := sdk.CreateAnnotationRequest{
		Time: objMeta.GetDeletionTimestamp().Unix() * 1000, // unix ts in ms
		Tags: []string{"exit"},
		Text: fmt.Sprintf("Child Deleted. Kind:%s Name:%s ", reflect.TypeOf(obj), objMeta.GetName()),
	}

	common.Common.Annotator.Insert(ga)
}

/////////////////////////////////////////////
//		Range Annotator
/////////////////////////////////////////////

// RangeAnnotation uses range annotations to indicate the duration of a Chaos.
// It consists of two parts. In the first part, a failure annotation is created
// with open end. When a new value is pushed to the timeEnd channel, the annotation is updated
// accordingly. TimeEnd channel can be used as many time as wished. The client is responsible to close the channel.
type RangeAnnotation struct {
	// Currently the Annotator works for a single wtached object. If we want to support more, use a map with
	// the key being the object name.
	reqID uint
}

func (a *RangeAnnotation) Add(obj interface{}) {
	objMeta, ok := obj.(metav1.Object)
	if !ok {
		panic("this should never happen")
	}

	ga := sdk.CreateAnnotationRequest{
		Time:    objMeta.GetCreationTimestamp().Unix() * 1000, // unix ts in ms
		TimeEnd: 0,
		Tags:    []string{"failure"},
		Text:    fmt.Sprintf("Chaos injected. Kind:%s Name:%s", reflect.TypeOf(obj), objMeta.GetName()),
	}

	a.reqID = common.Common.Annotator.Insert(ga)
}

func (a *RangeAnnotation) Delete(obj interface{}) {
	objMeta, ok := obj.(metav1.Object)
	if !ok {
		panic("this should never happen")
	}

	ga := sdk.PatchAnnotationRequest{
		Time:    objMeta.GetCreationTimestamp().Unix() * 1000, // unix ts in ms
		TimeEnd: objMeta.GetDeletionTimestamp().Unix() * 1000,
		Tags:    []string{"failure"},
		Text:    fmt.Sprintf("Chaos revoked. Kind:%s Name:%s", reflect.TypeOf(obj), objMeta.GetName()),
	}

	common.Common.Annotator.Patch(a.reqID, ga)
}
