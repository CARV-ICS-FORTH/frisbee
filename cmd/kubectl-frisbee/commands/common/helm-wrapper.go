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

package common

import (
	"os"
	"strings"

	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/env"
	"github.com/carv-ics-forth/frisbee/pkg/process"
)

func HelmIgnoreNotFound(err error) error {
	if err != nil && strings.Contains(err.Error(), "release: not found") {
		return nil
	}

	return err
}

func Helm(testName string, command ...string) ([]byte, error) {
	var helmArgs []string

	if env.Default.KubeConfigPath != "" {
		helmArgs = append(helmArgs, "--kubeconfig", env.Default.KubeConfigPath)
	}

	if env.Default.Debug {
		helmArgs = append(helmArgs, "--debug")
	}

	if testName != "" {
		helmArgs = append(helmArgs, "--namespace", testName)
	}

	helmArgs = append(helmArgs, command...)

	return process.Execute(env.Default.Helm(), helmArgs...)
}

func LoggedHelm(testName string, command ...string) ([]byte, error) {
	var helmArgs []string

	if env.Default.KubeConfigPath != "" {
		helmArgs = append(helmArgs, "--kubeconfig", env.Default.KubeConfigPath)
	}

	if testName != "" {
		helmArgs = append(helmArgs, "--namespace", testName)
	}

	helmArgs = append(helmArgs, command...)

	return process.LoggedExecuteInDir("", os.Stdout, env.Default.Helm(), helmArgs...)
}
