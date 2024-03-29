/*
Copyright 2021-2023 ICS-FORTH.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils_test

/*
func TestGenerateSpecFromScheme(t *testing.T) {
	type args struct {
		tspec *v1alpha1.Scheme
	}

	// Examples: http://masterminds.github.io/sprig/string_slice.html
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "single",
			args: args{tspec: &v1alpha1.Scheme{
				Inputs: &v1alpha1.Inputs{Parameters: v1alpha1.UserInputs{"slaves": v1alpha1.ParameterValue("slave0")}},
				Spec:   []byte(`{{.inputs.parameters.slaves}}`),
			}},
			want:    "slave0",
			wantErr: false,
		},
		{
			name: "space-newlines",
			args: args{tspec: &v1alpha1.Scheme{
				Inputs: &v1alpha1.Inputs{Parameters: v1alpha1.UserInputs{"slaves": v1alpha1.ParameterValue("slave0 slave1 slave2")}},
				Spec: []byte(`
cat > test.yml <<EOF
	{{- range splitList " " .Inputs.Parameters.slaves}}
	rs.Add( {{.}} )
	{{- end}}
EOF
`),
			}},
			want: `
cat > test.yml <<EOF
	rs.Add( slave0 )
	rs.Add( slave1 )
	rs.Add( slave2 )
EOF
`,
			wantErr: false,
		},
		{
			name: "comma-nonewlines",
			args: args{tspec: &v1alpha1.Scheme{
				Inputs: &v1alpha1.Inputs{Parameters: v1alpha1.UserInputs{"slaves": v1alpha1.ParameterValue("slave0,slave1,slave2")}},
				Spec:   []byte(`{{range splitList "," .Inputs.Parameters.slaves -}}{{.}}{{- end -}}`),
			}},
			want:    "slave0slave1slave2",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.args.tspec.Evaluate()

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


*/
