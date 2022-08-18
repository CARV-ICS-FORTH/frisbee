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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
)

type InspectOptions struct {
	NoStatus, NoLogs, NoDashboards, NoEvents bool
}

func PopulateInspectFlags(cmd *cobra.Command, options *InspectOptions) {
	cmd.Flags().BoolVar(&options.NoStatus, "no-status", false, "disable status from scenario")
	cmd.Flags().BoolVar(&options.NoLogs, "no-logs", false, "disable logs output from executor pod")
	cmd.Flags().BoolVar(&options.NoDashboards, "no-dashboards", false, "disable information about dashboards")
	cmd.Flags().BoolVar(&options.NoEvents, "no-events", false, "disable events showing what's happening")
}

func NewInspectTestCmd() *cobra.Command {
	var options InspectOptions

	cmd := &cobra.Command{
		Use:     "test <testName>",
		Aliases: []string{"tests", "t"},
		Short:   "Get all available test information",
		Long:    "Gets test execution details, until it's in success/error state, blocks until gets complete state",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("please pass test name as argument")
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			client := common.GetClient(cmd)
			testName := args[0]

			if !options.NoDashboards {
				ui.NL()
				ui.Info("Dashboards:")

				err := common.Dashboards(testName)
				ui.ExitOnError("Getting Dashboards", err)
			}

			if !options.NoStatus {
				test, err := client.GetTest(testName)
				ui.ExitOnError("Getting Scenario Status", err)

				if test != nil {
					ui.NL()
					ui.Info("Test:")

					ui.Table(test.Status, os.Stdout)
				}
			}

			if !options.NoEvents {
				ui.NL()
				ui.Info("Events:")

				err := common.Events(testName)
				ui.ExitOnError("Getting Events", err)
			}

			if !options.NoLogs {
				ui.NL()
				ui.Info("Logs:")

				err := common.Logs(testName, false)
				ui.ExitOnError("Getting Logs", err)
			}
		},
	}

	PopulateInspectFlags(cmd, &options)

	return cmd
}
