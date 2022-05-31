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
	"fmt"
	"time"

	"github.com/carv-ics-forth/frisbee/controllers/utils"
	"github.com/go-logr/logr"
	notifier "github.com/golanghelper/grafana-webhook"
	"github.com/grafana-tools/sdk"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	WebhookURL *string

	EventDispatcher func(b *notifier.Body)

	RegisterFor metav1.Object

	Logger logr.Logger

	HTTPEndpoint *string
}

type Option func(*Options)

// WithNotifications will update the object's annotations if a Grafana alert is triggered
func WithNotifications(webhookURL string) Option {
	return func(args *Options) {
		args.WebhookURL = &webhookURL
	}
}

// WithRegisterFor will register the client with the given name. Registered clients are retrievable by GetClient().
func WithRegisterFor(obj metav1.Object) Option {
	return func(args *Options) {
		args.RegisterFor = obj
	}
}

// WithLogger will use the given logger for printing info.
func WithLogger(r logr.Logger) Option {
	return func(args *Options) {
		args.Logger = r
	}
}

// WithHTTP will use HTTP for connection with Grafana.
func WithHTTP(endpoint string) Option {
	return func(args *Options) {
		httpEndpoint := fmt.Sprintf("http://%s", endpoint)

		args.HTTPEndpoint = &httpEndpoint
	}
}

func New(ctx context.Context, setters ...Option) error {
	var args Options

	for _, setter := range setters {
		setter(&args)
	}

	var client Client

	client.ctx = ctx

	if args.Logger != nil {
		client.logger = args.Logger
	}

	// connect the controller to Grafana for pushing annotations.
	if args.HTTPEndpoint != nil {
		conn, err := sdk.NewClient(*args.HTTPEndpoint, "", sdk.DefaultHTTPClient)
		if err != nil {
			return errors.Wrapf(err, "conn error")
		}

		// retry until Grafana is ready to receive annotations.
		err = retry.OnError(healthCheckTimeout, func(_ error) bool { return true }, func() error {
			args.Logger.Info("Connecting to Grafana", "endpoint", *args.HTTPEndpoint)

			_, err := conn.GetHealth(ctx)

			return errors.Wrapf(err, "grafana health error")
		})

		if err != nil {
			return errors.Wrapf(err, "grafana is unreachable")
		}

		client.Conn = conn
	}

	// connect Grafana to controller for receiving alerts.
	if args.WebhookURL != nil {
		if err := client.SetNotificationChannel(*args.WebhookURL); err != nil {
			return errors.Wrapf(err, "cannot run a notification webhook")
		}

	}

	// Register the client. It will be used by GetClient(), ClientExistsFor()...
	if args.RegisterFor != nil {
		name, err := utils.ExtractPartOfLabel(args.RegisterFor)
		if err != nil {
			return errors.Wrapf(err, "cannot register client")
		}

		_, exists := clients[name]
		if exists {
			return errors.Errorf("client is already registered for '%s'", name)
		}

		clients[name] = client
	}

	return nil
}

var clients = map[string]Client{}

type Client struct {
	ctx    context.Context
	logger logr.Logger

	Conn *sdk.Client
}

// ClientExistsFor check if a client is registered for the given name. It panics if it cannot parse the object's metadata.
func ClientExistsFor(obj metav1.Object) bool {
	name, err := utils.ExtractPartOfLabel(obj)
	if err != nil {
		panic(err)
	}

	_, exists := clients[name]

	return exists
}

// GetClientFor returns the client with the given name. It panics if it cannot parse the object's metadata,
// if the client does not exist or if the client is empty.
func GetClientFor(obj metav1.Object) *Client {
	name, err := utils.ExtractPartOfLabel(obj)
	if err != nil {
		panic(err)
	}

	c, exists := clients[name]
	if !exists {
		panic(errors.Errorf("Grafana client '%s' does not exist", name))
	}

	if c == (Client{}) {
		panic(errors.Errorf("Grafana client '%s' exists but is nil", name))
	}

	return &c
}
