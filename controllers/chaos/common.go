package chaos

import (
	"fmt"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/grafana-tools/sdk"
)

// AnnotateChaos uses range annotations to indicate the duration of a Chaos.
// It consists of two parts. In the first part, a failure annotation is created
// with open end. When a new value is pushed to the timeEnd channel, the annotation is updated
// accordingly. TimeEnd channel can be used as many time as wished. The client is responsible to close the channel.
func AnnotateChaos(obj *v1alpha1.Chaos) {
	// chaos injection
	if obj.Status.AnnotationID == 0 {
		ga := sdk.CreateAnnotationRequest{
			Time:    obj.GetCreationTimestamp().Unix() * 1000, // unix ts in ms
			TimeEnd: 0,
			Tags:    []string{"failure"},
			Text:    fmt.Sprintf("Chaos injected. Name:%s", obj.GetName()),
		}

		if common.Common.Annotator == nil {
			common.Common.Logger.Info("omit annotation", "msg", ga.Text)

			obj.Status.AnnotationID = 1
		} else {
			obj.Status.AnnotationID = common.Common.Annotator.Insert(ga)
		}

		return
	}

	// chaos revocation

	/*
		// this is because in some conditions a delete elements does not have a deletion timestamp.
		// in this case, just use the current time.
		ts := objMeta.GetDeletionTimestamp()
		if ts == nil {
			ts = &metav1.Time{Time: time.Now()}
		}

	*/

	ga := sdk.PatchAnnotationRequest{
		Time:    obj.GetCreationTimestamp().Unix() * 1000, // unix ts in ms
		TimeEnd: obj.GetDeletionTimestamp().Unix() * 1000,
		Tags:    []string{"failure"},
		Text:    fmt.Sprintf("Chaos revoked. Name:%s", obj.GetName()),
	}

	if common.Common.Annotator == nil {
		common.Common.Logger.Info("omit annotation patch", "msg", ga.Text)
	} else {
		common.Common.Annotator.Patch(obj.Status.AnnotationID, ga)
	}
}
