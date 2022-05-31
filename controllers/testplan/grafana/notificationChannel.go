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
	"github.com/grafana-tools/sdk"
	"github.com/pkg/errors"
)

func (c *Client) SetNotificationChannel(webhookURL string) error {
	// use the webhook as notification channel for grafana
	feedback := sdk.AlertNotification{
		Name:                  "to-frisbee-controller",
		Type:                  "webhook",
		IsDefault:             true,
		DisableResolveMessage: false,
		SendReminder:          false,
		Settings: map[string]string{
			"url": webhookURL,
		},
	}

	_, err := c.Conn.CreateAlertNotification(c.ctx, feedback)

	return errors.Wrapf(err, "cannot create feedback notification channel")
}
