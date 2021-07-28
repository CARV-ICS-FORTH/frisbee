package common

import (
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var Common struct {
	cache.Cache
	client.Client
	logr.Logger

	// Annotator pushes annotation for evenete
	Annotator

	// Execute run commands within containers
	Executor
}

func InitCommon(mgr ctrl.Manager, logger logr.Logger) {
	Common.Cache = mgr.GetCache()
	Common.Client = mgr.GetClient()
	Common.Logger = logger.WithName("Common")
	Common.Annotator = &DefaultAnnotator{}
	Common.Executor = NewExecutor(mgr.GetConfig())
}

var DefaultBackoff = wait.Backoff{
	Duration: 5 * time.Second,
	Factor:   5,
	Jitter:   0.1,
	Steps:    4,
}

var DefaultTimeout = 30 * time.Second

var GracefulPeriodToRun = 1 * time.Minute
