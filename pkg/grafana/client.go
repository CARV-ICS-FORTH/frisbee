/*
Copyright 2021-2023 ICS-FORTH.

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
	"fmt"

	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/go-logr/logr"
	notifier "github.com/golanghelper/grafana-webhook"
	"github.com/grafana-tools/sdk"
	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

type Options struct {
	WebhookURL *string

	EventDispatcher func(b *notifier.Body)

	RegisterFor metav1.Object

	Logger logr.Logger

	HTTPEndpoint *string
}

type Option func(*Options)

// WithNotifications will update the object's annotations if a Grafana alert is triggered.
func WithNotifications(webhookURL string) Option {
	return func(args *Options) {
		args.WebhookURL = &webhookURL
	}
}

// WithRegisterFor will register the client with the given name. Registered clients are retrievable by GetFrisbeeClient().
func WithRegisterFor(obj metav1.Object) Option {
	return func(args *Options) {
		args.RegisterFor = obj
	}
}

// WithLogger will use the given logger for printing info.
func WithLogger(logger logr.Logger) Option {
	return func(args *Options) {
		if logger == (logr.Logger{}) {
			panic("trying to pass empty logger")
		}

		args.Logger = logger
	}
}

// WithHTTP will use HTTP for connection with Grafana.
func WithHTTP(endpoint string) Option {
	return func(args *Options) {
		httpEndpoint := fmt.Sprintf("http://%s", endpoint)

		args.HTTPEndpoint = &httpEndpoint
	}
}

type Client struct {
	logger logr.Logger

	Conn *sdk.Client

	GapiClient *gapi.Client

	BaseURL string
}

func New(parentCtx context.Context, setters ...Option) (*Client, error) {
	var args Options

	for _, setter := range setters {
		setter(&args)
	}

	client := &Client{}

	if args.Logger == (logr.Logger{}) {
		client.logger = defaultLogger
	} else {
		client.logger = args.Logger
	}

	// connect the controller to Grafana for pushing annotations.
	if args.HTTPEndpoint != nil {
		client.logger.Info("Connecting to Grafana ...", "endpoint", *args.HTTPEndpoint)

		conn, err := sdk.NewClient(*args.HTTPEndpoint, "", sdk.DefaultHTTPClient)
		if err != nil {
			return nil, errors.Wrapf(err, "client error")
		}

		retryCond := func(ctx context.Context) (done bool, err error) {
			resp, err := conn.GetHealth(ctx)
			// Retry
			if err != nil {
				client.logger.Info("Retry to connect to grafana", "Error", err)

				return false, nil
			}

			// Retry
			if resp.Database != "ok" {
				client.logger.Info("Grafana does not seem heath. Retry")

				return false, nil
			}

			// OK
			client.logger.Info("Connected to Grafana", "healthStatus", resp)

			return true, nil
		}

		if err := wait.ExponentialBackoffWithContext(parentCtx, common.DefaultBackoffForServiceEndpoint, retryCond); err != nil {
			return nil, errors.Wrapf(err, "endpoint is unreachable ('%s')", *args.HTTPEndpoint)
		}

		client.Conn = conn
		client.BaseURL = *args.HTTPEndpoint

		// Start Gapi client
		gapiClient, err := gapi.New(*args.HTTPEndpoint, gapi.Config{})
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to initialize gapi client")
		}

		client.GapiClient = gapiClient
	}

	/*---------------------------------------------------*
	 * Set Notification channel for receiving alerts
	 *---------------------------------------------------*/
	if args.WebhookURL != nil {
		client.logger.Info("Setting Notification Channel ...", "endpoint", args.WebhookURL)

		// Although the notification channel is backed by the Grafana Pod, the Grafana Service is different
		// from the Alerting Service. For this reason, we must be sure that both Services are linked to the Grafana Pod.
		if err := client.SetNotificationChannel(parentCtx, *args.WebhookURL); err != nil {
			return nil, errors.Wrapf(err, "failed to set notification channel")
		}
	}

	/*---------------------------------------------------*
	 * Save client to a pool associated with the scenario.
	 *---------------------------------------------------*/
	if args.RegisterFor != nil {
		// associated clients can be used by GetFrisbeeClient(), ClientExistsFor()...
		SetClientFor(args.RegisterFor, client)
	}

	return client, nil
}
