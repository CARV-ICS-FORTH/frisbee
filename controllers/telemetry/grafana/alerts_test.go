package grafana_test

import (
	"reflect"
	"testing"

	"github.com/carv-ics-forth/frisbee/controllers/telemetry/grafana"
	"github.com/grafana-tools/sdk"
)

func TestParseAlert(t *testing.T) {
	type args struct {
		query string
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
			args: args{query: "avg() OF query(wpFnYRwGk/2/bitrate, 15m, now) IS BELOW(14)"},
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
						Type:   grafana.ConvertEvaluatorAlias("BELOW"),
						Params: []float64{14},
					},
					Reducer: sdk.AlertReducer{
						Type:   "avg",
						Params: nil,
					},
				},
			},
			wantErr: false,
		},

		{
			name: "no-params",
			args: args{query: "avg() OF query(wpFnYRwGk/2/bitrate, 15m, now) IS NOVALUE()"},
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
						Type:   grafana.ConvertEvaluatorAlias("NOVALUE"),
						Params: nil,
					},
					Reducer: sdk.AlertReducer{
						Type:   "avg",
						Params: nil,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "multi-params",
			args: args{query: "avg() OF query(wpFnYRwGk/2/bitrate, 15m, now) IS WITHIN_RANGE(10,50)"},
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
						Type:   grafana.ConvertEvaluatorAlias("WITHIN_RANGE"),
						Params: []float64{10, 50},
					},
					Reducer: sdk.AlertReducer{
						Type:   "avg",
						Params: nil,
					},
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
