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
	"regexp"
	"strconv"
	"strings"

	"github.com/grafana-tools/sdk"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	Assertor *regexp.Regexp
)

func init() {
	// Expressions evaluated with https://regex101.com/r/sIspYb/1/
	Assertor = regexp.MustCompile(`(?m)^(?P<reducer>\w+)\(\)\s+OF\s+query\((?P<dashboardUID>\w+)\/(?P<panelID>\d+)\/(?P<metric>\w+),\s+(?P<from>\w+),\s+(?P<to>\w+)\)\s+IS\s+(?P<evaluator>\w+)\((?P<params>\d*[,\s]*\d*)\)$`)
}

func ConvertEvaluatorAlias(alias string) string {
	switch v := strings.ToLower(alias); v {
	case "below":
		return "lt"
	case "above":
		return "gt"
	case "novalue":
		return "no_value"
	default:
		return v
	}
}

// Metric points to the Grafana metric we are interested in.
// The location can retrieved from the Grafana URL.
// Example:
// URL: http://grafana.platform.science-hangar.eu/d/wpFnYRwGk/iperf?orgId=1&viewPanel=2
// Metric: wpFnYRwGk/2/bitrate
type Metric struct {
	DashboardUID string

	PanelID uint

	MetricName string
}

type Query struct {
	/* == Evaluator
	* below
	* above
	* within_range
	* outside_range
	* empty
	 */
	Evaluator sdk.AlertEvaluator

	/* == Reducers
	* avg
	* min
	* max
	* sum
	 */
	Reducer sdk.AlertReducer
}

type TimeRange struct {
	From string
	To   string
}

type Alert struct {
	Name string

	Message string

	Metric

	TimeRange

	Query
}

func NewAlert(query string) (*Alert, error) {
	matches := Assertor.FindStringSubmatch(query)
	if len(matches) == 0 {
		return nil, errors.Errorf(`erroneous query. Examples:
			1) avg() OF query(wpFnYRwGk/2/bitrate, 15m, now) IS BELOW(14)
			2) avg() OF query(wpFnYRwGk/2/bitrate, 15m, now) IS NOVALUE()
			3) avg() OF query(wpFnYRwGk/2/bitrate, 15m, now) IS WithinRange(4, 88)
		`)
	}

	alert := Alert{}

	for _, name := range Assertor.SubexpNames() {
		if name == "" {
			continue
		}

		index := Assertor.SubexpIndex(name)
		match := matches[index]

		switch name {
		case "reducer":
			alert.Reducer.Type = match
			alert.Reducer.Params = nil // Not captured by the present regex

		case "dashboardUID":
			alert.Metric.DashboardUID = match

		case "panelID":
			panelID, err := strconv.ParseUint(match, 10, 32)
			if err != nil {
				return nil, errors.Wrapf(err, "erroneous panelID")
			}

			alert.Metric.PanelID = uint(panelID)

		case "metric":
			alert.Metric.MetricName = match

		case "from":
			alert.TimeRange.From = match

		case "to":
			alert.TimeRange.To = match

		case "evaluator":
			alert.Evaluator.Type = ConvertEvaluatorAlias(match)

		case "params":
			// todo: improve the way we parse params
			if match == "" {
				continue
			}

			paramsStr := strings.Split(match, ",")

			params := make([]float64, len(paramsStr))

			for i, m := range paramsStr {
				param, err := strconv.ParseFloat(m, 32)
				if err != nil {
					return nil, errors.Wrapf(err, "erroneous parameters")
				}

				params[i] = param
			}

			alert.Evaluator.Params = params

		default:
			panic(errors.Errorf("invalid field %s", name))
		}
	}

	return &alert, nil
}

// ///////////////////////////////////////////
//		Grafana Alerting Client
// ///////////////////////////////////////////

// SetAlert adds a new alert to Grafana.
func (c *Client) SetAlert(alert *Alert) (uint, error) {
	board, _, err := c.Conn.GetDashboardByUID(context.Background(), alert.DashboardUID)
	if err != nil {
		return 0, errors.Wrapf(err, "cannot retrieve dashboard %s", alert.DashboardUID)
	}

	for _, panel := range board.Panels {
		if panel.ID == alert.PanelID {
			if panel.Alert != nil {
				return 0, errors.Errorf("An alert has already been set for this panel")
			}

			panel.Alert = &sdk.Alert{
				Name:    alert.Name,
				Message: alert.Message,
				Conditions: []sdk.AlertCondition{
					{
						Evaluator: alert.Evaluator,
						Operator: sdk.AlertOperator{
							Type: "and",
						},
						Query: sdk.AlertQuery{
							Params: []string{alert.Metric.MetricName, alert.TimeRange.From, alert.TimeRange.To},
						},
						Reducer: alert.Reducer,
						Type:    "query",
					},
				},
				ExecutionErrorState: "keep_state",
				NoDataState:         "keep_state",
				Notifications:       nil,

				Handler: 1, // Send to default notification channel (should be the controller)

				// Frequency specifies how often the scheduler should evaluate the alert rule.
				// This is referred to as the evaluation interval. Because in Frisbee we use alerts as
				// assertions, we only need to run them once. Default: 1m
				Frequency: "1m",

				// For specifies how long the query needs to violate the configured thresholds before the alert notification
				// triggers. Default: 5m
				For: "0s",
			}

			break
		}
	}

	params := sdk.SetDashboardParams{
		Overwrite:  true,
		PreserveId: true,
	}

	res, err := c.Conn.SetDashboard(context.Background(), board, params)
	if err != nil {
		return 0, errors.Wrap(err, "set dashboard")
	}

	logrus.Warn("STATUS ", res)

	if *res.Status == "success" {
		return 0, nil
	}

	return *res.ID, errors.Errorf("unable to set alert [%v]", res)
}

// UnsetAlert removes an alert from Grafana.
func (c *Client) UnsetAlert(alertID string) {
	_ = alertID
	logrus.Warn("ADD FUNCTION TO REMOVE A GRAFANA ALERT")
}
