/*
Copyright 2021 ICS-FORTH.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package grafana

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	notifier "github.com/golanghelper/grafana-webhook"
	"github.com/grafana-tools/sdk"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
)

var healthCheckTimeout = wait.Backoff{
	Duration: 3 * time.Second,
	Factor:   3,
	Jitter:   0.1,
	Steps:    4,
}

type Options struct {
	NotifyOnAlert func(b *notifier.Body)
}

type Option func(*Options)

// WithNotifyOnAlert will update the object's annotations if a Grafana alert is triggered
func WithNotifyOnAlert(cb func(b *notifier.Body)) Option {
	return func(args *Options) {
		args.NotifyOnAlert = cb
	}
}

func NewGrafanaClient(ctx context.Context, r logr.Logger, apiURI string, setters ...Option) error {
	// Default Options
	args := &Options{
		NotifyOnAlert: nil,
	}

	for _, setter := range setters {
		setter(args)
	}

	conn, err := sdk.NewClient(apiURI, "", sdk.DefaultHTTPClient)
	if err != nil {
		return errors.Wrapf(err, "conn error")
	}

	// retry until Grafana is ready to receive annotations.
	err = retry.OnError(healthCheckTimeout, func(_ error) bool { return true }, func() error {
		r.Info("Connecting to Grafana", "endpoint", apiURI)

		_, err := conn.GetHealth(ctx)

		return errors.Wrapf(err, "grafana health error")
	})

	if err != nil {
		return errors.Wrapf(err, "grafana is unreachable")
	}

	client := &Client{
		ctx:    ctx,
		logger: r,
		Conn:   conn,
	}

	// Use this client as the default annotator
	DefaultClient = client

	// Set webhook for getting back grafana alerts
	if err := client.SetNotificationChannel("6666", args.NotifyOnAlert); err != nil {
		return errors.Wrapf(err, "cannot run a notification webhook")
	}

	return nil
}

var DefaultClient *Client

type Client struct {
	ctx    context.Context
	logger logr.Logger

	Conn *sdk.Client
}
