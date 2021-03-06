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
	"strconv"
	"strings"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/grafana-tools/sdk"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func ConvertEvaluatorAlias(alias string) string {
	switch alias {
	case "below":
		return "lt"
	case "above":
		return "gt"
	case "novalue":
		return "no_value"
	default:
		return alias
	}
}

// Metric points to the Grafana metric we are interested in.
// The location can be retrieved from the Grafana URL.
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
	* count
	* last
	* median
	* diff
	* diff_abs
	* percent_diff
	* percent_diff_abs
	* count_non_null
	 */
	Reducer sdk.AlertReducer
}

type Execution struct {
	Every string

	For string
}

type TimeRange struct {
	From string
	To   string
}

type Alert struct {
	Metric

	TimeRange

	Query

	Execution
}

func ParseAlertExpr(query v1alpha1.ExprMetrics) (*Alert, error) {
	matches, err := query.Parse()
	if err != nil {
		return nil, errors.Wrapf(err, "parsing error")
	}

	alert := Alert{
		Metric:    Metric{},
		TimeRange: TimeRange{},
		Query:     Query{},
		Execution: Execution{ // These are optional, so we must have a default value.
			Every: DefaultEvaluationFrequency,
			For:   DefaultStabilityWindow,
		},
	}

	for _, field := range v1alpha1.ExprMetricsValidator.SubexpNames() {
		if field == "" { // Evaluate only existing fields.
			continue
		}

		index := v1alpha1.ExprMetricsValidator.SubexpIndex(field)
		match := matches[index]

		if match == "" { // The Field is not set
			continue
		}

		switch field {
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

		case "for":
			alert.For = match

		case "every":
			alert.Every = match

		default:
			panic(errors.Errorf("invalid field %s", field))
		}
	}

	return &alert, nil
}

// ///////////////////////////////////////////
//		Grafana Alerting Client
// ///////////////////////////////////////////

const (
	DefaultEvaluationFrequency = "1m"
	DefaultStabilityWindow     = "0s"
)

const (
	KeepState = "keep_state"
	NoData    = "no_data"

	// OK is a resolve Message, sent when the alerting state returns to false.
	OK = "ok"

	Alerting = "alerting"
)

// SetAlert adds a new alert to Grafana.
func (c *Client) SetAlert(ctx context.Context, alert *Alert, name string, msg string) error {
	if alert == nil {
		return errors.New("NIL alert was given")
	}

	board, _, err := c.Conn.GetDashboardByUID(ctx, alert.DashboardUID)
	if err != nil {
		return errors.Wrapf(err, "cannot retrieve dashboard %s", alert.DashboardUID)
	}

	for _, panel := range board.Panels {
		if panel.ID == alert.PanelID {
			if panel.Alert != nil {
				return errors.Errorf("Alert [%s] has already been set for this panel.", panel.Alert.Name)
			}

			panel.Alert = &sdk.Alert{
				Name:    name,
				Message: msg,
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
				ExecutionErrorState: KeepState,
				NoDataState:         KeepState,
				Notifications:       nil,

				Handler: 1, // Send to default notification channel (should be the controller)

				// Frequency specifies how often the scheduler should evaluate the alert rule.
				// This is referred to as the evaluation interval. Because in Frisbee we use alerts as
				// assertions, we only need to run them once. Default: 1m
				Frequency: alert.Every,

				// For specifies how long the query needs to violate the configured thresholds before the alert notification
				// triggers. Default: 5m
				For: alert.For,
			}

			break
		}
	}

	params := sdk.SetDashboardParams{
		Overwrite:  true,
		PreserveId: true,
	}

	res, err := c.Conn.SetDashboard(ctx, board, params)
	if err != nil {
		return errors.Wrap(err, "set dashboard")
	}

	if *res.Status == "success" {
		return nil
	}

	return errors.Errorf("unable to set alert [%v]", res)
}

// UnsetAlert removes an alert from Grafana.
func (c *Client) UnsetAlert(alertID string) {
	_ = alertID

	logrus.Warn("ADD FUNCTION TO REMOVE A GRAFANA ALERT")
}
