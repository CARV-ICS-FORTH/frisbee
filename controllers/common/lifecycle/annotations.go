package lifecycle

import (
	"fmt"
	"reflect"
	"time"

	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/grafana-tools/sdk"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// annotateAdd pushes an annotation to grafana indicating that a new component has joined the experiment.
func annotateAdd(obj interface{}) (id uint) {
	objMeta, ok := obj.(metav1.Object)
	if !ok {
		panic("this should never happen")
	}

	msg := fmt.Sprintf("Child added. Kind:%s Name:%s", reflect.TypeOf(obj), objMeta.GetName())

	ga := sdk.CreateAnnotationRequest{
		Time: objMeta.GetCreationTimestamp().Unix() * 1000, // unix ts in ms
		Tags: []string{"run"},
		Text: msg,
	}

	if common.Common.Annotator == nil {
		common.Common.Logger.Info("omit annotation", "msg", ga.Text)
		return 0
	}

	return common.Common.Annotator.Insert(ga)
}

// annotateDelete pushes an annotation to grafana indicating that a new component has left the experiment.
func annotateDelete(obj interface{}) (id uint) {
	objMeta, ok := obj.(metav1.Object)
	if !ok {
		panic("this should never happen")
	}

	msg := fmt.Sprintf("Child Deleted. Kind:%s Name:%s ", reflect.TypeOf(obj), objMeta.GetName())

	// this is because in some conditions a delete elements does not have a deletion timestamp.
	// in this case, just use the current time.
	ts := objMeta.GetDeletionTimestamp()
	if ts == nil {
		ts = &metav1.Time{Time: time.Now()}
	}

	ga := sdk.CreateAnnotationRequest{
		Time: ts.Unix() * 1000, // unix ts in ms
		Tags: []string{"exit"},
		Text: msg,
	}

	if common.Common.Annotator == nil {
		common.Common.Logger.Info("omit annotation", "msg", ga.Text)
		return 0
	}

	return common.Common.Annotator.Insert(ga)
}
