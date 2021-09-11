package common

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/grafana-tools/sdk"
	"github.com/pkg/errors"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var Globals struct {
	cache.Cache
	client.Client
	logr.Logger

	// Annotator pushes annotation for evenete
	Annotator

	// Execute run commands within containers
	Executor

	Namespace string
}

func SetNamespace(nm string) {
	Globals.Namespace = nm
}

func SetCommon(mgr ctrl.Manager, logger logr.Logger) {
	Globals.Cache = mgr.GetCache()
	Globals.Client = mgr.GetClient()
	Globals.Logger = logger.WithName("Globals")
	Globals.Annotator = &DefaultAnnotator{}
	Globals.Executor = NewExecutor(mgr.GetConfig())
}

func SetGrafana(ctx context.Context, apiURI string) error {
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

	Globals.Annotator = &GrafanaAnnotator{
		ctx:    ctx,
		Client: client,
	}

	return nil
}
