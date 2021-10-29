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

package workflow

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/fnikolai/frisbee/controllers/utils/grafana"
	"github.com/fnikolai/frisbee/pkg/netutils"
	notifier "github.com/golanghelper/grafana-webhook"
	"github.com/grafana-tools/sdk"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
)

func (r *Controller) initGrafana(ctx context.Context, apiURI string) error {

	var healthCheckTimeout = wait.Backoff{
		Duration: 5 * time.Second,
		Factor:   5,
		Jitter:   0.1,
		Steps:    4,
	}

	grafanaClient, err := sdk.NewClient(apiURI, "", sdk.DefaultHTTPClient)
	if err != nil {
		return errors.Wrapf(err, "grafanaClient error")
	}

	// retry until Grafana is ready to receive annotations.
	err = retry.OnError(healthCheckTimeout, func(_ error) bool { return true }, func() error {
		_, err := grafanaClient.GetHealth(ctx)

		return errors.Wrapf(err, "grafana health error")
	})

	if err != nil {
		return errors.Wrapf(err, "grafana is unreachable")
	}

	url, err := r.runNotificationWebhook(ctx, "6666")
	if err != nil {
		return errors.Wrapf(err, "cannot run a notification webhook")
	}

	r.Logger.Info("Grafana webhook is listening on", "url", url)

	// create a feedback alert notification channel
	feedback := sdk.AlertNotification{
		Name:                  "to-frisbee-controller",
		Type:                  "webhook",
		IsDefault:             true,
		DisableResolveMessage: true,
		SendReminder:          false,
		Settings: map[string]string{
			"url": url,
		},
	}

	if _, err := grafanaClient.CreateAlertNotification(ctx, feedback); err != nil {
		return errors.Wrapf(err, "cannot create feedback notification channel")
	}

	grafana.SetAnnotator(ctx, grafanaClient)

	return nil
}

func (r *Controller) runNotificationWebhook(ctx context.Context, port string) (string, error) {
	// get local ip
	ip, err := netutils.GetPublicIP()
	if err != nil {
		return "", errors.Wrapf(err, "cannot get controller's public ip")
	}

	handler := http.DefaultServeMux
	handler.HandleFunc("/", notifier.HandleWebhook(func(w http.ResponseWriter, b *notifier.Body) {
		r.Info("Grafana Alert",
			"title", b.Title,
			"message", b.Message,
			"matches", b.EvalMatches,
			"state", b.State,
		)
	}, 0))

	addr := fmt.Sprintf("%s:%s", ip.String(), port)

	errCh := make(chan error, 1)

	go func() {
		errCh <- http.ListenAndServe(addr, handler)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return "", errors.Wrapf(err, "webhook server failed")
		}
	case <-ctx.Done():
		return "", errors.Wrapf(ctx.Err(), "webhook server failed")
	default:
		url := fmt.Sprintf("http://%s", addr)

		return url, nil
	}

	panic("should never happen")
}
