package workflow_test

import (
	"testing"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/utils/lifecycle"
	"github.com/fnikolai/frisbee/controllers/workflow"
)

func TestAssertState(t *testing.T) {
	state := &lifecycle.Classifier{}
	state.Reset()

	// set jobs
	{
		var job v1alpha1.Service

		job.SetName("service0")
		job.SetReconcileStatus(v1alpha1.Lifecycle{
			Phase:   v1alpha1.PhaseSuccess,
			Reason:  "MockSuccess",
			Message: "mocking the success condition",
		})
		state.Classify(job.GetName(), &job)
	}
	{
		var job v1alpha1.Service

		job.SetName("service1")
		job.SetReconcileStatus(v1alpha1.Lifecycle{
			Phase:   v1alpha1.PhaseFailed,
			Reason:  "MockFailure",
			Message: "mocking the failure condition",
		})
		state.Classify(job.GetName(), &job)
	}

	type args struct {
		expr  string
		state *lifecycle.Classifier
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "empty expression",
			args: args{
				expr:  "",
				state: nil,
			},
			wantErr: false,
		},

		{
			name: "invalid objects",
			args: args{
				expr:  `{{.Runtime.IsSuccessful "service0"}} == true`,
				state: nil,
			},
			wantErr: true,
		},
		{
			name: "invalid expression",
			args: args{
				expr:  `{{.Runtime.IsSomethingWrong "service0"}} == true`,
				state: state,
			},
			wantErr: true,
		},
		{
			name: "test should pass",
			args: args{
				expr:  `{{.Runtime.IsSuccessful "service0"}} == true`,
				state: state,
			},
			wantErr: false,
		},
		{
			name: "test should fail",
			args: args{
				expr:  `{{.Runtime.IsFailed "service0"}} == true`,
				state: state,
			},
			wantErr: true,
		},

		{
			name: "valid synthetic expression",
			args: args{
				expr:  `{{.Runtime.IsSuccessful "service0"}} == true && {{.Runtime.IsFailed "service1"}} == true`,
				state: state,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := workflow.AssertState(tt.args.expr, &workflow.BuiltinState{Runtime: tt.args.state}); (err != nil) != tt.wantErr {
				t.Errorf("AssertState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
