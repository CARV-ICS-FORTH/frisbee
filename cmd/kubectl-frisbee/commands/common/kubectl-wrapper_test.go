package common

import (
	"testing"

	_ "github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/env"
)

func TestSetQuota(t *testing.T) {
	namespace := "my-test"

	if err := CreateNamespace(namespace); err != nil {
		t.Fatalf("cannot create namespace '%s'. err: '%s'", namespace, err)
	}

	defer func() {
		if err := DeleteNamespaces("", namespace); err != nil {
			t.Fatalf("cannot delete namespace '%s'. err: '%s'", namespace, err)
		}
	}()

	type args struct {
		testName string
		cpu      string
		memory   string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "Default",
			args:    args{testName: namespace, cpu: "", memory: ""},
			wantErr: false,
		},
		{
			name:    "CPU-Only",
			args:    args{testName: namespace, cpu: "0.1", memory: ""},
			wantErr: false,
		},

		{
			name:    "Memory-Only",
			args:    args{testName: namespace, cpu: "", memory: "10Mi"},
			wantErr: false,
		},
		{
			name:    "Both",
			args:    args{testName: namespace, cpu: "0.1", memory: "10Mi"},
			wantErr: false,
		},
		{
			name:    "Invalid",
			args:    args{testName: namespace, cpu: "0.1", memory: "10Mb"},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SetQuota(tt.args.testName, tt.args.cpu, tt.args.memory); (err != nil) != tt.wantErr {
				t.Errorf("SetQuota() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
