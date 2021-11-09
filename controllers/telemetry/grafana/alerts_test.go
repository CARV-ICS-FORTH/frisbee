package grafana

import (
	"reflect"
	"testing"

	"github.com/grafana-tools/sdk"
)

func TestParseAlert(t *testing.T) {
	type args struct {
		query string
	}
	tests := []struct {
		name    string
		args    args
		want    *Alert
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
			want: &Alert{
				Metric: Metric{
					DashboardUID: "wpFnYRwGk",
					PanelID:      2,
					MetricName:   "bitrate",
				},
				TimeRange: TimeRange{
					From: "15m",
					To:   "now",
				},
				Query: Query{
					Evaluator: sdk.AlertEvaluator{
						Type:   ConvertEvaluatorAlias("BELOW"),
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
			want: &Alert{
				Metric: Metric{
					DashboardUID: "wpFnYRwGk",
					PanelID:      2,
					MetricName:   "bitrate",
				},
				TimeRange: TimeRange{
					From: "15m",
					To:   "now",
				},
				Query: Query{
					Evaluator: sdk.AlertEvaluator{
						Type:   ConvertEvaluatorAlias("NOVALUE"),
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
			want: &Alert{
				Metric: Metric{
					DashboardUID: "wpFnYRwGk",
					PanelID:      2,
					MetricName:   "bitrate",
				},
				TimeRange: TimeRange{
					From: "15m",
					To:   "now",
				},
				Query: Query{
					Evaluator: sdk.AlertEvaluator{
						Type:   ConvertEvaluatorAlias("WITHIN_RANGE"),
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
			got, err := NewAlert(tt.args.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAlert() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewAlert() got = %v, want %v", got, tt.want)
			}
		})
	}
}
