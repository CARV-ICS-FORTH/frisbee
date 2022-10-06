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

package env

import (
	"os"
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

	kubeConfigInfo, err := os.Stat(*env.Config.KubeConfig)
	if err != nil {
		// DO NOT error if no KubeConfig is found. Not all commands require one.
		return
	}

	perm := kubeConfigInfo.Mode().Perm()
	if perm&0o040 > 0 {
		ui.Warn("Kubernetes configuration file is group-readable. This is insecure. Location: ", *env.Config.KubeConfig)
	}

	if perm&0o004 > 0 {
		ui.Warn("Kubernetes configuration file is world-readable. This is insecure. Location: ", *env.Config.KubeConfig)
	}
}

func (env *EnvironmentSettings) LookupBinaries() {
	kubectlPath, err := exec.New().LookPath("kubectl")
	ui.ExitOnError("Frisbee requires 'kubectl' to be installed in your system.", err)

	env.kubectlPath = kubectlPath

	helmPath, err := exec.New().LookPath("helm")
	ui.ExitOnError("Frisbee requires 'helm' to be installed in your system.", err)

	env.helmPath = helmPath

	nodejsPath, err := exec.New().LookPath("node")
	if err != nil {
		ui.Warn("Disable PDF exporter. It requires 'NodeJs' to be install in your system.")
	}

	env.nodejsPath = nodejsPath

	npmPath, err := exec.New().LookPath("npm")
	if err != nil {
		ui.Warn("Disable PDF exporter. It requires 'NPM' to be install in your system.")
	}

	env.npmPath = npmPath
}
