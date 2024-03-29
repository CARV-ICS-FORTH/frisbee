package fuzz_test

import (
	"testing"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
)

func TestFromTemplate_Validate(t1 *testing.T) {
	type fields struct {
		TemplateRef string
		Instances   int
		Inputs      []v1alpha1.UserInputs
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
				Inputs: []v1alpha1.UserInputs{
					{"keyA": v1alpha1.ParameterValue("valA"), "keyB": v1alpha1.ParameterValue("valB")},
				},
			},
			args:    args{false},
			wantErr: false,
		},
		{
			name: "multiple-inputs-single-parameters",
			fields: fields{
				TemplateRef: "validName",
				Instances:   0,
				Inputs: []v1alpha1.UserInputs{
					{"keyA": v1alpha1.ParameterValue("valA")},
					{"keyB": v1alpha1.ParameterValue("valB")},
				},
			},
			args:    args{true},
			wantErr: false,
		},
		{
			name: "multiple-inputs-multiple-parameters",
			fields: fields{
				TemplateRef: "validName",
				Instances:   0,
				Inputs: []v1alpha1.UserInputs{
					{"keyA": v1alpha1.ParameterValue("valA"), "keyB": v1alpha1.ParameterValue("valB")},
					{"keyB": v1alpha1.ParameterValue("valB"), "valB": v1alpha1.ParameterValue("valC")},
				},
			},
			args:    args{true},
			wantErr: false,
		},
		{
			name: "violation-inputConstraints",
			fields: fields{
				TemplateRef: "validName",
				Instances:   0,
				Inputs: []v1alpha1.UserInputs{
					{"keyA": v1alpha1.ParameterValue("valA")},
					{"keyB": v1alpha1.ParameterValue("valB")},
				},
			},
			args:    args{false},
			wantErr: true,
		},
		{
			name: "copiedInputsPerInstance",
			fields: fields{
				TemplateRef: "validName",
				Instances:   3,
				Inputs: []v1alpha1.UserInputs{
					{"keyA": v1alpha1.ParameterValue("valA"), "keyB": v1alpha1.ParameterValue("valB")},
				},
			},
			args:    args{true},
			wantErr: false,
		},
		{
			name: "inconsistentInputsPerInstance",
			fields: fields{
				TemplateRef: "validName",
				Instances:   3,
				Inputs: []v1alpha1.UserInputs{
					{"keyA": v1alpha1.ParameterValue("valA")},
					{"keyB": v1alpha1.ParameterValue("valB")},
				},
			},
			args:    args{true},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &v1alpha1.GenerateObjectFromTemplate{
				TemplateRef:  tt.fields.TemplateRef,
				MaxInstances: tt.fields.Instances,
				Inputs:       tt.fields.Inputs,
			}
			if err := t.Prepare(tt.args.allowMultipleInputs); (err != nil) != tt.wantErr {
				t1.Errorf("Prepare() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
