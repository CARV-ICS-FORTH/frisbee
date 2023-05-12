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
	"reflect"
	"sync"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/go-logr/logr"
	notifier "github.com/golanghelper/grafana-webhook"
	"github.com/grafana-tools/sdk"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
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

func New(ctx context.Context, setters ...Option) (*Client, error) {
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

		if err := retry.OnError(common.DefaultBackoffForServiceEndpoint,
			// retry condition
			func(err error) bool {
				client.logger.Info("Retry to connect to grafana", "Error", err)
				return true
			},
			// execution
			func() error {
				resp, err := conn.GetHealth(ctx)
				if err != nil {
					return err
				}
				client.logger.Info("Connected to Grafana", "healthStatus", resp)

				return nil
			},
			// error checking
		); err != nil {
			return nil, errors.Errorf("endpoint '%s' is unreachable", *args.HTTPEndpoint)
		}

		client.Conn = conn
		client.BaseURL = *args.HTTPEndpoint
	}

	/*---------------------------------------------------*
	 * Set Notification channel for receiving alerts
	 *---------------------------------------------------*/
	if args.WebhookURL != nil {
		client.logger.Info("Setting Notification Channel ...", "endpoint", args.WebhookURL)

		// Although the notification channel is backed by the Grafana Pod, the Grafana Service is different
		// from the Alerting Service. For this reason, we must be sure that both Services are linked to the Grafana Pod.
		if err := client.SetNotificationChannel(ctx, *args.WebhookURL); err != nil {
			return nil, errors.Wrapf(err, "notification channel error")
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

var (
	clientsLocker sync.RWMutex
	clients       = map[types.NamespacedName]*Client{}
)

type Client struct {
	logger logr.Logger

	Conn *sdk.Client

	BaseURL string
}

func getKey(obj metav1.Object) types.NamespacedName {
	if !v1alpha1.HasScenarioLabel(obj) {
		panic(errors.Errorf("Object '%s/%s' does not have Scenario labels", obj.GetNamespace(), obj.GetName()))
	}

	// The key structure is as follows:
	// Namespaces: provides separation between test-cases
	// Scenario: Is a flag that is propagated all over the test-cases.
	return types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      v1alpha1.GetScenarioLabel(obj),
	}
}

// SetClientFor creates a new client for the given object.  It panics if it cannot parse the object's metadata,
// or if another client is already registers.
func SetClientFor(obj metav1.Object, client *Client) {
	key := getKey(obj)

	clientsLocker.RLock()
	_, exists := clients[key]
	clientsLocker.RUnlock()

	if exists {
		panic(errors.Errorf("client is already registered for '%s'", key))
	}

	clientsLocker.Lock()
	clients[key] = client
	clientsLocker.Unlock()

	client.logger.Info("Set Grafana client for", "obj", key)
}

// GetClientFor returns the client with the given name. It panics if it cannot parse the object's metadata,
// if the client does not exist, or if the client is empty.
func GetClientFor(obj metav1.Object) *Client {
	if !v1alpha1.HasScenarioLabel(obj) {
		logrus.Warn("No Scenario FOR ", obj.GetName(), " type ", reflect.TypeOf(obj))

		return nil
	}

	key := getKey(obj)

	clientsLocker.RLock()
	defer clientsLocker.RUnlock()

	client, exists := clients[key]
	if !exists || client == nil {
		panic("nil grafana client was found for object: " + obj.GetName())
	}

	return client
}

// HasClientFor returns whether there is a non-nil grafana client is registered for the given object.
func HasClientFor(obj metav1.Object) bool {
	if !v1alpha1.HasScenarioLabel(obj) {
		return false
	}

	key := getKey(obj)

	clientsLocker.RLock()
	defer clientsLocker.RUnlock()

	client, exists := clients[key]
	if !exists || client == nil {
		return false
	}

	return true
}

// DeleteClientFor removes the client registered for the given object.
func DeleteClientFor(obj metav1.Object) {
	key := getKey(obj)

	clientsLocker.Lock()
	defer clientsLocker.Unlock()

	delete(clients, key)
}
