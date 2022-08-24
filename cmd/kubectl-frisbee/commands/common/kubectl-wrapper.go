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
	"github.com/carv-ics-forth/frisbee/pkg/structure"
	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/kubeshop/testkube/pkg/process"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

func SetQuota(testName string, cpu, memory string) error {
	if cpu == "" && memory == "" {
		return nil
	}

	// Create quota specification
	scheme := v1alpha1.Scheme{
		Inputs: &v1alpha1.Inputs{Parameters: map[string]string{"CPU": cpu, "Memory": memory}},
		Spec: []byte(`
---
apiVersion: v1			
kind: ResourceQuota
metadata:
 name: mem-cpu-quota
spec:
 hard:
  {{- if .Inputs.Parameters.CPU}}
    requests.cpu: {{.Inputs.Parameters.CPU}}
    limits.cpu: {{.Inputs.Parameters.CPU}}
  {{- end}}
  {{- if .Inputs.Parameters.Memory}}
    requests.memory: {{.Inputs.Parameters.Memory}}
    limits.memory: {{.Inputs.Parameters.Memory}}
   {{- end}}
`)}

	quota, err := v1alpha1.ExprState(scheme.Spec).Evaluate(scheme)
	if err != nil {
		return errors.Wrapf(err, "cannot set quota")
	}

	// Create random file
	f, err := os.CreateTemp("/tmp", testName)
	if err != nil {
		return errors.Wrapf(err, "cannot create quota file")
	}

	if _, err := f.WriteString(quota); err != nil {
		return errors.Wrapf(err, "cannot store quota file")
	}

	if err := f.Sync(); err != nil {
		return errors.Wrapf(err, "cannot sync quota file")
	}

	command := []string{"apply", "--wait", "-n", testName, "-f", f.Name()}

	_, err = process.Execute(Kubectl, command...)
	return err
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

func NotFound(testName, out string) bool {
	return out == fmt.Sprintf("No resources found in %s namespace.\n", testName)
}

func GetPodLogs(testName string, tail bool, pods ...string) error {
	command := []string{"logs", "-n", testName,
		"-c", v1alpha1.MainContainerName,
		"--prefix=true",
	}

	if tail {
		// If tail, print everything
		command = append(command, fmt.Sprintf("--follow=true"))
	} else {
		command = append(command, fmt.Sprintf("--tail=5"))
	}

	switch {
	// Run with --logs all pod1 ...
	case len(pods) > 1 && structure.ContainsStrings(pods, "all"):
		return errors.Errorf("expects either 'all' or pod names")
	// Run with --logs all
	case len(pods) == 1 && pods[0] == "all":
		command = append(command, "-l", fmt.Sprintf("%s", v1alpha1.LabelScenario))
		// Run with --logs pod1 pod2 ...
	case pods != nil:
		command = append(command, pods...)
	default:
		return errors.Errorf("invalid pod arguments: '%v'", pods)
	}

	getLogs := func() (done bool, err error) {
		out, err := process.LoggedExecuteInDir("", os.Stdout, Kubectl, command...)
		switch {
		case NotFound(testName, string(out)): // resource not found
			return false, nil
		case err != nil: // execution error
			if tail { // on tail, we want to ignore errors and continue
				return false, nil
			}
			return false, err // without tail, we want to return the error immediately
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

func CreateNamespace(name string, labels ...string) error {
	// Create namespace
	command := []string{"create", "namespace", name}

	_, err := process.Execute(Kubectl, command...)
	if err != nil {
		return errors.Wrapf(err, "cannot create namespace")
	}

	// Label namespace
	if labels != nil {
		command = []string{"label", "namespaces", name, "--overwrite=true",
			strings.Join(labels, ",")}

		_, err := process.Execute(Kubectl, command...)
		if err != nil {
			return errors.Wrapf(err, "cannot label namespace")
		}
	}

	return nil
}

func DeleteNamespaces(selector string, testNames ...string) error {
	command := []string{"delete", "namespace",
		// "--dry-run=client",
		"--cascade=foreground",
	}

	if selector != "" {
		command = append(command, "-l", ManagedNamespace)
	} else {
		command = append(command, testNames...)
	}

	_, err := process.Execute(Kubectl, command...)
	return errors.Wrapf(err, "cannot delete namespace")
}

var K8SRemoveFinalizer = []string{
	"--patch=\\'[", "{",
	"op:", "remove,",
	"path:", "/metadata/finalizers",
	"}", "]\\'",
}

// ForceDelete iterates the Frisbee CRDs and remove its finalizers.
func ForceDelete(testName string) error {

	crds := []string{Scenarios, Clusters, Services, Chaos, Cascades, Calls, VirtualObjects, Templates}

	for _, crd := range crds {
		resourceQuery := []string{"get", crd, "-n", testName, "-o", "jsonpath='{.items[*].metadata.name}'"}

		resources, err := process.Execute(Kubectl, resourceQuery...)

		logrus.Warn("Resources:", resources, " a ", string(resources))

		switch {
		case err != nil:
			ui.Debug("skip operation", crd, err.Error())
		case string(resources) == "''":
			ui.Debug(crd, " resources are deleted.")
		default:
			patch := []string{"patch", crd, "-n", testName, "--type", "json"}
			patch = append(patch, K8SRemoveFinalizer...)
			patch = append(patch, string(resources))

			ui.Debug("Use patch", Kubectl, strings.Join(patch, " "))

			if _, err := process.Execute(Kubectl, patch...); err != nil {
				return errors.Wrapf(err, "cannot patch '%s' finalizers", crd)
			}
		}
	}

	return nil
}
