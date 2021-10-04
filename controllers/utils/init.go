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

package utils

import (
	"context"
	"time"

	"github.com/grafana-tools/sdk"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
)

var HealthCheckTimeout = wait.Backoff{
	Duration: 5 * time.Second,
	Factor:   5,
	Jitter:   0.1,
	Steps:    4,
}

// Annotate pushes annotation for evenete
var Annotate *GrafanaAnnotator

func SetGrafana(ctx context.Context, apiURI string) error {
	grafanaClient, err := sdk.NewClient(apiURI, "", sdk.DefaultHTTPClient)
	if err != nil {
		return errors.Wrapf(err, "grafanaClient error")
	}

	// retry until Grafana is ready to receive annotations.
	err = retry.OnError(HealthCheckTimeout, func(_ error) bool { return true }, func() error {
		_, err := grafanaClient.GetHealth(ctx)

		return errors.Wrapf(err, "grafana health error")
	})

	if err != nil {
		return errors.Wrapf(err, "grafana is unreachable")
	}

	Annotate = &GrafanaAnnotator{
		ctx:    ctx,
		Client: grafanaClient,
	}

	return nil
}
