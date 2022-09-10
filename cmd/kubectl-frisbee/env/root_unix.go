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
	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"k8s.io/utils/exec"
	"os"
	"os/user"
	"path/filepath"
)

func (env *EnvSettings) CheckKubePerms() {
	// This function MUST NOT FAIL, as it is just a check for a common permissions problem.
	// If for some reason the function hits a stopping condition, it may panic. But only if
	// we can be sure that it is panicking because Helm cannot proceed.

	kc := env.KubeConfig
	if kc == "" {
		kc = os.Getenv("KUBECONFIG")
	}
	if kc == "" {
		u, err := user.Current()
		if err != nil {
			// No idea where to find KubeConfig, so return silently. Many helm commands
			// can proceed happily without a KUBECONFIG, so this is not a fatal error.
			return
		}
		kc = filepath.Join(u.HomeDir, ".kube", "config")
	}
	fi, err := os.Stat(kc)
	if err != nil {
		// DO NOT error if no KubeConfig is found. Not all commands require one.
		return
	}

	perm := fi.Mode().Perm()
	if perm&0040 > 0 {
		ui.Warn("Kubernetes configuration file is group-readable. This is insecure. Location: ", kc)
	}
	if perm&0004 > 0 {
		ui.Warn("Kubernetes configuration file is world-readable. This is insecure. Location: ", kc)
	}

	env.KubeConfig = kc
}


func (env *EnvSettings) LookupBinaries() {
	kubectlPath, err := exec.New().LookPath("kubectl")
	ui.ExitOnError("Frisbee requires 'kubectl' to be installed in your system.", err)
	env.kubectlPath = kubectlPath

	helmPath, err := exec.New().LookPath("helm")
	ui.ExitOnError("Frisbee requires 'helm' to be installed in your system.", err)
	env.helmPath = helmPath

	nodejsPath, err := exec.New().LookPath("node")
	if err != nil {
		ui.Warn("Disable PDF exporter. It requires NodeJs to be install in your system ")
	}
	env.nodejsPath = nodejsPath

	npmPath, err := exec.New().LookPath("npm")
	if err != nil {
		ui.Warn("Disable PDF exporter. It requires NodeJs to be install in your system ")
	}
	env.npmPath = npmPath
}
