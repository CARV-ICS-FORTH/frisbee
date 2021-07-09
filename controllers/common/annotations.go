package common

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/grafana-tools/sdk"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/runtime"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// {{{ Internal types

type response string

const (
	// grafana specific.
	statusAnnotationAdded   = response("Annotation added")
	statusAnnotationPatched = response("Annotation patched")
)

var (
	grafanaClient     *sdk.Client
	grafanaCtx        context.Context
	annotationTimeout = 30 * time.Second
)

func EnableAnnotations(ctx context.Context, apiURI string) error {
	client, err := sdk.NewClient(apiURI, "", sdk.DefaultHTTPClient)
	if err != nil {
		return err
	}

	grafanaClient = client
	grafanaCtx = ctx

	common.logger.Info("Sending annotations to ", "grafana", apiURI)

	return nil
}

func annotateAdd(obj interface{}) {
	objMeta := obj.(metav1.Object)

	msg := fmt.Sprintf("Child added. Kind:%s Name:%s ", reflect.TypeOf(obj), objMeta.GetName())

	common.logger.Info(msg)

	ga := sdk.CreateAnnotationRequest{
		Time: objMeta.GetCreationTimestamp().Unix() * 1000, // unix ts in ms
		Tags: []string{"run"},
		Text: msg,
	}

	submit(ga, statusAnnotationAdded)
}

// add an annotation to grafana
func annotateDelete(obj interface{}) {
	objMeta := obj.(metav1.Object)

	msg := fmt.Sprintf("Child Deleted. Kind:%s Name:%s ", reflect.TypeOf(obj), objMeta.GetName())

	common.logger.Info(msg)

	ga := sdk.CreateAnnotationRequest{
		Time: objMeta.GetDeletionTimestamp().Unix() * 1000, // unix ts in ms
		Tags: []string{"exit"},
		Text: msg,
	}

	submit(ga, statusAnnotationAdded)
}

func submit(ga sdk.CreateAnnotationRequest, validation response) {
	if grafanaClient == nil {
		common.logger.Info("omit annotation",
			"reason", "disabled",
			"msg", ga.Text,
		)
		return
	}

	ctx, cancel := context.WithTimeout(grafanaCtx, annotationTimeout)
	defer cancel()

	// submit
	gaResp, err := grafanaClient.CreateAnnotation(ctx, ga)
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "annotation failed"))

		return
	}

	// validate
	switch {
	case gaResp.Message == nil:
		runtime.HandleError(errors.Wrapf(err, "empty annotation response"))

	case *gaResp.Message == string(validation):
		// valid
		return

	default:
		runtime.HandleError(errors.Wrapf(err,
			"unexpected annotation response. expected %s but got %s", validation, *gaResp.Message,
		))
	}
}
