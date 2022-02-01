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
	"fmt"
	"net/http"

	"github.com/carv-ics-forth/frisbee/controllers/utils"
	"github.com/carv-ics-forth/frisbee/pkg/netutils"
	notifier "github.com/golanghelper/grafana-webhook"
	"github.com/grafana-tools/sdk"
	"github.com/pkg/errors"
)

func (c *Client) SetNotificationChannel(webhookPort string, cb func(b *notifier.Body)) error {
	var addr string

	var url string

	// If the controller runs within the Kubernetes cluster, we use the assigned name as the advertised host
	// If the controller runs externally to the Kubernetes cluster, we use the public IP of the local machine.
	if host := utils.DefaultConfiguration.AdvertisedHost; host != "" {
		addr = fmt.Sprintf("%s:%s", host, webhookPort)
		url = fmt.Sprintf("http://%s", addr)
	} else {
		ip, err := netutils.GetPublicIP()
		if err != nil {
			return errors.Wrapf(err, "cannot get controller's public ip")
		}

		addr = fmt.Sprintf("%s:%s", ip.String(), webhookPort)
		url = fmt.Sprintf("http://%s", addr)
	}

	errCh := make(chan error, 1)

	go func() {
		handler := http.DefaultServeMux
		handler.HandleFunc("/", notifier.HandleWebhook(func(w http.ResponseWriter, b *notifier.Body) {
			cb(b)
		}, 0))

		errCh <- http.ListenAndServe(addr, handler)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return errors.Wrapf(err, "webhook server failed")
		}
	case <-c.ctx.Done():
		return errors.Wrapf(c.ctx.Err(), "webhook server failed")
	default: // continue
	}

	c.logger.Info("Frisbee Controller is listening for Grafana alerts on", "url", url)

	// use the webhook as notification channel for grafana
	feedback := sdk.AlertNotification{
		Name:                  "to-frisbee-controller",
		Type:                  "webhook",
		IsDefault:             true,
		DisableResolveMessage: false,
		SendReminder:          false,
		Settings: map[string]string{
			"url": url,
		},
	}

	_, err := c.Conn.CreateAlertNotification(c.ctx, feedback)

	return errors.Wrapf(err, "cannot create feedback notification channel")
}
