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
		want    *grafana.AlertRule
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
			want: &grafana.AlertRule{
				Metric: grafana.Metric{
					DashboardUID: "wpFnYRwGk",
					PanelID:      2,
					MetricName:   "bitrate",
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
				FromTime:  "15m",
				ToTime:    "now",
				Frequency: grafana.DefaultEvaluationFrequency,
				Duration:  grafana.DefaultDecisionWindow,
			},
			wantErr: false,
		},

		{
			name: "no-params",
			args: args{query: "avg() of query(wpFnYRwGk/2/bitrate, 15m, now) is novalue()"},
			want: &grafana.AlertRule{
				Metric: grafana.Metric{
					DashboardUID: "wpFnYRwGk",
					PanelID:      2,
					MetricName:   "bitrate",
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
				FromTime:  "15m",
				ToTime:    "now",
				Frequency: grafana.DefaultEvaluationFrequency,
				Duration:  grafana.DefaultDecisionWindow,
			},
			wantErr: false,
		},
		{
			name: "multi-params",
			args: args{query: "avg() of query(wpFnYRwGk/2/bitrate, 15m, now) is within_range(10,50)"},
			want: &grafana.AlertRule{
				Metric: grafana.Metric{
					DashboardUID: "wpFnYRwGk",
					PanelID:      2,
					MetricName:   "bitrate",
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
				FromTime:  "15m",
				ToTime:    "now",
				Frequency: grafana.DefaultEvaluationFrequency,
				Duration:  grafana.DefaultDecisionWindow,
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
