package common

import (
	"context"
	"math/rand"

	"github.com/grafana-tools/sdk"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
)

type Annotator interface {
	Insert(sdk.CreateAnnotationRequest) (id uint)
	Patch(reqID uint, ga sdk.PatchAnnotationRequest)
}

// ///////////////////////////////////////////
//		Logger Annotator
// ///////////////////////////////////////////

type DefaultAnnotator struct{}

func (a *DefaultAnnotator) Insert(ga sdk.CreateAnnotationRequest) (id uint) {
	Globals.Logger.Info("omit annotation", "msg", ga.Text)

	return uint(rand.Uint64())
}

func (a *DefaultAnnotator) Patch(_ uint, ga sdk.PatchAnnotationRequest) {
	Globals.Logger.Info("omit patch annotation", "msg", ga.Text)
}

// ///////////////////////////////////////////
//		Grafana Annotator
// ///////////////////////////////////////////

const (
	statusAnnotationAdded = "Annotation added"

	statusAnnotationPatched = "Annotation patched"
)

type GrafanaAnnotator struct {
	ctx context.Context

	*sdk.Client
}

// Insert inserts a new annotation to Grafana
func (c *GrafanaAnnotator) Insert(ga sdk.CreateAnnotationRequest) (reqID uint) {
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()

	// submit
	gaResp, err := c.Client.CreateAnnotation(ctx, ga)
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "annotation failed"))

		return
	}

	if gaResp.Message == nil {
		runtime.HandleError(errors.Wrapf(err, "empty annotation response"))
	} else if *gaResp.Message != string(statusAnnotationAdded) {
		runtime.HandleError(errors.Wrapf(err, "expected message '%s', but got '%s'", statusAnnotationAdded, *gaResp.Message))
	}

	return *gaResp.ID
}

// Patch updates an existing annotation to Grafana
func (c *GrafanaAnnotator) Patch(reqID uint, ga sdk.PatchAnnotationRequest) {
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()

	// submit
	gaResp, err := c.Client.PatchAnnotation(ctx, reqID, ga)
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "annotation failed"))

		return
	}

	if gaResp.Message == nil {
		runtime.HandleError(errors.Wrapf(err, "empty annotation response"))
	} else if *gaResp.Message != string(statusAnnotationPatched) {
		runtime.HandleError(errors.Wrapf(err, "expected message '%s', but got '%s'", statusAnnotationPatched, *gaResp.Message))
	}
}
