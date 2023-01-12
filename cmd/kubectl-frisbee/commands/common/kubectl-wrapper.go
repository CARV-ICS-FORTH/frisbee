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
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/env"
	"github.com/carv-ics-forth/frisbee/pkg/process"
	"github.com/carv-ics-forth/frisbee/pkg/structure"
	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/exec"
)

const (
	ManagedNamespace = "app.kubernetes.io/managed-by=Frisbee"
	pollTimeout      = 24 * time.Hour
	pollInterval     = 5 * time.Second
)

const (
	// Regex Validated over https://regex101.com/r/eXgekO/1

	NotReadyRegex        = `.* container "(\w+)" in pod "(.*)" is waiting to start: (\w+)`
	NoPodsFoundReg       = `.* pods "\w+" not found`
	NotResourcesFoundReg = `No resources found*`
	NotFound             = `Error from server (NotFound)`
)

func ErrNotFound(out []byte) bool {
	{ // First form
		if strings.Contains(string(out), NotFound) {
			return true
		}
	}

	{ // Second form
		match, err := regexp.Match(NotResourcesFoundReg, out)
		if err != nil {
			panic("unhandled output")
		}

		if match {
			return true
		}
	}

	{ // Third form
		match, err := regexp.Match(NoPodsFoundReg, out)
		if err != nil {
			panic("unhandled output")
		}

		if match {
			return true
		}
	}

	return false
}

func ErrContainerNotReady(out []byte) bool {
	match, err := regexp.Match(NotReadyRegex, out)
	if err != nil {
		panic("unhandled output")
	}

	return match
}

func Kubectl(testName string, arguments ...string) ([]byte, error) {
	arguments = append(arguments, "--kubeconfig", *env.Default.Config.KubeConfig)

	if testName != "" {
		arguments = append(arguments, "-n", testName)
	}

	return process.Execute(env.Default.Kubectl(), arguments...)
}

func LoggedKubectl(testName string, arguments ...string) ([]byte, error) {
	arguments = append(arguments, "--kubeconfig", *env.Default.Config.KubeConfig)

	if testName != "" {
		arguments = append(arguments, "-n", testName)
	}

	return process.LoggedExecuteInDir("", os.Stdout, env.Default.Kubectl(), arguments...)
}

func HelmIgnoreNotFound(err error) error {
	if err != nil && strings.Contains(err.Error(), "release: not found") {
		return nil
	}

	return err
}

func Helm(testName string, arguments ...string) ([]byte, error) {
	arguments = append(arguments, "--kubeconfig", *env.Default.Config.KubeConfig)

	if testName != "" {
		arguments = append(arguments, "-n", testName)
	}

	return process.Execute(env.Default.Helm(), arguments...)
}

func LoggedHelm(testName string, arguments ...string) ([]byte, error) {
	arguments = append(arguments, "--kubeconfig", *env.Default.Config.KubeConfig)

	if testName != "" {
		arguments = append(arguments, "-n", testName)
	}

	return process.LoggedExecuteInDir("", os.Stdout, env.Default.Helm(), arguments...)
}

func setOutput(command []string) []string {
	outputType := OutputType(env.Default.OutputType)
	if outputType == "table" ||
		outputType == "json" ||
		outputType == "yaml" {
		command = append(command, "-o", string(outputType))
	}

	return command
}

/*
******************************************************************

					Frisbee Resources

******************************************************************
*/

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

var FrisbeeResourceInspectionFields = strings.Join([]string{
	"custom-columns=Name:.metadata.namespace",
	"Kind:.kind",
	"Job:.metadata.name",
	"Component:.metadata.labels.scenario\\.frisbee\\.dev\\/component",
	"Phase:.status.phase",
	"Reason:.status.reason",
	"Message:.status.message",
	// "Conditions:.status.conditions[*].type",
}, ",")

const EmptyResourceInspectionFields = "Name   Kind   Job   Component   Phase   Reason   Message"

func GetFrisbeeResources(testName string, watch bool) error {
	command := []string{
		"get",
		"--show-kind=true",
		"-l", v1alpha1.LabelScenario,
		"-o", FrisbeeResourceInspectionFields,
	}

	command = setOutput(command)

	// decide the presentation method
	if watch {
		// monitor the top-level scenario overview
		command = append(command, "--watch=true", Scenarios)

		_, err := LoggedKubectl(testName, command...)

		return err
	}

	// monitor all sub-resources, sorted by creation Timestamp
	command = append(command, "--sort-by=.metadata.creationTimestamp", strings.Join([]string{
		Clusters, Services, Chaos, Cascades, Calls, VirtualObjects,
	}, ","))
	out, err := Kubectl(testName, command...)

	if strings.Contains(string(out), EmptyResourceInspectionFields) {
		return nil
	}

	ui.Info(string(out))

	return err
}

var TemplateInspectionFields = strings.Join([]string{
	"custom-columns=Chart:.metadata.annotations.meta\\.helm\\.sh\\/release-name",
	"Template:.metadata.name",
	"Parameters:spec.inputs.parameters",
}, ",")

const EmptyTemplateResources = "Chart   Template   Parameters"

func GetTemplateResources(testName string) error {
	command := []string{"get"}

	command = append(command, Templates)

	command = append(command, "-o", TemplateInspectionFields)

	out, err := Kubectl(testName, command...)
	if ErrNotFound(out) || strings.Contains(string(out), EmptyTemplateResources) {
		return nil
	}

	ui.Info(string(out))

	return err
}

func WaitForCondition(testName string, condition v1alpha1.ConditionType, timeout string) error {
	command := []string{
		"wait", "scenario", "--all=true",
		"--for=condition=" + condition.String(),
		"--timeout=" + timeout,
	}

	_, err := LoggedKubectl(testName, command...)

	return err
}

/*
******************************************************************

					CHAOS Resources

******************************************************************
*/

const (
	NetworkChaos = "networkchaos.chaos-mesh.org"
	PodChaos     = "podchaos.chaos-mesh.org"
	IOChaos      = "iochaos.chaos-mesh.org"
	KernelChaos  = "kernelchaos.chaos-mesh.org"
	TimeChaos    = "timechaos.chaos-mesh.org"
)

var ChaosResourceInspectionFields = strings.Join([]string{
	"custom-columns=Kind:.kind",
	"Job:.metadata.name",
	"InjectionTime:.metadata.creationTimestamp",
	"Phase:.status.experiment.desiredPhase",
	"Target:.status.experiment.containerRecords[*].id",
}, ",")

const EmptyChaosResourceInspectionFields = "Kind   Job   InjectionTime   Phase   Target"

func GetChaosResources(testName string) error {
	command := []string{
		"get",
		"--show-kind=true",
		"--sort-by=.metadata.creationTimestamp",
		"-l", v1alpha1.LabelScenario,
	}

	command = append(command, strings.Join([]string{NetworkChaos, PodChaos, IOChaos, KernelChaos, TimeChaos}, ","))

	command = setOutput(command)

	command = append(command, "-o", ChaosResourceInspectionFields)

	out, err := Kubectl(testName, command...)
	if ErrNotFound(out) || strings.Contains(string(out), EmptyChaosResourceInspectionFields) {
		return nil
	}

	ui.Info(string(out))

	return err
}

/*
******************************************************************

					K8s Resources

******************************************************************
*/

func GetK8sEvents(testName string) error {
	out, err := Kubectl(testName, "get", "events")
	if ErrNotFound(out) {
		return nil
	}

	ui.Info(string(out))

	return err
}

const (
	K8PODs            = "pods"
	K8PVCs            = "persistentvolumeclaims"
	K8PVs             = "persistentvolumes"
	K8SStorageClasses = "storageclasses.storage.k8s.io"
)

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

const EmptyK8SResourceInspectionFields = "API   Kind   Name   Action   Component   Phase*   Reason*   Message*"

func GetK8sResources(testName string) error {
	// Filter out pods that belong to a scenario
	command := []string{"get", "--show-kind=true", "-l", v1alpha1.LabelScenario}

	command = append(command, strings.Join([]string{K8PODs, K8PVCs, K8PVs, K8SStorageClasses}, ","))

	command = setOutput(command)

	command = append(command, "-o", K8SResourceInspectionFields)

	out, err := Kubectl(testName, command...)
	if ErrNotFound(out) || strings.Contains(string(out), EmptyK8SResourceInspectionFields) {
		return nil
	}

	ui.Info(string(out))

	return err
}

const (
	FilterSYS = string(v1alpha1.LabelComponent + "=" + v1alpha1.ComponentSys)
	FilterSUT = string(v1alpha1.LabelComponent + "=" + v1alpha1.ComponentSUT)
)

/*
GetPodLogs provides convenience on printing the logs from prods.
Filter query:
  - Run with '--all'.
  - Run with '--logs all pod1 ...'.
  - Run with '--logs all'.
  - Run with '--logs SYS'.
  - Run with '--logs SUT'.
  - Run with '--logs pod1 pod2 ...'.
*/
func GetPodLogs(testName string, tail bool, lines int, pods ...string) error {
	command := []string{
		"logs",
		"-c", v1alpha1.MainContainerName,
		"--prefix=true",
		fmt.Sprintf("--tail=%d", lines),
	}

	switch {
	case len(pods) == 0:
		command = append(command, "-l", v1alpha1.LabelScenario)
	case len(pods) > 1 && structure.ContainsStrings(pods, "all"):
		return errors.Errorf("expects either 'all' or pod names")
	case len(pods) == 1 && pods[0] == "all":
		command = append(command, "-l", v1alpha1.LabelScenario)
	case len(pods) == 1 && pods[0] == string(v1alpha1.ComponentSys):
		command = append(command, "-l", strings.Join([]string{v1alpha1.LabelScenario, FilterSYS}, ","))
	case len(pods) == 1 && pods[0] == string(v1alpha1.ComponentSUT):
		command = append(command, "-l", strings.Join([]string{v1alpha1.LabelScenario, FilterSUT}, ","))
	case pods != nil:
		command = append(command, pods...)
	default:
		panic(errors.Errorf("invalid GetPodLogs arguments: '%v'", pods))
	}

	// set output and retry policy
	if tail {
		command = append(command, "--ignore-errors=false", "--follow=true")

		return wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
			out, err := LoggedKubectl(testName, command...)

			switch {
			case len(out) == 0, ErrNotFound(out), ErrContainerNotReady(out): // resource initialization
				// ui.Info("Waiting for pods to become ready ...", string(out))
				return false, nil
			case err != nil: // abort
				return false, err
			default: // ok
				// Output printing is not required as it is printed by the os.Stdout
				return true, nil
			}
		})
	}

	command = append(command, "--ignore-errors=true")
	out, err := Kubectl(testName, command...)

	switch {
	case len(out) == 0, ErrNotFound(out), ErrContainerNotReady(out): // resource initialization
		// ui.Info("Waiting for pods to become ready ...", string(out))
		return nil
	case err != nil: // abort
		return err
	default: // ok
		ui.Info(string(out))

		return nil
	}
}

func OpenShell(testName string, podName string, shellArgs ...string) error {
	command := []string{
		"exec",
		"--kubeconfig", *env.Default.Config.KubeConfig,
		"--stdin", "--tty", "-n", testName, podName,
	}

	if len(shellArgs) == 0 {
		ui.Info("Interactive Shell:")

		command = append(command, "--", "/bin/sh")
	} else {
		ui.Info("Oneliner Shell:")
		command = append(command, "--")
		command = append(command, shellArgs...)
	}

	shell := exec.New().Command(env.Default.Kubectl(), command...)
	shell.SetStdin(os.Stdin)
	shell.SetStdout(os.Stdout)
	shell.SetStderr(os.Stderr)

	return shell.Run()
}

type ValidationMode uint8

const (
	ValidationNone ValidationMode = iota
	ValidationClient
	ValidationServer
)

func RunTest(testName string, testFile string, mode ValidationMode) error {
	command := []string{"apply", "--wait", "-f", testFile}

	switch mode {
	case ValidationClient:
		command = append(command, "--dry-run=client")
	case ValidationServer:
		command = append(command, "--dry-run=server")
	}

	out, err := Kubectl(testName, command...)

	ui.Debug(string(out))

	return err
}

func Dashboards(testName string) error {
	command := []string{
		"get", "ingress",
		"-l", v1alpha1.LabelScenario,
	}

	command = setOutput(command)

	out, err := Kubectl(testName, command...)
	if ErrNotFound(out) {
		return nil
	}

	ui.Info(string(out))

	return err
}

func CreateNamespace(name string, labels ...string) error {
	// Create namespace
	command := []string{"create", "namespace", name}

	_, err := Kubectl("", command...)
	if err != nil {
		return errors.Wrapf(err, "cannot create namespace")
	}

	// Label namespace
	return LabelNamespace(name, labels...)
}

func LabelNamespace(name string, labels ...string) error {
	// Label namespace
	if labels != nil {
		command := []string{
			"label", "namespaces", name, "--overwrite=true",
			strings.Join(labels, ","),
		}

		_, err := Kubectl("", command...)
		if err != nil {
			return errors.Wrapf(err, "cannot label namespace")
		}
	}

	return nil
}

func DeleteNamespaces(selector string, testNames ...string) error {
	command := []string{
		"delete", "namespace",
		// "--dry-run=client",
		"--cascade=foreground",
	}

	if selector != "" {
		command = append(command, "-l", ManagedNamespace)
	} else {
		command = append(command, testNames...)
	}

	out, err := Kubectl("", command...)
	if ErrNotFound(out) {
		return nil
	}

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
		resourceQuery := []string{"get", crd, "-o", "jsonpath='{.items[*].metadata.name}'"}

		resources, err := Kubectl(testName, resourceQuery...)

		switch {
		case err != nil:
			ui.Debug("skip operation", crd, err.Error())
		case string(resources) == "''":
			ui.Debug(crd, " resources are deleted.")
		default:
			patch := []string{"patch", crd, "--type", "json"}
			patch = append(patch, K8SRemoveFinalizer...)
			patch = append(patch, string(resources))

			ui.Debug("Use patch", env.Default.Kubectl(), strings.Join(patch, " "))

			if _, err := Kubectl(testName, patch...); err != nil {
				return errors.Wrapf(err, "cannot patch '%s' finalizers", crd)
			}
		}
	}

	return DeleteNamespaces("", testName)
}

/*
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
    requests.cpu: {{.inputs.parameters.CPU}}
    limits.cpu: {{.inputs.parameters.CPU}}
  {{- end}}
  {{- if .Inputs.Parameters.Memory}}
    requests.memory: {{.inputs.parameters.Memory}}
    limits.memory: {{.inputs.parameters.Memory}}
   {{- end}}
`),
	}

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

	command := []string{"apply", "--wait", "-f", f.Name()}

	_, err = Kubectl(testName, command...)
	return err
}

*/

/*
******************************************************************

					Helm Resources

******************************************************************
*/

func ListHelm(testName string) error {
	command := []string{"list"}

	command = setOutput(command)

	out, err := Helm(testName, command...)

	ui.Info(string(out))

	return err
}
