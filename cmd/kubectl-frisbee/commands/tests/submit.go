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

	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/commands/common"
	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/kubeshop/testkube/pkg/process"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/rand"
)

type TestSubmitOptions struct {
	CPUQuota, MemoryQuota string
	Wait, Log             bool
}

func PopulateTestSubmitFlags(cmd *cobra.Command, options *TestSubmitOptions) {
	cmd.Flags().StringVar(&options.CPUQuota, "cpu", "", "set quotas for the total CPUs (e.g, 0.5) that can be used by all Pods running in the test.")
	cmd.Flags().StringVar(&options.MemoryQuota, "memory", "", "set quotas for the total Memory (e.g, 100Mi) that can be used by all Pods running in the test.")
	cmd.Flags().BoolVarP(&options.Wait, "wait", "w", false, "wait for the scenario to complete")
	cmd.Flags().BoolVarP(&options.Log, "log", "l", false, "tail logs until completion")
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
				return errors.Errorf("Pass Test Name and Test File Path")
			}

			return nil
		},

		Run: func(cmd *cobra.Command, args []string) {
			client := common.GetClient(cmd)

			testName := args[0]
			testFile := args[1]

			// Generate test name, if needed
			if strings.HasSuffix(testName, "-") {
				testName = fmt.Sprintf("%s%d", testName, rand.Intn(1000))
				ui.Info("Generate test name: ", testName)
			}

			// Query Kubernetes API for conflicting tests
			{
				scenario, err := client.GetScenario(testName)
				if err != nil && !k8errors.IsNotFound(errors.Cause(err)) {
					ui.Failf("Can't query Kubernetes API for test with name '%s'", testName)
				}

				// Check for conflicting tests
				if scenario != nil {
					ui.Failf("test with name '%s' already exists", testName)
				}

				ui.ExitOnError("Check for conflicting tests", err)
			}

			// Validate the scenario
			{
				err := common.RunTest(testName, testFile, true)
				ui.ExitOnError("Validating testfile"+testFile, err)
			}

			// Ensure the namespace for hosting the scenario
			{
				err := common.CreateNamespace(testName, common.ManagedNamespace)
				ui.ExitOnError("Create managed namespace:"+testName, err)

				if options.CPUQuota != "" || options.MemoryQuota != "" {
					err := common.SetQuota(testName, options.CPUQuota, options.MemoryQuota)
					ui.ExitOnError("Setting namespace quotas", err)
				}

			}

			// Install Helm Dependencies, if any
			{
				helmCharts := args[2:]
				for _, chart := range helmCharts {
					command := []string{"upgrade", "--install",
						filepath.Base(chart), chart,
						"--namespace", testName,
						"--create-namespace",
					}

					_, err := process.Execute(common.Helm, command...)
					ui.ExitOnError("Installing Dependency: "+chart, err)
				}
			}

			// Submit Scenario
			{
				err := common.RunTest(testName, testFile, false)
				ui.ExitOnError("Starting test-case execution ", err)
			}

			// Control test output
			ControlOutput(testName, &options)
		},
	}

	PopulateTestSubmitFlags(cmd, &options)

	return cmd
}

func ControlOutput(testName string, options *TestSubmitOptions) {
	if options.Wait {
		err := common.WaitTestSuccess(testName)
		ui.ExitOnError("waiting test completion "+testName, err)

	} else if options.Log {
		err := common.GetPodLogs(testName, true, "all")
		ui.ExitOnError("getting logs", err)
	}
}
