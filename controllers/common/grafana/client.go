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
	"sync"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/go-logr/logr"
	notifier "github.com/golanghelper/grafana-webhook"
	"github.com/grafana-tools/sdk"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Options struct {
	WebhookURL *string

	EventDispatcher func(b *notifier.Body)

	RegisterFor metav1.Object

	Logger *logr.Logger

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
func WithLogger(r *logr.Logger) Option {
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

	client := &Client{
		ctx: ctx,
	}

	if args.Logger != nil {
		client.logger = args.Logger
	}

	// connect the controller to Grafana for pushing annotations.
	if args.HTTPEndpoint != nil {
		args.Logger.Info("Connecting to Grafana ...", "endpoint", *args.HTTPEndpoint)

		conn, err := sdk.NewClient(*args.HTTPEndpoint, "", sdk.DefaultHTTPClient)
		if err != nil {
			return errors.Wrapf(err, "conn error")
		}

		if common.AbortAfterRetry(ctx, args.Logger, func() error {
			resp, errHealth := conn.GetHealth(ctx)
			if errHealth != nil {
				return errors.Wrapf(errHealth, "cannot get GrafanaHealth")
			}

			args.Logger.Info("Grafana Health status", "details", resp)

			return nil
		}) {
			return errors.Errorf("endpoint '%s' is unreachable", *args.HTTPEndpoint)
		}

		client.Conn = conn
	}

	// connect Grafana to controller for receiving alerts.
	if args.WebhookURL != nil {
		args.Logger.Info("Setting Notification Channel ...", "endpoint", args.WebhookURL)

		// Although the notification channel is backed by the Grafana Pod, the Grafana Service is different
		// from the Alerting Service. For this reason, we must be sure that both Services are linked to the Grafana Pod.
		if common.AbortAfterRetry(ctx, args.Logger, func() error {
			return client.SetNotificationChannel(*args.WebhookURL)
		}) {
			return errors.Errorf("notification channel error")
		}
	}

	// Register the client. It will be used by GetClient(), ClientExistsFor()...
	if args.RegisterFor != nil {
		SetClientFor(args.RegisterFor, client)
	}

	return nil
}

var clientsLocker sync.Mutex
var clients = map[string]*Client{}

type Client struct {
	ctx    context.Context
	logger *logr.Logger

	Conn *sdk.Client
}

// SetClientFor creates a new client for the given object.  It panics if it cannot parse the object's metadata,
// or if another client is already registers.
func SetClientFor(obj metav1.Object, c *Client) {
	scenario := v1alpha1.GetScenarioLabel(obj)

	clientsLocker.Lock()
	_, exists := clients[scenario]
	clientsLocker.Unlock()

	if exists {
		panic(errors.Errorf("client is already registered for scenario '%s'", scenario))
	}

	clientsLocker.Lock()
	clients[scenario] = c
	clientsLocker.Unlock()
}

// ClientExistsFor check if a client is registered for the given name. It panics if it cannot parse the object's metadata.
func ClientExistsFor(obj metav1.Object) bool {
	if !v1alpha1.HasScenarioLabel(obj) {
		return false
	}

	clientsLocker.Lock()
	_, exists := clients[v1alpha1.GetScenarioLabel(obj)]
	clientsLocker.Unlock()

	return exists
}

// GetClientFor returns the client with the given name. It panics if it cannot parse the object's metadata,
// if the client does not exist or if the client is empty.
func GetClientFor(obj metav1.Object) *Client {
	scenario := v1alpha1.GetScenarioLabel(obj)

	clientsLocker.Lock()
	c, exists := clients[scenario]
	clientsLocker.Unlock()

	if !exists {
		panic(errors.Errorf("Grafana client for scenario '%s' does not exist", scenario))
	}

	if c == nil {
		panic(errors.Errorf("Grafana client for scenario '%s' exists but is nil", scenario))
	}

	return c
}

// DeleteClientFor removes the client registered for the given object.
func DeleteClientFor(obj metav1.Object) {
	scenario := v1alpha1.GetScenarioLabel(obj)

	clientsLocker.Lock()
	delete(clients, scenario)
	clientsLocker.Unlock()
}
