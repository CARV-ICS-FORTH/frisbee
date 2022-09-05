/*
Copyright 2022 ICS-FORTH.

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

package common

import (
	"os"

	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/utils/exec"
)

var (
	Kubectl string
	Helm    string

	NodeJS string
	NPM    string
)

func init() {
	kubectlPath, err := exec.New().LookPath("kubectl")
	ui.ExitOnError("Frisbee requires 'kubectl' to be installed in your system.", err)
	Kubectl = kubectlPath

	helmPath, err := exec.New().LookPath("helm")
	ui.ExitOnError("Frisbee requires 'helm' to be installed in your system.", err)
	Helm = helmPath

	nodejsPath, err := exec.New().LookPath("node")
	if err != nil {
		ui.Warn("Disable PDF exporter. It requires NodeJs to be install in your system ")
	}
	NodeJS = nodejsPath

	npmPath, err := exec.New().LookPath("npm")
	if err != nil {
		ui.Warn("Disable PDF exporter. It requires NodeJs to be install in your system ")
	}
	NPM = npmPath

	/*
		Ensure that the Installation Dir exists.
	*/
	if err := os.MkdirAll(InstallationDir, os.ModePerm); err != nil {
		ui.Fail(errors.Wrap(err, "Cannot open Frisbee cache"))
	}

	os.Setenv("PATH", os.Getenv("PATH")+":"+InstallationDir)
	os.Setenv("NODE_PATH", os.Getenv("NODE_PATH")+":"+InstallationDir)
}

func Hint(cmd *cobra.Command, msg string, sub ...string) {
	if ok, _ := cmd.Flags().GetBool("hints"); ok {
		ui.Success(msg, sub...)
	}
}

func Verbose(cmd *cobra.Command) bool {
	ok, _ := cmd.Flags().GetBool("verbose")

	return ok
}
