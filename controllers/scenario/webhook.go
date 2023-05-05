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

package scenario

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/carv-ics-forth/frisbee/pkg/expressions"
	notifier "github.com/golanghelper/grafana-webhook"
	"github.com/pkg/errors"
)

var gracefulShutDownTimeout = 30 * time.Second

// CreateWebhookServer  creates a Webhook for listening for events from Grafana.
func (r *Controller) CreateWebhookServer(ctx context.Context) error {
	endpoint := "http://" + net.JoinHostPort(common.DefaultAdvertisedAlertingServiceHost, common.DefaultAdvertisedAlertingServicePort)

	r.Logger.Info("StartWebhook", "URL", endpoint)

	webhook := http.DefaultServeMux

	webhook.Handle("/", notifier.HandleWebhook(func(w http.ResponseWriter, b *notifier.Body) {
		if err := expressions.DispatchAlert(ctx, r, b); err != nil {
			r.Logger.Error(err, "Drop alert", "body", b)
		}
	}, 0))

	// Start the server
	srv := &http.Server{
		Addr:              ":" + common.DefaultAdvertisedAlertingServicePort,
		Handler:           webhook,
		ReadHeaderTimeout: 1 * time.Minute, // To DDos that open multiple concurrent streams.
	}

	idleConnectionsClosed := make(chan error)

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			idleConnectionsClosed <- err
		}
	}()

	go func() {
		select {
		case <-ctx.Done():
			r.Logger.Info("Shutdown signal received, waiting for webhook server to finish")

		case err := <-idleConnectionsClosed:
			r.Logger.Error(err, "Shutting down the webhook server")
		}

		// need a new background context for the graceful shutdown. the ctx is already cancelled.
		gracefulShutDown, cancel := context.WithTimeout(ctx, gracefulShutDownTimeout)
		defer cancel()

		if err := srv.Shutdown(gracefulShutDown); err != nil {
			r.Logger.Error(err, "shutting down the webhook server")
		}

		close(idleConnectionsClosed)
	}()

	r.notificationEndpoint = endpoint

	return nil
}
