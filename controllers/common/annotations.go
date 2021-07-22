package common

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/grafana-tools/sdk"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var annnotator annotator

type annotator interface {
	Add(sdk.CreateAnnotationRequest)
}

func EnableAnnotations(ctx context.Context, client annotator) {
	annnotator = client
}

func annotateAdd(obj interface{}) {
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

	if annnotator == nil {
		common.logger.Info("omit annotation", "msg", msg)
	} else {
		annnotator.Add(ga)
	}
}

func annotateDelete(obj interface{}) {
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

	if annnotator == nil {
		common.logger.Info("omit annotation", "msg", msg)
	} else {
		annnotator.Add(ga)
	}
}
