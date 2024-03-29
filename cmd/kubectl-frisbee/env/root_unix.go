/*
Copyright 2022-2023 ICS-FORTH.

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

package env

import (
	"github.com/kubeshop/testkube/pkg/ui"
	"k8s.io/utils/exec"
)

func (env *EnvironmentSettings) LookupBinaries() {
	// kubectl
	kubectlPath, err := exec.New().LookPath("kubectl")
	ui.ExitOnError("Frisbee requires 'kubectl' to be installed in your system.", err)

	env.kubectlPath = kubectlPath

	// helm
	helmPath, err := exec.New().LookPath("helm")
	ui.ExitOnError("Frisbee requires 'helm' to be installed in your system.", err)

	env.helmPath = helmPath

	// nodejs
	nodejsPath, err := exec.New().LookPath("node")
	if err != nil {
		ui.Warn("Disable PDF exporter due to missing dependency.", "NodeJs")
	}

	env.nodejsPath = nodejsPath

	// npm
	npmPath, err := exec.New().LookPath("npm")
	if err != nil {
		ui.Warn("Disable PDF exporter due to missing dependency.", "NPM")
	}

	env.npmPath = npmPath
}
