package common

import (
	"context"
	"math/rand"

	"github.com/grafana-tools/sdk"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/util/retry"
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
	Common.Logger.Info("omit annotation", "msg", ga.Text)

	return uint(rand.Uint64())
}

func (a *DefaultAnnotator) Patch(_ uint, ga sdk.PatchAnnotationRequest) {
	Common.Logger.Info("omit patch annotation", "msg", ga.Text)
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

func EnableGrafanaAnnotator(ctx context.Context, apiURI string) error {
	client, err := sdk.NewClient(apiURI, "", sdk.DefaultHTTPClient)
	if err != nil {
		return errors.Wrapf(err, "client error")
	}

	// retry until Grafana is ready to receive annotations.
	err = retry.OnError(DefaultBackoff, func(_ error) bool { return true }, func() error {
		_, err := client.GetHealth(ctx)
		return errors.Wrapf(err, "grafana health error")
	})

	if err != nil {
		return errors.Wrapf(err, "grafana is unreachable")
	}

	Common.Annotator = &GrafanaAnnotator{
		ctx:    ctx,
		Client: client,
	}

	return nil
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
