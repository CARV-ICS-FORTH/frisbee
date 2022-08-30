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
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var defaultLogger = zap.New(zap.UseDevMode(true)).WithName("default.grafana")

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
		if r == (logr.Logger{}) {
			panic("trying to pass empty logger")
		}

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

	client := &Client{ctx: ctx}

	if args.Logger == (logr.Logger{}) {
		client.logger = defaultLogger
	} else {
		client.logger = args.Logger
	}

	// connect the controller to Grafana for pushing annotations.
	if args.HTTPEndpoint != nil {
		args.Logger.Info("Connecting to Grafana ...", "endpoint", *args.HTTPEndpoint)

		conn, err := sdk.NewClient(*args.HTTPEndpoint, "", sdk.DefaultHTTPClient)
		if err != nil {
			return errors.Wrapf(err, "conn error")
		}

		if err := wait.ExponentialBackoffWithContext(ctx, common.BackoffForServiceEndpoint, func() (done bool, err error) {
			resp, errHealth := conn.GetHealth(ctx)
			switch {
			case errHealth != nil: // API connection error. Just retry
				return false, nil
			default:
				args.Logger.Info("Connected to Grafana", "healthStatus", resp)
				return true, nil
			}
		}); err != nil {
			return errors.Errorf("endpoint '%s' is unreachable", *args.HTTPEndpoint)
		}

		client.Conn = conn
	}

	// connect Grafana to controller for receiving alerts.
	if args.WebhookURL != nil {
		args.Logger.Info("Setting Notification Channel ...", "endpoint", args.WebhookURL)

		// Although the notification channel is backed by the Grafana Pod, the Grafana Service is different
		// from the Alerting Service. For this reason, we must be sure that both Services are linked to the Grafana Pod.
		if err := client.SetNotificationChannel(ctx, *args.WebhookURL); err != nil {
			return errors.Wrapf(err, "notification channel error")
		}
	}

	// Register the client. It will be used by GetClient(), ClientExistsFor()...
	if args.RegisterFor != nil {
		SetClientFor(args.RegisterFor, client)
	}

	return nil
}

var clientsLocker sync.RWMutex
var clients = map[types.NamespacedName]*Client{}

type Client struct {
	ctx    context.Context
	logger logr.Logger

	Conn *sdk.Client
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
func SetClientFor(obj metav1.Object, c *Client) {

	key := getKey(obj)

	clientsLocker.RLock()
	_, exists := clients[key]
	clientsLocker.RUnlock()

	if exists {
		panic(errors.Errorf("client is already registered for '%s'", key))
	}

	clientsLocker.Lock()
	clients[key] = c
	clientsLocker.Unlock()

	c.logger.Info("Set Grafana client for", "obj", key)
}

// GetClientFor returns the client with the given name. It panics if it cannot parse the object's metadata,
// if the client does not exist or if the client is empty.
func GetClientFor(obj metav1.Object) *Client {
	if !v1alpha1.HasScenarioLabel(obj) {
		logrus.Warn("No Scenario FOR ", obj.GetName(), " type ", reflect.TypeOf(obj))
		return nil
	}

	key := getKey(obj)

	clientsLocker.RLock()
	c := clients[key]
	clientsLocker.RUnlock()

	return c
}

// DeleteClientFor removes the client registered for the given object.
func DeleteClientFor(obj metav1.Object) {
	key := getKey(obj)

	clientsLocker.Lock()
	delete(clients, key)
	clientsLocker.Unlock()
}
