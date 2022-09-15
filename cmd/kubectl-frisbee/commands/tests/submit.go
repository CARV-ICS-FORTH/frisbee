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

package tests

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/commands/common"
	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/env"
	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TestSubmitOptions struct {
	CPUQuota, MemoryQuota                     string
	Watch                                     bool
	ExpectSuccess, ExpectFailure, ExpectError bool
	Timeout                                   string

	Logs     []string
	Loglines int
}

func PopulateTestSubmitFlags(cmd *cobra.Command, options *TestSubmitOptions) {
	// cmd.Flags().StringVar(&options.CPUQuota, "cpu", "", "set quotas for the total CPUs (e.g, 0.5) that can be used by all Pods running in the test.")
	// cmd.Flags().StringVar(&options.MemoryQuota, "memory", "", "set quotas for the total Memory (e.g, 100Mi) that can be used by all Pods running in the test.")
	cmd.Flags().StringSliceVarP(&options.Logs, "logs", "l", nil, "show logs output from executor pod (if unsure, use 'all')")
	cmd.Flags().IntVar(&options.Loglines, "log-lines", 5, "Lines of recent log file to display.")

	cmd.Flags().BoolVarP(&options.Watch, "watch", "w", false, "watch status")

	cmd.Flags().BoolVar(&options.ExpectSuccess, "expect-success", false, "wait for the scenario to complete successfully.")
	cmd.Flags().BoolVar(&options.ExpectFailure, "expect-failure", false, "wait for the scenario to fail ungracefully.")
	cmd.Flags().BoolVar(&options.ExpectError, "expect-error", false, "wait for the scenario to abort due to an assertion error.")
	cmd.Flags().StringVarP(&options.Timeout, "timeout", "t", "1m", "wait for the scenario to complete or to fail.")
}

func NewSubmitTestCmd() *cobra.Command {
	var options TestSubmitOptions

	cmd := &cobra.Command{
		Use:     "test <Name> <Scenario> <Dependencies...> ",
		Aliases: []string{"t"},
		Short:   "Submit a new test",
		Long:    `Submit starts new test based on Test Custom Resource name, returns results to console`,
		Example: `# Submit multiple workflows from files:
  kubectl frisbee submit test my-wf.yaml
# Submit and wait for completion:
  kubectl frisbee submit test --wait my-wf.yaml
# Submit and watch until completion:
  kubectl frisbee submit test --watch my-wf.yaml
# Submit and tail logs until completion:
  kubectl frisbee submit test --log my-wf.yaml
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				ui.Failf("Pass Test Name and Test File Path")
			}

			if options.ExpectSuccess && options.ExpectFailure && options.ExpectError {
				ui.Failf("Use one of --expect-success or --expect-failure or --expect-error.")
			}

			return nil
		},

		Run: func(cmd *cobra.Command, args []string) {
			testName, testFile := args[0], args[1]

			// Generate test name, if needed
			if strings.HasSuffix(testName, "-") {
				testName = fmt.Sprintf("%s%d", testName, rand.Intn(1000))
			}
			ui.Success("Submitting test ...", testName)

			// Validate the scenario
			{
				err := common.RunTest(testName, testFile, true)
				ui.ExitOnError("Validating testfile: "+testFile, err)
			}

			// Query Kubernetes API for conflicting tests
			{
				scenario, err := env.Settings.GetFrisbeeClient().GetScenario(cmd.Context(), testName)

				if scenario != nil {
					ui.Failf("test '%s' already exists", testName)
				}

				ui.ExitOnError("Looking for conflicts", client.IgnoreNotFound(err))
			}

			// Ensure the namespace for hosting the scenario
			{
				err := common.CreateNamespace(testName, common.ManagedNamespace)
				ui.ExitOnError("Creating managed namespace: "+testName, err)

				if options.CPUQuota != "" || options.MemoryQuota != "" {
					err := common.SetQuota(testName, options.CPUQuota, options.MemoryQuota)
					ui.ExitOnError("Setting namespace quotas", err)
				}
			}

			// Install Helm Dependencies, if any
			{
				helmCharts := args[2:]
				for _, chart := range helmCharts {
					command := []string{
						"upgrade", "--install",
						filepath.Base(chart), chart,
						"--create-namespace",
					}

					_, err := common.Helm(testName, command...)
					ui.ExitOnError("Installing Dependency: "+chart, err)
				}
			}

			// Submit Scenario
			{
				err := common.RunTest(testName, testFile, false)
				ui.ExitOnError("Starting test-case execution ", err)
			}

			ui.Success("Test has been successfully submitted.")

			// Control test output
			ControlOutput(testName, &options)
		},
	}

	PopulateTestSubmitFlags(cmd, &options)

	return cmd
}

func ControlOutput(testName string, options *TestSubmitOptions) {
	switch {
	case options.ExpectSuccess:
		ui.Info("Expecting the test to complete successfully within ", options.Timeout)

		err := common.WaitForCondition(testName, v1alpha1.ConditionAllJobsAreCompleted, options.Timeout)

		env.Settings.Hint("To inspect the execution:", "kubectl frisbee inspect test ", testName)
		ui.ExitOnError("waiting for test to complete successfully", err)

	case options.ExpectFailure:
		ui.Info("Expecting the test to fail within ", options.Timeout)

		err := common.WaitForCondition(testName, v1alpha1.ConditionJobUnexpectedTermination, options.Timeout)

		env.Settings.Hint("To inspect the execution:", "kubectl frisbee inspect test ", testName)
		ui.ExitOnError("waiting for test to fail", err)

	case options.ExpectError:
		ui.Info("Expecting the test to raise an assertion error within ", options.Timeout)

		err := common.WaitForCondition(testName, v1alpha1.ConditionAssertionError, options.Timeout)

		env.Settings.Hint("To inspect the execution:", "kubectl frisbee inspect test ", testName)
		ui.ExitOnError("waiting for test to raise an assertion error", err)

	case options.Watch:
		ui.Info("Watching for changes in the test status.")

		err := common.GetFrisbeeResources(testName, true)
		ui.ExitOnError("Watching for changes in the test status error", err)

	case options.Logs != nil:
		ui.Info("Tailing test logs ...", "log-lines", fmt.Sprint(options.Loglines))

		err := common.GetPodLogs(testName, true, options.Loglines, options.Logs...)
		env.Settings.Hint("To inspect the execution logs use:",
			"kubectl frisbee inspect test ", testName, " --logs all")
		ui.ExitOnError("Getting logs", err)
	}
}
