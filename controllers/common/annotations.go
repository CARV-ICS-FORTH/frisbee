package common

import (
	"context"
	"fmt"
	"reflect"

	"github.com/grafana-tools/sdk"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var grafanaClient annotationClient

type annotationClient interface {
	Add(sdk.CreateAnnotationRequest)
}

func EnableAnnotations(ctx context.Context, client annotationClient) {
	grafanaClient = client
}

func annotateAdd(obj interface{}) {
	objMeta, ok := obj.(metav1.Object)
	if !ok {
		panic("this should never happen")
	}

	msg := fmt.Sprintf("Child added. Kind:%s Name:%s ", reflect.TypeOf(obj), objMeta.GetName())

	ga := sdk.CreateAnnotationRequest{
		Time: objMeta.GetCreationTimestamp().Unix() * 1000, // unix ts in ms
		Tags: []string{"run"},
		Text: msg,
	}

	if grafanaClient == nil {
		common.logger.Info("omit annotation", "msg", msg)
	} else {
		grafanaClient.Add(ga)
	}
}

// add an annotation to grafana
func annotateDelete(obj interface{}) {
	objMeta, ok := obj.(metav1.Object)
	if !ok {
		panic("this should never happen")
	}

	msg := fmt.Sprintf("Child Deleted. Kind:%s Name:%s ", reflect.TypeOf(obj), objMeta.GetName())

	ga := sdk.CreateAnnotationRequest{
		Time: objMeta.GetDeletionTimestamp().Unix() * 1000, // unix ts in ms
		Tags: []string{"exit"},
		Text: msg,
	}

	if grafanaClient == nil {
		common.logger.Info("omit annotation", "msg", msg)
	} else {
		grafanaClient.Add(ga)
	}
}
