/*
Copyright 2023 ICS-FORTH.

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
	"time"
)

type URL struct {
	Endpoint     string
	DashboardUID *string
	FromTS       *time.Time
	ToTS         *time.Time
	PanelID      *uint
	Kiosk        bool
}

// NewURL access an endpoint at the form: grafana-fedbed-48.knot-platform.eu
func NewURL(endpoint string) *URL {
	return &URL{
		Endpoint: endpoint,
	}
}

func (url *URL) WithKiosk() *URL {
	url.Kiosk = true

	return url
}

func (url *URL) WithFromTS(ts time.Time) *URL {
	url.FromTS = &ts

	return url
}

func (url *URL) WithToTS(ts time.Time) *URL {
	url.ToTS = &ts

	return url
}

func (url *URL) WithPanel(panelID uint) *URL {
	url.PanelID = &panelID

	return url
}

func (url *URL) WithDashboard(dashboardUID string) *URL {
	url.DashboardUID = &dashboardUID

	return url
}

func (url *URL) APIQuery() string {
	return fmt.Sprintf("http://%s/api/ds/query", url.Endpoint)
}

/*
func (url *URL) APIQuery() string {
	var final strings.Builder

	final.WriteString("http://" + url.Endpoint + string(url.RelPath))

	if url.FromTS != nil {
		//	final.WriteString(fmt.Sprintf("&from=%d", url.FromTS.UnixMilli()))
	}

	if url.ToTS != nil {
		//	final.WriteString(fmt.Sprintf("&to=%d", url.ToTS.UnixMilli()))
	}

	if url.PanelID != nil {
		final.WriteString(fmt.Sprintf("&viewPanel=%d", url.PanelID))
	}

	if url.Kiosk {
		final.WriteString("&kiosk")
	}

	/*
		if url.InspectData {
			final.WriteString("&inspectTab=data")
		}


	return final.String()

}
*/

func BuildURL(grafanaEndpoint string, dashboard string, from int64, to int64, postfix string) string {
	return fmt.Sprintf("http://%s/d/%s?orgId=1&from=%d&to=%d%s", grafanaEndpoint, dashboard, from, to, postfix)
}
