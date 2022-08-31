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
	"os"

	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/commands/common"
	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/spf13/cobra"
)

type InspectOptions struct {
	Overview, Events, ExternalResources, Charts bool
	All                                         bool
	Shell                                       string
	Logs                                        []string
}

func PopulateInspectFlags(cmd *cobra.Command, options *InspectOptions) {
	cmd.Flags().BoolVar(&options.Overview, "overview", true, "show test overview")
	cmd.Flags().BoolVar(&options.ExternalResources, "all-resources", false, "list Chaos and K8s resources")
	cmd.Flags().BoolVar(&options.Events, "events", false, "show events hinting what's happening")
	cmd.Flags().BoolVar(&options.Charts, "charts", false, "show installed templates from dependent Helm charts")

	cmd.Flags().BoolVar(&options.All, "all", false, "enable all no-* features ")
	cmd.Flags().StringVar(&options.Shell, "shell", "", "opens a shell to a running container")

	cmd.Flags().StringSliceVar(&options.Logs, "logs", nil, "show logs output from executor pod")
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
			ui.Logo()

			client := common.GetClient(cmd)

			testName := args[0]

			// Interactive is exclusive
			if options.Shell != "" {
				ui.NL()

				err := common.OpenShell(testName, options.Shell, args[1:]...)
				ui.ExitOnError("Opening Shell", err)

				return
			}

			// Always-on functions

			if options.Overview || options.All {
				test, err := client.GetScenario(testName)
				ui.ExitOnError("Getting Test Information", err)

				if test != nil {
					ui.NL()
					err = common.RenderList(cmd, test, os.Stdout)
					ui.ExitOnError("== Scenario Overview ==", err)

					ui.NL()
					err = common.RenderList(cmd, test.Status, os.Stdout)
					ui.ExitOnError("== Scenario Status ==", err)
				}

				ui.NL()
				err = common.GetFrisbeeResources(cmd, testName)
				ui.ExitOnError("== Active Frisbee Resources ==", err)

				ui.NL()
				err = common.Dashboards(cmd, testName)

				common.Hint(cmd, "For more Frisbee Resource information use:",
					"kubectl describe <Kind>.frisbee.dev [Names...] -n", testName)
				ui.ExitOnError("== Visualization Dashboards ==", err)
			}

			if options.ExternalResources || options.All {
				ui.NL()
				err := common.GetChaosResources(cmd, testName)

				common.Hint(cmd, "For more Chaos Resource information use:",
					"kubectl describe <Kind>.chaos-mesh.org [Names...] -n", testName)
				ui.ExitOnError("== Active Chaos Resources ==", err)

				ui.NL()
				err = common.GetK8sResources(cmd, testName)

				common.Hint(cmd, "For more K8s Resource information use:",
					"kubectl describe <Kind> [Names...] -n", testName)
				ui.ExitOnError("== Active K8s Resources ==", err)
			}

			if options.Charts || options.All {
				ui.NL()
				err := common.GetTemplateResources(cmd, testName)

				common.Hint(cmd, "For more Template info use:",
					"kubectl describe templates -n", testName, "[template...]")
				ui.ExitOnError("== Frisbee Templates ==", err)

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

				common.Hint(cmd, "For more events use:", "kubectl get events -n", testName)
				ui.ExitOnError("== Events ==", err)
			}

			if options.Logs != nil || options.All {
				ui.NL()
				vobjects, err := client.ListVirtualObjects(testName)

				for _, vobject := range vobjects.Items {
					if err := common.RenderList(cmd, vobject, os.Stdout); err != nil {
						ui.ExitOnError(vobject.GetName(), err)
					}
				}

				ui.ExitOnError("== Logs From Virtual Objects ==", err)

				ui.NL()
				err = common.GetPodLogs(testName, false, options.Logs...)

				common.Hint(cmd, "For more logs use:", "kubectl logs -n", testName, "<podNames>")
				ui.ExitOnError("== Logs From Pods ==", err)
			}
		},
	}

	PopulateInspectFlags(cmd, &options)

	return cmd
}
