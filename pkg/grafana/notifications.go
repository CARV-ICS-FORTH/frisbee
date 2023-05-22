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

	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/grafana-tools/sdk"
	"k8s.io/apimachinery/pkg/util/wait"
)

func (c *Client) SetNotificationChannel(ctx context.Context, webhookURL string) error {
	// use the webhook as notification channel for grafana
	feedback := sdk.AlertNotification{
		Name:                  "Frisbee-Webhook",
		Type:                  "webhook",
		IsDefault:             true,
		DisableResolveMessage: false,
		SendReminder:          false,
		Settings: map[string]string{
			"url": webhookURL,
		},
	}

	// Although the notification channel is backed by the Grafana Pod, the Grafana Service is different
	// from the Alerting Service. For this reason, we must be sure that both Services are linked to the Grafana Pod.
	retryCond := func() (done bool, err error) {
		if _, err := c.Conn.CreateAlertNotification(ctx, feedback); err != nil {
			// Retry
			defaultLogger.Info("connection error", "Err", err)

			return false, nil
		}

		// OK
		return true, nil
	}

	return wait.ExponentialBackoffWithContext(ctx, common.DefaultBackoffForServiceEndpoint, retryCond)
}
