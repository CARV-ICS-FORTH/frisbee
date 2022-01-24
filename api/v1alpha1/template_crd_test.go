package v1alpha1

import (
	"testing"
)

func TestFromTemplate_Validate(t1 *testing.T) {
	type fields struct {
		TemplateRef string
		Instances   int
		Inputs      []map[string]string
	}

	type args struct {
		allowMultipleInputs bool
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "empty",
			fields: fields{
				TemplateRef: "",
				Instances:   0,
				Inputs:      nil,
			},
			args:    args{false},
			wantErr: true,
		},
		{
			name: "single-input-multiple-parameters",
			fields: fields{
				TemplateRef: "validName",
				Instances:   0,
				Inputs:      []map[string]string{{"keyA": "valA", "keyB": "valB"}},
			},
			args:    args{false},
			wantErr: false,
		},
		{
			name: "multiple-inputs-single-parameters",
			fields: fields{
				TemplateRef: "validName",
				Instances:   0,
				Inputs:      []map[string]string{{"keyA": "valA"}, {"keyB": "valB"}},
			},
			args:    args{true},
			wantErr: false,
		},
		{
			name: "multiple-inputs-multiple-parameters",
			fields: fields{
				TemplateRef: "validName",
				Instances:   0,
				Inputs:      []map[string]string{{"keyA": "valA", "keyB": "valB"}, {"keyB": "valB", "valB": "valC"}},
			},
			args:    args{true},
			wantErr: false,
		},
		{
			name: "violation-inputConstraints",
			fields: fields{
				TemplateRef: "validName",
				Instances:   0,
				Inputs:      []map[string]string{{"keyA": "valA"}, {"keyB": "valB"}},
			},
			args:    args{false},
			wantErr: true,
		},
		{
			name: "copiedInputsPerInstance",
			fields: fields{
				TemplateRef: "validName",
				Instances:   3,
				Inputs:      []map[string]string{{"keyA": "valA", "keyB": "valB"}},
			},
			args:    args{true},
			wantErr: false,
		},
		{
			name: "inconsistentInputsPerInstance",
			fields: fields{
				TemplateRef: "validName",
				Instances:   3,
				Inputs:      []map[string]string{{"keyA": "valA"}, {"keyB": "valB"}},
			},
			args:    args{true},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &GenerateFromTemplate{
				TemplateRef:  tt.fields.TemplateRef,
				MaxInstances: tt.fields.Instances,
				Inputs:       tt.fields.Inputs,
			}
			if err := t.Validate(tt.args.allowMultipleInputs); (err != nil) != tt.wantErr {
				t1.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
