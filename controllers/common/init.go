package common

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/grafana-tools/sdk"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var Common struct {
	cache.Cache
	client.Client
	logr.Logger

	Annotator
}

func InitCommon(mgr ctrl.Manager, logger logr.Logger) {
	Common.Cache = mgr.GetCache()
	Common.Client = mgr.GetClient()
	Common.Logger = logger.WithName("Common")
}

type Annotator interface {
	Insert(sdk.CreateAnnotationRequest) (id uint)
	Patch(reqID uint, ga sdk.PatchAnnotationRequest) (id uint)
}

func EnableAnnotations(ctx context.Context, annotator Annotator) {
	Common.Annotator = annotator
}

var DefaultBackoff = wait.Backoff{
	Duration: 5 * time.Second,
	Factor:   5,
	Jitter:   0.1,
	Steps:    4,
}

var DefaultTimeout = 30 * time.Second

var GracefulPeriodToRun = 1 * time.Minute
