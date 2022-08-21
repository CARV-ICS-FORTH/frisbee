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
	"os"
)

type InspectOptions struct {
	NoDashboards, NoStatus, Events, Logs, NoResources, Charts bool
	All, NoHint                                               bool
	Interactive                                               string
}

func PopulateInspectFlags(cmd *cobra.Command, options *InspectOptions) {
	cmd.Flags().BoolVar(&options.NoStatus, "no-status", false, "disable status from scenario")
	cmd.Flags().BoolVar(&options.NoDashboards, "no-dashboards", false, "disable information about dashboards")
	cmd.Flags().BoolVar(&options.NoResources, "no-resources", false, "disable listing resources")
	cmd.Flags().BoolVar(&options.Events, "events", false, "show events hinting what's happening")
	cmd.Flags().BoolVar(&options.Logs, "logs", false, "show logs output from executor pod")
	cmd.Flags().BoolVar(&options.Charts, "charts", false, "show installed templates from dependent Helm charts")

	cmd.Flags().BoolVar(&options.All, "all", false, "enable all no-* features ")
	cmd.Flags().StringVar(&options.Interactive, "interactive", "", "opens a shell to a running container")
	cmd.Flags().BoolVar(&options.NoHint, "no-hint", false, "disable hints")
}

func Hint(options *InspectOptions, msg string, sub ...string) {
	if !options.NoHint {
		ui.Success(msg, sub...)
	}
}

func NewInspectTestCmd() *cobra.Command {
	var options InspectOptions

	cmd := &cobra.Command{
		Use:     "test <testName> [--interactive podName [-- ShellArgs]]",
		Aliases: []string{"tests", "t"},
		Short:   "Get all available test information",
		Long:    "Gets test execution details, until it's in success/error state, blocks until gets complete state",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				ui.Failf("Please Pass Test name as argument")
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			client := common.GetClient(cmd)

			testName := args[0]

			// Interactive is exclusive
			if options.Interactive != "" {
				ui.NL()

				err := common.OpenShell(testName, options.Interactive, args[1:]...)
				ui.ExitOnError("Opening Shell", err)

				return
			}

			if !options.NoStatus || options.All {
				test, err := client.GetTest(testName)
				ui.ExitOnError("Getting Test Information", err)

				if test != nil {
					ui.NL()
					err = common.RenderList(cmd, test, os.Stdout)
					ui.ExitOnError("== Scenario Overview ==", err)

					ui.NL()
					err = common.RenderList(cmd, test.Status, os.Stdout)
					ui.ExitOnError("== Scenario Status ==", err)

					Hint(&options, "For more information use:", "kubectl describe scenario -n", testName)
				} else {
					ui.SuccessAndExit("no such test:", testName)
				}
			}

			if !options.NoResources || options.All {
				ui.NL()
				err := common.GetFrisbeeResources(cmd, testName)
				ui.ExitOnError("== Active Frisbee Resources ==", err)
				Hint(&options, "For more Frisbee Resource information use:",
					"kubectl describe <Kind> [Names...] -n", testName)

				ui.NL()
				err = common.GetK8sResources(cmd, testName)
				ui.ExitOnError("== Active K8s Resources ==", err)

				Hint(&options, "For more K8s Resource information use:",
					"kubectl describe <Kind> [Names...] -n", testName)
			}

			if !options.NoDashboards || options.All {
				ui.NL()

				err := common.Dashboards(cmd, testName)
				ui.ExitOnError("== Visualization Dashboards ==", err)
			}

			if options.Charts || options.All {
				ui.NL()
				err := common.GetTemplateResources(cmd, testName)
				ui.ExitOnError("== Frisbee Templates ==", err)
				Hint(&options, "For more Template info use:",
					"kubectl describe templates -n", testName, "[template...]")

				/*
					ui.NL()
					err = common.ListHelm(cmd, testName)
					ui.ExitOnError("== Helm Charts ==", err)
					ui.Success("For more Helm info use:", "helm list -a -n", testName)
				*/
			}

			if options.Events || options.All {
				ui.NL()
				err := common.Events(testName)
				ui.ExitOnError("== Events ==", err)

				Hint(&options, "For more events use:", "kubectl get events -n", testName)
			}

			if options.Logs || options.All {
				ui.NL()
				err := common.Logs(cmd, testName, false)
				ui.ExitOnError("== Logs ==", err)

				Hint(&options, "For more logs use:", "kubectl logs -n", testName, "pod/<podName>")
			}
		},
	}

	PopulateInspectFlags(cmd, &options)

	return cmd
}
