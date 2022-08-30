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
	"os/exec"

	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/spf13/cobra"
)

var (
	Kubectl string
	Helm    string
)

func init() {
	kubectlPath, err := exec.LookPath("kubectl")
	if err != nil {
		panic(err)
	}

	helmPath, err := exec.LookPath("helm")
	if err != nil {
		panic(err)
	}

	Kubectl = kubectlPath
	Helm = helmPath
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
