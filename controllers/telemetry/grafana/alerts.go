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

	"github.com/davecgh/go-spew/spew"
	"github.com/grafana-tools/sdk"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Reducer = string

const (
	Avg = Reducer("avg")
	Min = Reducer("min")
	Max = Reducer("max")
	Sum = Reducer("sum")
)

type Evaluator = string

const (
	IsAbove = Evaluator("gt")

	IsBelow = Evaluator("lt")

	IsWithinRange = Evaluator("within_range")

	IsOutsideRange = Evaluator("outside_range")

	HasNotValue = Evaluator("no_value")
)

type Operator = string

const (
	And = Reducer("and")
)

// wpFnYRwGk/iperf?orgId=1&from=now-5m&to=now&viewPanel=4
type Metric struct {
	DashboardUID string

	PanelID uint

	MetricName string
}

type Query struct {
	Evaluator sdk.AlertEvaluator

	Reducer sdk.AlertReducer
}

type TimeRange struct {
	From metav1.Time
	To   metav1.Time
}

// Original avg() OF query(A, 15m, now) IS BELOW 14
type Assertion struct {
	// Name is a unique identifier of the alert
	Name string

	Message string

	Metric

	TimeRange

	Query
}

func (c *Client) SetAlert() error {
	// objectCreated :=  metav1.Time{Time: time.Now().Add(time.Duration(5) * time.Minute)}
	// objectCompleted :=  metav1.Time{Time: time.Now().Add(time.Duration(2) * time.Minute)}

	assert := Assertion{
		Name:    "User Alerting",
		Message: "avg() OF query(bitrate, 15m, now) IS BELOW 14",
		Metric: Metric{
			DashboardUID: "wpFnYRwGk",
			PanelID:      2,
			MetricName:   "bitrate",
		},
		Query: Query{
			Evaluator: sdk.AlertEvaluator{
				Type:   IsBelow,
				Params: []float64{1000},
			},
			Reducer: sdk.AlertReducer{
				Type:   Avg,
				Params: nil,
			},
		},
	}

	// The best way to get stuff like that is to create the desired alert in grafana, and then describe it.
	alert := sdk.Alert{
		Name:    assert.Name,
		Message: assert.Message,
		Conditions: []sdk.AlertCondition{
			{
				Evaluator: assert.Evaluator,
				Operator: sdk.AlertOperator{
					Type: And,
				},
				Query: sdk.AlertQuery{
					// Params: []string{"bitrate", "5m", "now"},
					Params: []string{assert.Metric.MetricName, "5m", "now"},
				},
				Reducer: assert.Reducer,
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
		For: "15s",
	}

	board, _, err := c.Conn.GetDashboardByUID(context.Background(), assert.DashboardUID)
	if err != nil {
		return errors.Wrapf(err, "cannot retrieve dashboard %s", assert.DashboardUID)
	}

	for _, panel := range board.Panels {
		if panel.ID == assert.PanelID {
			panel.Alert = &alert
		}
	}

	params := sdk.SetDashboardParams{
		Overwrite:  true,
		PreserveId: true,
	}

	status, err := c.Conn.SetDashboard(context.Background(), board, params)
	if err != nil {
		return errors.Wrap(err, "set dashboard")
	}

	logrus.Warn("Dashboard DEBUG")

	spew.Dump(status)

	return nil
}
