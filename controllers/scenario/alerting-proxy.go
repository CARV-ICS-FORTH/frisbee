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
	"os"
	"time"

	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/carv-ics-forth/frisbee/pkg/expressions"
	notifier "github.com/golanghelper/grafana-webhook"
	"github.com/pkg/errors"
)

const (
	OverrideAdvertisedHost = "FRISBEE_ADVERTISED_HOST"
)

var gracefulShutDownTimeout = 30 * time.Second

// NewAlertingProxy  creates a Webhook for listening for events from Grafana.
func NewAlertingProxy(ctx context.Context, r *Controller) error {
	/*---------------------------------------------------*
	 * Register Alert Handlers
	 *---------------------------------------------------*/
	webhook := http.DefaultServeMux

	webhook.Handle("/", notifier.HandleWebhook(func(w http.ResponseWriter, b *notifier.Body) {
		if err := expressions.DispatchAlert(ctx, r, b); err != nil {
			r.Logger.Error(err, "Drop alert", "body", b)
		}
	}, 0))

	/*---------------------------------------------------*
	 * Start the Alerting Proxy Server
	 *---------------------------------------------------*/
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

	/*---------------------------------------------------*
	 * Advertise the Alerting Proxy Server
	 *---------------------------------------------------*/
	advertisedHost := common.DefaultAdvertisedAlertingServiceHost

	// if runs in the development mode, use the host ip of the local controller.
	if overrideHostIP := os.Getenv(OverrideAdvertisedHost); overrideHostIP != "" {
		advertisedHost = overrideHostIP
	}

	address := net.JoinHostPort(advertisedHost, common.DefaultAdvertisedAlertingServicePort)

	r.alertingProxy = "http://" + address

	r.Logger.Info("Alert Proxy Listen", "proto", "http", "address:", address)

	return nil
}
