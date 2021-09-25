// Licensed to FORTH/ICS under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. FORTH/ICS licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

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
	Globals.Executor = NewExecutor(mgr.GetConfig())
}

func SetGrafana(ctx context.Context, apiURI string) error {
	grafanaClient, err := sdk.NewClient(apiURI, "", sdk.DefaultHTTPClient)
	if err != nil {
		return errors.Wrapf(err, "grafanaClient error")
	}

	// retry until Grafana is ready to receive annotations.
	err = retry.OnError(DefaultBackoff, func(_ error) bool { return true }, func() error {
		_, err := grafanaClient.GetHealth(ctx)

		return errors.Wrapf(err, "grafana health error")
	})

	if err != nil {
		return errors.Wrapf(err, "grafana is unreachable")
	}

	Globals.Annotator = &GrafanaAnnotator{
		ctx:    ctx,
		Client: grafanaClient,
	}

	return nil
}
