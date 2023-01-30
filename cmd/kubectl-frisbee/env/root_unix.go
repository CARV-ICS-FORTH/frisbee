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
	"os/user"
	"path/filepath"

	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"k8s.io/utils/exec"
)

// CheckKubePerms MUST NOT FAIL, as it is just a check for a common permissions' problem.
// If for some reason the function hits a stopping condition, it may panic. But only if
// we can be sure that it is panicking because Helm cannot proceed.
func (env *EnvironmentSettings) CheckKubePerms() {
	if env.Config.KubeConfig == nil || *env.Config.KubeConfig == "" {
		currentUser, err := user.Current()
		if err != nil {
			// No idea where to find KubeConfig, so return silently. Many helm commands
			// can proceed happily without a KUBECONFIG, so this is not a fatal error.
			return
		}

		defaultPath := filepath.Join(currentUser.HomeDir, ".kube", "config")
		env.Config.KubeConfig = &defaultPath
	}

	ui.Info("Using config:", *env.Config.KubeConfig)
}

func (env *EnvironmentSettings) LookupBinaries() {
	// kubectl
	kubectlPath, err := exec.New().LookPath("kubectl")
	ui.ExitOnError("Frisbee requires 'kubectl' to be installed in your system.", err)

	env.kubectlPath = kubectlPath

	// helm
	helmPath, err := exec.New().LookPath("helm")
	ui.ExitOnError("Frisbee requires 'helm' to be installed in your system.", err)

	env.helmPath = helmPath

	// stern
	sternPath, err := exec.New().LookPath("stern")
	ui.ExitOnError("Frisbee requires 'stern' to be installed in your system.", err)

	env.sternPath = sternPath

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
