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
	"fmt"
	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/kubeshop/testkube/pkg/process"
	"k8s.io/apimachinery/pkg/util/wait"
	"os"
	"time"
)

func WaitTestPending(testName string) error {
	command := []string{"wait", "-n", testName,
		"scenario", "--all=true",
		"--for=jsonpath=.status.phase=Pending",
	}

	out, err := process.Execute(Kubectl, command...)
	if err != nil {
		return err
	}

	ui.Info("Waiting for test initialization", string(out))

	return nil
}

func WaitTestSuccess(testName string) error {
	command := []string{"wait", "-n", testName,
		"scenario", "--all=true",
		"--for=jsonpath=.status.phase=Success",
	}

	out, err := process.Execute(Kubectl, command...)
	if err != nil {
		return err
	}

	ui.Info("Waiting for test completion", string(out))

	return nil
}

var (
	pollTimeout  = 24 * time.Hour
	pollInterval = 1 * time.Second
)

func Logs(testName string, tail bool) error {
	command := []string{"logs", "-n", testName,
		"-l", fmt.Sprintf("%s=%s", v1alpha1.LabelComponent, v1alpha1.ComponentSUT),
		"-c", v1alpha1.MainContainerName,
		"--prefix=true",
		fmt.Sprintf("--follow=%t", tail),
	}

	errNotFound := fmt.Sprintf("No resources found in %s namespace.\n", testName)

	getLogs := func() (done bool, err error) {
		out, err := process.LoggedExecuteInDir("", os.Stdout, Kubectl, command...)
		switch {
		case err != nil: // abort
			return false, nil
		case string(out) == errNotFound:
			return false, nil
		default: // completed
			return true, nil
		}
	}

	if tail {
		return wait.PollImmediate(pollInterval, pollTimeout, getLogs)
	} else {
		_, err := getLogs()
		return err
	}
}

func Events(testName string) error {
	command := []string{"get", "events",
		"-n", testName,
	}

	out, err := process.Execute(Kubectl, command...)
	if err != nil {
		return err
	}

	ui.Info(string(out))

	return nil
}

func Dashboards(testName string) error {
	command := []string{"get", "ingress",
		"-n", testName,
		"-l", fmt.Sprintf("%s=%s", v1alpha1.LabelComponent, v1alpha1.ComponentSUT),
	}

	out, err := process.Execute(Kubectl, command...)
	if err != nil {
		return err
	}

	ui.Info(string(out))

	return nil
}
