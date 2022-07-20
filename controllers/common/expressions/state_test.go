package expressions

import (
	"testing"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common/lifecycle"
)

func setJobs(state *lifecycle.Classifier) {
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
	{
		var job v1alpha1.Service

		job.SetName("service2")
		job.SetReconcileStatus(v1alpha1.Lifecycle{
			Phase:   v1alpha1.PhaseRunning,
			Reason:  "MockRunning",
			Message: "mocking the running condition",
		})
		state.Classify(job.GetName(), &job)
	}
	{
		var job v1alpha1.Service

		job.SetName("service3")
		job.SetReconcileStatus(v1alpha1.Lifecycle{
			Phase:   v1alpha1.PhaseRunning,
			Reason:  "MockRunning",
			Message: "mocking the running condition",
		})
		state.Classify(job.GetName(), &job)
	}
}

func TestFiredState(t *testing.T) {
	state := lifecycle.Classifier{}
	state.Reset()

	setJobs(&state)

	type args struct {
		expr  v1alpha1.ExprState
		state lifecycle.ClassifierReader
	}

	tests := []struct {
		name     string
		args     args
		wantErr  bool
		wantPass bool
	}{
		{
			name: "empty expression",
			args: args{
				expr:  "",
				state: lifecycle.Classifier{},
			},
			wantErr:  false,
			wantPass: true,
		},

		{
			name: "invalid objects",
			args: args{
				expr:  `{{.IsSuccessful "service0"}} == true`,
				state: lifecycle.Classifier{},
			},
			wantErr:  false,
			wantPass: false,
		},
		{
			name: "invalid expression",
			args: args{
				expr:  `{{.IsSomethingWrong "service0"}} == true`,
				state: state,
			},
			wantErr:  true,
			wantPass: false,
		},
		{
			name: "test should pass",
			args: args{
				expr:  `{{.IsSuccessful "service0"}} == true`,
				state: state,
			},
			wantErr:  false,
			wantPass: true,
		},
		{
			name: "test should fail",
			args: args{
				expr:  `{{.IsFailed "service0"}} == true`,
				state: state,
			},
			wantErr:  false,
			wantPass: false,
		},
		{
			name: "valid synthetic expression",
			args: args{
				expr:  `{{.IsSuccessful "service0"}} == true && {{.IsFailed "service1"}} == true`,
				state: state,
			},
			wantErr:  false,
			wantPass: true,
		},
		{
			name: "valid numeric comparison",
			args: args{
				expr:  `{{.NumRunningJobs}} == 2`,
				state: state,
			},
			wantErr:  false,
			wantPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, pass, err := FiredState(tt.args.expr, tt.args.state)
			if (err != nil) != tt.wantErr {
				t.Errorf("FiredState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if pass != tt.wantPass {
				t.Errorf("FiredState() pass = %v, want %v", pass, tt.wantPass)
			}
		})
	}
}
