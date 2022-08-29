package grafana_test

import (
	"reflect"
	"testing"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/pkg/grafana"
	"github.com/grafana-tools/sdk"
)

func TestParseAlert(t *testing.T) {
	type args struct {
		query v1alpha1.ExprMetrics
	}
	tests := []struct {
		name    string
		args    args
		want    *grafana.Alert
		wantErr bool
	}{
		{
			name:    "empty",
			args:    args{query: ""},
			want:    nil,
			wantErr: true,
		},
		{
			name: "single-params",
			args: args{query: "avg() of query(wpFnYRwGk/2/bitrate, 15m, now) is below(14)"},
			want: &grafana.Alert{
				Metric: grafana.Metric{
					DashboardUID: "wpFnYRwGk",
					PanelID:      2,
					MetricName:   "bitrate",
				},
				TimeRange: grafana.TimeRange{
					From: "15m",
					To:   "now",
				},
				Query: grafana.Query{
					Evaluator: sdk.AlertEvaluator{
						Type:   grafana.ConvertEvaluatorAlias("below"),
						Params: []float64{14},
					},
					Reducer: sdk.AlertReducer{
						Type:   "avg",
						Params: nil,
					},
				},
				Execution: grafana.Execution{
					Every: grafana.DefaultEvaluationFrequency,
					For:   grafana.DefaultStabilityWindow,
				},
			},
			wantErr: false,
		},

		{
			name: "no-params",
			args: args{query: "avg() of query(wpFnYRwGk/2/bitrate, 15m, now) is novalue()"},
			want: &grafana.Alert{
				Metric: grafana.Metric{
					DashboardUID: "wpFnYRwGk",
					PanelID:      2,
					MetricName:   "bitrate",
				},
				TimeRange: grafana.TimeRange{
					From: "15m",
					To:   "now",
				},
				Query: grafana.Query{
					Evaluator: sdk.AlertEvaluator{
						Type:   grafana.ConvertEvaluatorAlias("novalue"),
						Params: nil,
					},
					Reducer: sdk.AlertReducer{
						Type:   "avg",
						Params: nil,
					},
				},
				Execution: grafana.Execution{
					Every: grafana.DefaultEvaluationFrequency,
					For:   grafana.DefaultStabilityWindow,
				},
			},
			wantErr: false,
		},
		{
			name: "multi-params",
			args: args{query: "avg() of query(wpFnYRwGk/2/bitrate, 15m, now) is within_range(10,50)"},
			want: &grafana.Alert{
				Metric: grafana.Metric{
					DashboardUID: "wpFnYRwGk",
					PanelID:      2,
					MetricName:   "bitrate",
				},
				TimeRange: grafana.TimeRange{
					From: "15m",
					To:   "now",
				},
				Query: grafana.Query{
					Evaluator: sdk.AlertEvaluator{
						Type:   grafana.ConvertEvaluatorAlias("within_range"),
						Params: []float64{10, 50},
					},
					Reducer: sdk.AlertReducer{
						Type:   "avg",
						Params: nil,
					},
				},
				Execution: grafana.Execution{
					Every: grafana.DefaultEvaluationFrequency,
					For:   grafana.DefaultStabilityWindow,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := grafana.ParseAlertExpr(tt.args.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAlertExpr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseAlertExpr() got = %v, want %v", got, tt.want)
			}
		})
	}
}
