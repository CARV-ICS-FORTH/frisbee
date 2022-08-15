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
	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/commands/common"
	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/commands/common/validator"
	"github.com/carv-ics-forth/frisbee/pkg/client"
	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
	"time"
)

func NewWatchTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "test <testName>",
		Aliases: []string{"w", "watch"},
		Short:   "Watch logs output from executor pod",
		Long:    `Gets test execution details, until it's in success/error state, blocks until gets complete state`,
		Args:    validator.TestName,
		Run: func(cmd *cobra.Command, args []string) {
			client := common.GetClient(cmd)

			testName := args[0]

			scenario, err := client.GetTest(testName)
			ui.ExitOnError("getting test "+testName, err)

			if scenario == nil {
				ui.ExitOnError("validate test", errors.Errorf("test '%s' is nil", testName))
			}

			if scenario.Status.Phase.Is(v1alpha1.PhaseSuccess, v1alpha1.PhaseFailed) {
				ui.Completed("scenario is already finished")
			} else {
				watchLogs(testName, client)
			}
		},
	}
}

func watchLogs(testName string, c client.Client) {
	ui.Info("Getting pod logs")

	logs, err := c.Logs(testName)
	ui.ExitOnError("getting logs from executor", err)

	for l := range logs {
		switch l.Type_ {
		case output.TypeError:
			ui.UseStderr()
			ui.Errf(l.Content)
			if l.Result != nil {
				ui.Errf("Error: %s", l.Result.ErrorMessage)
				ui.Debug("Output: %s", l.Result.Output)
			}
			uiShellGetExecution(testName)
			os.Exit(1)
			return
		case output.TypeResult:
			ui.Info("Execution completed", l.Result.Output)
		default:
			ui.LogLine(l.String())
		}
	}

	ui.NL()

	// TODO Websocket research + plug into Event bus (EventEmitter)
	// watch for success | error status - in case of connection error on logs watch need fix in 0.8
	for range time.Tick(time.Second) {

		scenario, err := c.GetTest(testName)
		ui.ExitOnError("getting test "+testName, err)

		if scenario == nil {
			ui.ExitOnError("validate test", errors.Errorf("test '%s' is nil", testName))
		}

		if scenario.Status.Phase.Is(v1alpha1.PhaseSuccess, v1alpha1.PhaseFailed) {
			fmt.Println()

			uiShellGetExecution(testName)

			return
		}
	}

	uiShellGetExecution(testName)
}

func uiShellGetExecution(id string) {
	ui.ShellCommand(
		"Use following command to get test execution details",
		"kubectl testkube get execution "+id,
	)

	ui.NL()
}
