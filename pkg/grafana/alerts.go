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
	"strconv"
	"strings"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/grafana-tools/sdk"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	DefaultEvaluationFrequency = "1m"
	DefaultDecisionWindow      = "0s"
)

type (
	ExecErrState string
	NoDataState  string
)

const (
	ErrOK          ExecErrState = "OK"
	ErrError       ExecErrState = "Error"
	ErrAlerting    ExecErrState = "Alerting"
	NoDataOk       NoDataState  = "OK"
	NoData         NoDataState  = "NoData"
	NoDataAlerting NoDataState  = "Alerting"
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
// Metric: wpFnYRwGk/2/bitrate.
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

// AlertRule is a set of evaluation criteria that determines whether an alert will fire.
// The alert rule consists of one or more queries and expressions, a condition, the frequency of evaluation,
// and optionally, the duration over which the condition is met.
type AlertRule struct {
	Metric

	Query

	// FromTime indicate a relative duration accounted for the alerting. e.g, 15m ago
	FromTime string

	// ToTime indicate a point of reference accounted for the alerting. e.g, now
	ToTime string

	// Frequency specifies how frequently an alert rule is evaluated Must be a multiple of 10 seconds. For examples, 1m, 30s.
	Frequency string

	// Duration, when configured, specifies the duration for which the condition must be true before an alert fires.
	Duration string
}

func ParseAlertExpr(query v1alpha1.ExprMetrics) (*AlertRule, error) {
	matches, err := query.Parse()
	if err != nil {
		return nil, errors.Wrapf(err, "parsing error")
	}

	alert := AlertRule{
		Frequency: DefaultEvaluationFrequency,
		Duration:  DefaultDecisionWindow,
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
			alert.FromTime = match

		case "to":
			alert.ToTime = match

		case "evaluator":
			alert.Evaluator.Type = ConvertEvaluatorAlias(match)

		case "params":
			paramsStr := strings.Split(match, ",")

			params := make([]float64, len(paramsStr))

			for i, m := range paramsStr {

				quantity, err := resource.ParseQuantity(m)
				if err != nil {
					return nil, errors.Wrapf(err, "erroneous parameters")
				}

				params[i] = quantity.AsApproximateFloat64()
			}

			alert.Evaluator.Params = params

		case "for":
			alert.Duration = match

		case "every":
			alert.Frequency = match

		default:
			panic(errors.Errorf("invalid field %s", field))
		}
	}

	return &alert, nil
}

// SetAlert adds a new alert to Grafana using the Legacy API.
func (c *Client) SetAlert(ctx context.Context, alert *AlertRule, name string, msg string) error {
	if c == nil {
		panic("empty client was given")
	}

	if alert == nil {
		return errors.New("NIL alert was given")
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()

	/*---------------------------------------------------*
	 * Get the dashboard
	 *---------------------------------------------------*/
	board, _, err := c.Conn.GetDashboardByUID(ctx, alert.DashboardUID)
	if err != nil {
		return errors.Wrapf(err, "cannot retrieve dashboard %s", alert.DashboardUID)
	}

	/*---------------------------------------------------*
	 * Set Alert to the appropriate Panel
	 *---------------------------------------------------*/
	var panelExists bool

	for _, panel := range board.Panels {
		if panel.ID != alert.PanelID {
			// skip irrelevant panels
			continue
		}

		if panel.Alert != nil {
			return errors.Errorf("alert [%s] has already been set for this panel", panel.Alert.Name)
		}

		panel.CommonPanel.Alert = &sdk.Alert{
			Name:          name,
			AlertRuleTags: map[string]string{"my-alert": "yeeha"},
			Conditions: []sdk.AlertCondition{
				{
					Evaluator: alert.Evaluator,
					Operator: sdk.AlertOperator{
						Type: "and",
					},
					Query: sdk.AlertQuery{
						Params: []string{alert.Metric.MetricName, alert.FromTime, alert.ToTime},
					},
					Reducer: alert.Reducer,
					Type:    "query",
				},
			},

			ExecutionErrorState: string(ErrError),
			NoDataState:         string(NoData),

			// Frequency specifies how often the scheduler should evaluate the alert rule.
			// This is referred to as the evaluation interval. Because in Frisbee we use alerts as
			// assertions, we only need to run them once. Default: 1m
			Frequency: alert.Frequency,

			// For specifies how long the query needs to violate the configured thresholds before the alert notification
			// triggers. Default: 5m
			For: alert.Duration,

			Notifications: nil,
			Message:       msg,
			// Handler: 1, // Send to default notification channel (should be the controller)
		}

		panelExists = true
	}

	if !panelExists {
		c.logger.Info("No matching panel for alert", "alertRule", alert)

		return errors.New("Invalid panel reference")
	}

	/*---------------------------------------------------*
	 * Update the dashboard
	 *---------------------------------------------------*/
	retryCond := func(ctx context.Context) (done bool, err error) {
		resp, errReq := c.Conn.SetDashboard(ctx, board, sdk.SetDashboardParams{
			Overwrite:  true, // Needed to avoid "someone else had written the dashboard".
			PreserveId: true, // Needed to avoid "someone else had written the dashboard".
		})

		// Retry
		if errReq != nil {
			c.logger.Info("Connection error. Retry", "alertName", name, "resp", resp, "err", errReq)

			return false, nil
		}

		// OK
		if resp.Status != nil && *resp.Status == respAlertSuccess {
			c.logger.Info("Set alert", "alertName", name)

			return true, nil
		}

		panic("should not go here")
	}

	if err := wait.ExponentialBackoffWithContext(ctxTimeout, common.DefaultBackoffForServiceEndpoint, retryCond); err != nil {
		return errors.Wrapf(err, "cannot set alert '%s'", name)
	}

	return nil
}

// UnsetAlert removes an alert from Grafana.
func (c *Client) UnsetAlert(alertID string) {
	_ = alertID

	logrus.Warn("ADD FUNCTION TO REMOVE A GRAFANA ALERT")
}
