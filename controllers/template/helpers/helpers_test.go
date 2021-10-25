package thelpers

import (
	"testing"

	"github.com/fnikolai/frisbee/api/v1alpha1"
)

func TestGenerateSpecFromScheme(t *testing.T) {
	type args struct {
		tspec *v1alpha1.Scheme
	}

	// Examples: http://masterminds.github.io/sprig/string_slice.html
	tests := []struct {
		name    string
		args    args
		want    GenericSpec
		wantErr bool
	}{
		{
			name: "single",
			args: args{tspec: &v1alpha1.Scheme{
				Inputs: &v1alpha1.Inputs{Parameters: map[string]string{"slaves": "slave0"}},
				Spec:   `{{.Inputs.Parameters.slaves}}`,
			}},
			want:    GenericSpec("slave0"),
			wantErr: false,
		},
		{
			name: "space-newlines",
			args: args{tspec: &v1alpha1.Scheme{
				Inputs: &v1alpha1.Inputs{Parameters: map[string]string{"slaves": "slave0 slave1 slave2"}},
				Spec: `
cat > test.yml <<EOF
	{{- range splitList " " .Inputs.Parameters.slaves}}
	rs.Add( {{.}} )
	{{- end}}
EOF
`,
			}},
			want: GenericSpec(
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
			args: args{tspec: &v1alpha1.Scheme{
				Inputs: &v1alpha1.Inputs{Parameters: map[string]string{"slaves": "slave0,slave1,slave2"}},
				Spec:   `{{range splitList "," .Inputs.Parameters.slaves -}}{{.}}{{- end -}}`,
			}},
			want:    GenericSpec("slave0slave1slave2"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateSpecFromScheme(tt.args.tspec)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateSpecFromScheme() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GenerateSpecFromScheme() got = %v, want %v", got, tt.want)
			}
		})
	}
}
