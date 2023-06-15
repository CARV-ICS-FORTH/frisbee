package env

import (
	"bytes"
	"fmt"

	"github.com/dimiro1/banner"
	"github.com/kubeshop/testkube/pkg/ui"
)

func logo() string {
	buf := bytes.NewBuffer(nil)

	/*
			templ := `{{ .Title "Frisbee" "" 4 }}
		   {{ .AnsiColor.BrightCyan }}The title will be ascii and indented 4 spaces{{ .AnsiColor.Default }}
		   GoVersion: {{ .GoVersion }}
		   GOOS: {{ .GOOS }}
		   GOARCH: {{ .GOARCH }}
		   NumCPU: {{ .NumCPU }}
		   GOPATH: {{ .GOPATH }}
		   GOROOT: {{ .GOROOT }}
		   Compiler: {{ .Compiler }}
		   ENV: {{ .Env "GOPATH" }}
		   Now: {{ .Now "Monday, 2 Jan 2006" }}
		   {{ .AnsiColor.BrightGreen }}This text will appear in Green
		   {{ .AnsiColor.BrightRed }}This text will appear in Red{{ .AnsiColor.Default }}`
	*/

	templ := `
{{ .AnsiColor.BrightRed }}
{{ .Title "Frisbee" "" 4 }}
{{ .AnsiColor.BrightGreen }}
	`

	banner.InitString(buf, true, true, templ)

	return buf.String()
}

func Logo() {
	fmt.Fprint(ui.Writer, ui.Blue(logo()))
	fmt.Fprintln(ui.Writer)

	ui.Success("Kubernetes API:", Default.KubeConfig.Host)
}
