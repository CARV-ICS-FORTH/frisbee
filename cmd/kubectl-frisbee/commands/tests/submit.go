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
	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/commands/common"
	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/spf13/cobra"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	"os"
)

func NewSubmitTestCmd() *cobra.Command {
	var (
		wait  bool
		watch bool
		log   bool
	)

	cmd := &cobra.Command{
		Use:     "test <NAME> <FILE>",
		Aliases: []string{"t"},
		Short:   "Submit a new test",
		Long:    `Submit starts new test based on Test Custom Resource name, returns results to console`,
		Example: `# Submit multiple workflows from files:
  kubectl-frisbee submit test my-wf.yaml
# Submit and wait for completion:
  kubectl-frisbee submit test --wait my-wf.yaml
# Submit and watch until completion:
  kubectl-frisbee submit test --watch my-wf.yaml
# Submit and tail logs until completion:
  kubectl-frisbee submit test --log my-wf.yaml
`,

		Run: func(cmd *cobra.Command, args []string) {
			client := common.GetClient(cmd)

			switch {
			case len(args) == 2:
				testName := args[0]
				testFile := args[1]

				// Query Kubernetes API for conflicting tests
				scenario, err := client.GetTest(testName)
				ui.ExitOnError("getting test "+testName, err)

				if err != nil && !k8errors.IsNotFound(err) {
					ui.UseStderr()
					ui.Errf("Can't query Kubernetes API for test with name '%s'", testName)
					ui.Debug(err.Error())
					os.Exit(1)
				}

				// Check for conflicting tests
				if scenario != nil {
					ui.UseStderr()
					ui.Errf("test with name '%s' already exists", testName)
					ui.Debug("Created", scenario.GetCreationTimestamp().String())
					ui.Debug("Status", scenario.GetReconcileStatus().Phase.String())
					os.Exit(1)
				}

				resources, err := client.SubmitTestFromFile(testName, testFile)
				for _, r := range resources {
					ui.Debug("Create", r)
				}

				ui.ExitOnError("starting test execution "+testName, err)

			default:
				ui.Failf("Pass Test Name and Test File Path")
			}
		},
	}

	cmd.Flags().BoolVarP(&wait, "wait", "w", false, "wait for the scenario to complete")
	cmd.Flags().BoolVar(&watch, "watch", false, "watch the scenario until it completes")
	cmd.Flags().BoolVar(&log, "log", false, "log the scenario until it completes")

	return cmd
}
