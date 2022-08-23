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
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/exec"
	"os"
	"strings"
	"time"
)

var (
	ManagedNamespace = "app.kubernetes.io/managed-by=Frisbee"
)

var Execute = process.Execute

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

func NotFound(testName, out string) bool {
	return out == fmt.Sprintf("No resources found in %s namespace.\n", testName)
}

func GetPodLogs(cmd *cobra.Command, testName string, tail bool) error {
	command := []string{"logs", "-n", testName,
		"-l", fmt.Sprintf("%s", v1alpha1.LabelScenario),
		"-c", v1alpha1.MainContainerName,
		"--prefix=true",
		fmt.Sprintf("--follow=%t", tail),
	}

	getLogs := func() (done bool, err error) {
		out, err := process.LoggedExecuteInDir("", os.Stdout, Kubectl, command...)
		switch {
		case NotFound(testName, string(out)): // resource not found
			return false, nil
		case err != nil: // execution error
			return false, err
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
	command := []string{"get", "events", "-n", testName}

	out, err := process.Execute(Kubectl, command...)
	switch {
	case NotFound(testName, string(out)): // resource not found
		return nil
	case err != nil: // execution error
		return err
	default: // completed
		ui.Info(string(out))
		return nil
	}
}

func Dashboards(cmd *cobra.Command, testName string) error {
	command := []string{"get", "ingress",
		"-n", testName,
		"-l", fmt.Sprintf("%s", v1alpha1.LabelScenario),
	}

	// restricted format supported by Kubectl
	outputType := OutputType(cmd.Flag("output").Value.String())
	if outputType == "table" ||
		outputType == "json" ||
		outputType == "yaml" {
		command = append(command, "-o", string(outputType))
	}

	out, err := process.Execute(Kubectl, command...)
	switch {
	case NotFound(testName, string(out)): // resource not found
		return nil
	case err != nil: // execution error
		return err
	default: // completed
		ui.Info(string(out))
		return nil
	}
}

const (
	Scenarios      = "scenarios.frisbee.dev"
	Clusters       = "clusters.frisbee.dev"
	Services       = "services.frisbee.dev"
	Chaos          = "chaos.frisbee.dev"
	Cascades       = "cascades.frisbee.dev"
	Calls          = "calls.frisbee.dev"
	VirtualObjects = "virtualobjects.frisbee.dev"
	Templates      = "templates.frisbee.dev"
)

var CallableInspectionFields = strings.Join([]string{
	"custom-columns=Kind:.kind",
	"Job:.metadata.name",
	"Phase:.status.phase",
	"Stdout:.status.data.stdout",
	"Stderr:.status.data.stderr",
}, ",")

func GetCallableLogs(cmd *cobra.Command, testName string, tail bool) error {
	command := []string{"get", "-n", testName,
		"--show-kind=true",
		"-l", fmt.Sprintf("%s", v1alpha1.LabelScenario)}

	command = append(command, VirtualObjects)

	// restricted format supported by Kubectl
	outputType := OutputType(cmd.Flag("output").Value.String())
	if outputType == "table" ||
		outputType == "json" ||
		outputType == "yaml" {
		command = append(command, "-o", string(outputType))
	}

	command = append(command, "-o", CallableInspectionFields)

	out, err := process.Execute(Kubectl, command...)
	switch {
	case NotFound(testName, string(out)): // resource not found
		return nil
	case err != nil: // execution error
		return err
	default: // completed
		ui.Info(string(out))
		return nil
	}
}

var ResourceInspectionFields = strings.Join([]string{
	"custom-columns=Kind:.kind",
	"Job:.metadata.name",
	"Component:.metadata.labels.scenario\\.frisbee\\.dev\\/component",
	"Phase:.status.phase",
	"Reason:.status.reason",
	"Message:.status.message",
	// "Conditions:.status.conditions[*].type",
}, ",")

func GetFrisbeeResources(cmd *cobra.Command, testName string) error {
	command := []string{"get", "-n", testName,
		"--show-kind=true",
		"--sort-by=.metadata.creationTimestamp",
		// "--sort-by=.status.phase",
		"-l", fmt.Sprintf("%s", v1alpha1.LabelScenario)}

	command = append(command, strings.Join([]string{
		// Scenarios,
		Clusters, Services, Chaos, Cascades, Calls, VirtualObjects,
	}, ","))

	// restricted format supported by Kubectl
	outputType := OutputType(cmd.Flag("output").Value.String())
	if outputType == "table" ||
		outputType == "json" ||
		outputType == "yaml" {
		command = append(command, "-o", string(outputType))
	}

	command = append(command, "-o", ResourceInspectionFields)

	out, err := process.Execute(Kubectl, command...)
	switch {
	case NotFound(testName, string(out)): // resource not found
		return nil
	case err != nil: // execution error
		return err
	default: // completed
		ui.Info(string(out))
		return nil
	}
}

var TemplateInspectionFields = strings.Join([]string{
	"custom-columns=API:.apiVersion",
	"Kind:.kind",
	"Name:.metadata.name",
	"HelmRelease:.metadata.annotations.meta\\.helm\\.sh\\/release-name",
}, ",")

func GetTemplateResources(cmd *cobra.Command, testName string) error {
	command := []string{"get", "-n", testName}

	command = append(command, Templates)

	// restricted format supported by Kubectl
	outputType := OutputType(cmd.Flag("output").Value.String())
	if outputType == "table" ||
		outputType == "json" ||
		outputType == "yaml" {
		command = append(command, "-o", string(outputType))
	}

	command = append(command, "-o", TemplateInspectionFields)

	out, err := process.Execute(Kubectl, command...)
	switch {
	case NotFound(testName, string(out)): // resource not found
		return nil
	case err != nil: // execution error
		return err
	default: // completed
		ui.Info(string(out))
		return nil
	}
}

var K8SResourceInspectionFields = strings.Join([]string{
	"custom-columns=API:.apiVersion",
	"Kind:.kind",
	"Name:.metadata.name",
	"Action:.metadata.labels.scenario\\.frisbee\\.dev\\/action",
	"Component:.metadata.labels.scenario\\.frisbee\\.dev\\/component",
	"Phase*:.status.phase",
	"Reason*:.status.reason",
	"Message*:.status.message",
}, ",")

func GetK8sResources(cmd *cobra.Command, testName string) error {
	// Filter out pods that belong to a scenario
	command := []string{"get", "-n", testName, "-l", v1alpha1.LabelScenario, "all"}

	// restricted format supported by Kubectl
	outputType := OutputType(cmd.Flag("output").Value.String())
	if outputType == "table" ||
		outputType == "json" ||
		outputType == "yaml" {
		command = append(command, "-o", string(outputType))
	}

	command = append(command, "-o", K8SResourceInspectionFields)

	out, err := process.Execute(Kubectl, command...)
	switch {
	case NotFound(testName, string(out)): // resource not found
		return nil
	case err != nil: // execution error
		return err
	default: // completed
		ui.Info(string(out))
		return nil
	}
}

func OpenShell(testName string, podName string, shellArgs ...string) error {
	command := []string{"exec", "--stdin", "--tty", "-n", testName, podName}

	if len(shellArgs) == 0 {
		ui.Info("Interactive Shell:")
		command = append(command, "--", "/bin/sh")
	} else {
		ui.Info("Oneliner Shell:")
		command = append(command, "--")
		command = append(command, shellArgs...)
	}

	shell := exec.New().Command(Kubectl, command...)
	shell.SetStdin(os.Stdin)
	shell.SetStdout(os.Stdout)
	shell.SetStderr(os.Stderr)

	return shell.Run()
}

func RunTest(testName string, testFile string, dryrun bool) error {
	command := []string{"apply", "--wait",
		"-n", testName,
		"-f", testFile}

	if dryrun {
		command = append(command, "--dry-run=client")
	}

	_, err := process.Execute(Kubectl, command...)
	return err
}

func DeleteTests(selector string, testNames []string) error {
	command := []string{"delete", "namespace",
		// "--dry-run=client",
		"--cascade=foreground",
	}

	if selector != "" {
		command = append(command, "-l", ManagedNamespace)
	} else {
		command = append(command, testNames...)
	}

	_, err := process.LoggedExecuteInDir("", os.Stdout, Kubectl, command...)
	return err
}

func ListHelm(cmd *cobra.Command, testName string) error {
	command := []string{"list", "-n", testName}

	// output format supported by Helm
	outputType := OutputType(cmd.Flag("output").Value.String())
	if outputType == "table" ||
		outputType == "json" ||
		outputType == "yaml" {
		command = append(command, "-o", string(outputType))
	}

	out, err := process.Execute(Helm, command...)

	ui.Info(string(out))
	return err
}
