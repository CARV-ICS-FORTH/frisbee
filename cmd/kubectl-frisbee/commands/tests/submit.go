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

package tests

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/commands/common"
	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/env"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func SubmitTestCmdCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	switch {
	case len(args) == 0:
		return []string{"test-"}, cobra.ShellCompDirectiveDefault

	case len(args) == 1:
		return nil, cobra.ShellCompDirectiveDefault

	default:
		return nil, cobra.ShellCompDirectiveDefault
	}
}

type SubmitTestCmdOptions struct {
	CPUQuota, MemoryQuota                     string
	Watch                                     bool
	ExpectSuccess, ExpectFailure, ExpectError bool
	Timeout                                   string

	Logs []string
}

func SubmitTestCmdFlags(cmd *cobra.Command, options *SubmitTestCmdOptions) {
	// cmd.Flags().StringVar(&options.CPUQuota, "cpu", "", "set quotas for the total CPUs (e.g, 0.5) that can be used by all Pods running in the test.")
	// cmd.Flags().StringVar(&options.MemoryQuota, "memory", "", "set quotas for the total Memory (e.g, 100Mi) that can be used by all Pods running in the test.")
	cmd.Flags().StringSliceVarP(&options.Logs, "logs", "l", nil, "show logs output from executor pod (all|SUT|SYS|pod)")

	cmd.Flags().BoolVarP(&options.Watch, "watch", "w", false, "watch status")

	cmd.Flags().BoolVar(&options.ExpectSuccess, "expect-success", false, "wait for the scenario to complete successfully.")
	cmd.Flags().BoolVar(&options.ExpectFailure, "expect-failure", false, "wait for the scenario to fail ungracefully.")
	cmd.Flags().BoolVar(&options.ExpectError, "expect-error", false, "wait for the scenario to abort due to an assertion error.")
	cmd.Flags().StringVarP(&options.Timeout, "timeout", "t", "1m", "wait for the scenario to complete or to fail.")
}

func NewSubmitTestCmd() *cobra.Command {
	var options SubmitTestCmdOptions

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
		ValidArgsFunction: SubmitTestCmdCompletion,

		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				ui.Failf("Pass Test Name and Test File Path")
			}

			if strings.Contains(args[0], "/") {
				ui.Failf("Invalid format for test name: %s. \n%s", args[0],
					"Allowed formats are: 1) example (fixed name) and 2) example- (auto-generated)")
			}

			testFileExt := filepath.Ext(args[1])
			if testFileExt != ".yaml" && testFileExt != ".yml" {
				ui.Failf("Invalid format for test file: %s \n%s", args[1],
					"Allowed formats are: .yaml or .yml")
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

			/*---------------------------------------------------
			 * Client-side validation of the spec
			 *---------------------------------------------------*/
			// Lightweight validation of the scenario performed on the client side.
			// This allows us to filter-out some poorly written scenarios before interacting with the server.
			// More complex validation is performed on the server side (using admission webhooks) during
			// the actual submission.
			err := common.RunTest(testName, testFile, common.ValidationClient)
			ui.ExitOnError("Validating testfile: "+testFile, err)
			ui.Success("Scenario validated:", testFile)

			/*---------------------------------------------------
			 * Ensure environment isolation
			 *---------------------------------------------------*/
			// Query Kubernetes API for conflicting tests
			scenario, err := env.Default.GetFrisbeeClient().GetScenario(cmd.Context(), testName)
			ui.ExitOnError("Looking for conflicts", client.IgnoreNotFound(err))

			if scenario != nil {
				ui.Failf("test '%s' already exists", testName)
			}

			// ensure isolated namespace
			err = common.CreateNamespace(testName, common.ManagedNamespace)
			ui.ExitOnError("Creating managed namespace", err)

			/*
				if options.CPUQuota != "" || options.MemoryQuota != "" {
					err := common.SetQuota(testName, options.CPUQuota, options.MemoryQuota)
					ui.ExitOnError("Setting namespace quotas", err)
				}
			*/
			ui.Success("Namespace is ready:", testName)

			/*---------------------------------------------------
			 * Install Helm Dependencies, if any
			 *---------------------------------------------------*/
			{
				dependentCharts := args[2:]
				for _, dependency := range dependentCharts {
					_, err := common.Helm(testName,
						"upgrade", "--install",
						filepath.Base(dependency), dependency,
						"--create-namespace",
					)
					ui.ExitOnError("Installing Dependency: "+dependency, err)
				}

				ui.Success("Installed Dependencies:", dependentCharts...)
			}

			/*---------------------------------------------------
			 * Submit Scenario
			 *---------------------------------------------------*/
			err = common.RunTest(testName, testFile, common.ValidationNone)
			ui.ExitOnError("Starting test-case execution ", err)
			ui.Success("Scenario submitted.")

			// Control test output
			ControlOutput(cmd.Context(), testName, &options)
		},
	}

	SubmitTestCmdFlags(cmd, &options)

	return cmd
}

func ControlOutput(ctx context.Context, testName string, options *SubmitTestCmdOptions) {
	switch {
	case options.ExpectSuccess:
		ui.Info("Expecting the test to complete successfully within ", options.Timeout)

		err := common.WaitForCondition(ctx, testName, v1alpha1.ConditionAllJobsAreCompleted, options.Timeout)

		env.Default.Hint("To inspect the execution:", "kubectl frisbee inspect test ", testName)
		ui.ExitOnError("waiting for test to complete successfully", err)

	case options.ExpectFailure:
		ui.Info("Expecting the test to fail within ", options.Timeout)

		err := common.WaitForCondition(ctx, testName, v1alpha1.ConditionJobUnexpectedTermination, options.Timeout)

		env.Default.Hint("To inspect the execution:", "kubectl frisbee inspect test ", testName)
		ui.ExitOnError("waiting for test to fail", err)

	case options.ExpectError:
		ui.Info("Expecting the test to raise an assertion error within ", options.Timeout)

		err := common.WaitForCondition(ctx, testName, v1alpha1.ConditionAssertionError, options.Timeout)

		env.Default.Hint("To inspect the execution:", "kubectl frisbee inspect test ", testName)
		ui.ExitOnError("waiting for test to raise an assertion error", err)

	case options.Watch:
		ui.Info("Watching for changes in the test status.")

		err := common.GetFrisbeeResources(testName, true)
		ui.ExitOnError("Watching for changes in the test status error", err)

	case options.Logs != nil:
		ui.Warn("Streaming Logs from:", options.Logs...)

		err := common.KubectlLogs(ctx, testName, true, -1, options.Logs...)
		env.Default.Hint("To inspect the execution logs use:",
			"kubectl frisbee inspect test ", testName, " --logs all")

		ui.ExitOnError("Getting logs", err)
	}
}
