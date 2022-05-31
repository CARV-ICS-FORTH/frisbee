/*
Copyright 2021 ICS-FORTH.

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

import (
	"testing"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	templateutils "github.com/carv-ics-forth/frisbee/controllers/template/utils"
)

func TestGenerateSpecFromScheme(t *testing.T) {
	type args struct {
		tspec *templateutils.Scheme
	}

	// Examples: http://masterminds.github.io/sprig/string_slice.html
	tests := []struct {
		name    string
		args    args
		want    templateutils.GenericSpec
		wantErr bool
	}{
		{
			name: "single",
			args: args{tspec: &templateutils.Scheme{
				Inputs: &v1alpha1.Inputs{Parameters: map[string]string{"slaves": "slave0"}},
				Spec:   []byte(`{{.Inputs.Parameters.slaves}}`),
			}},
			want:    templateutils.GenericSpec("slave0"),
			wantErr: false,
		},
		{
			name: "space-newlines",
			args: args{tspec: &templateutils.Scheme{
				Inputs: &v1alpha1.Inputs{Parameters: map[string]string{"slaves": "slave0 slave1 slave2"}},
				Spec: []byte(`
cat > test.yml <<EOF
	{{- range splitList " " .Inputs.Parameters.slaves}}
	rs.Add( {{.}} )
	{{- end}}
EOF
`),
			}},
			want: templateutils.GenericSpec(
				`
cat > test.yml <<EOF
	rs.Add( slave0 )
	rs.Add( slave1 )
	rs.Add( slave2 )
EOF
`),
			wantErr: false,
		},
		{
			name: "comma-nonewlines",
			args: args{tspec: &templateutils.Scheme{
				Inputs: &v1alpha1.Inputs{Parameters: map[string]string{"slaves": "slave0,slave1,slave2"}},
				Spec:   []byte(`{{range splitList "," .Inputs.Parameters.slaves -}}{{.}}{{- end -}}`),
			}},
			want:    templateutils.GenericSpec("slave0slave1slave2"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := templateutils.Evaluate(tt.args.tspec)

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
