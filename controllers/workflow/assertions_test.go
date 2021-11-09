package workflow_test

import (
	"testing"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/utils/lifecycle"
	"github.com/fnikolai/frisbee/controllers/workflow"
)

func Test_evaluate(t *testing.T) {
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

	builtin := workflow.BuiltinObjects{
		Workflow: &v1alpha1.Workflow{},
		Runtime:  state,
	}

	type args struct {
		expr    string
		objects workflow.BuiltinObjects
	}

	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "empty expression",
			args: args{
				expr:    "",
				objects: builtin,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "invalid objects",
			args: args{
				expr:    `{{.Runtime.IsSuccessful "service0"}} == true`,
				objects: workflow.BuiltinObjects{},
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "invalid expression",
			args: args{
				expr:    `{{.Runtime.IsSomethingWrong "service0"}} == true`,
				objects: builtin,
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "test should pass",
			args: args{
				expr:    `{{.Runtime.IsSuccessful "service0"}} == true`,
				objects: builtin,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "test should fail",
			args: args{
				expr:    `{{.Runtime.IsFailed "service0"}} == true`,
				objects: builtin,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "valid synthetic expression",
			args: args{
				expr:    `{{.Runtime.IsSuccessful "service0"}} == true && {{.Runtime.IsFailed "service1"}} == true`,
				objects: builtin,
			},
			want:    true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := workflow.Evaluate(tt.args.expr, tt.args.objects)
			if (err != nil) != tt.wantErr {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Evaluate() got = %v, want %v", got, tt.want)
			}
		})
	}
}
