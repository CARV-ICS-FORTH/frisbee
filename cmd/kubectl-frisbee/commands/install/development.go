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

package install

import (
	"fmt"
	"net"
	"os"

	"github.com/carv-ics-forth/frisbee/pkg/netutils"
	"github.com/carv-ics-forth/frisbee/pkg/process"

	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

const FrisbeeChartLocalPath = "charts/platform" // relative to Frisbee root.

func NewInstallDevelopmentCmdCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	switch {
	case len(args) == 0:
		return []string{"./"}, cobra.ShellCompDirectiveFilterDirs

	case len(args) == 1:
		return []string{netutils.GetOutboundIP().String()}, cobra.ShellCompDirectiveNoFileComp

	default:
		return common.CompleteFlags(cmd, args, toComplete)
	}
}

func NewInstallDevelopmentCmd() *cobra.Command {
	var (
		options          common.FrisbeeInstallOptions
		chartPath        string
		advertisedHostIP net.IP
		values           string
	)

	cmd := &cobra.Command{
		Use:               "development <FrisbeePath> <PublicIP>",
		Short:             "Install Frisbee in development mode.",
		Long:              "Install all Frisbee components, except for the controller which will run externally.",
		ValidArgsFunction: NewInstallDevelopmentCmdCompletion,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				ui.Failf("please pass project path and public ip as argument")
			}

			// Check Project Path
			chartPath = fmt.Sprintf("%s/charts/platform", args[0])
			_, err := os.Stat(chartPath + "/Chart.yaml")
			ui.ExitOnError("Check Helm Chart", err)

			// Check Public IP
			advertisedHostIP = net.ParseIP(args[1])

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			// use make generate to update the manifest of the project.
			err := os.Chdir(args[0])
			ui.ExitOnError("chdir to "+args[0], err)

			_, err = process.Execute("make", "generate")
			ui.ExitOnError("Update Manifest", err)

			command := []string{
				"upgrade", "--install", "--wait",
				"--create-namespace", "--namespace", common.FrisbeeNamespace,
				"--set", fmt.Sprintf("operator.enabled=%t", false),
				"--set", fmt.Sprintf("operator.advertisedHost=%s", advertisedHostIP),
			}

			if values != "" {
				command = append(command, "--values", values)
			}

			command = append(command, common.FrisbeeInstallation, chartPath)

			common.InstallFrisbeeOnK8s(command, &options)
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			ui.NL()
			ui.Success("Frisbee installed in development mode. Run it with: ",
				fmt.Sprintf("FRISBEE_ADVERTISED_HOST=%s FRISBEE_NAMESPACE=%s make run",
					advertisedHostIP,
					common.FrisbeeNamespace,
				),
			)
			ui.NL()

			ui.Success(" Happy Testing! 🚀")
		},
	}

	cmd.Flags().StringVarP(&values, "values", "f", FrisbeeChartLocalPath+"/values.yaml", "helm values file")

	common.PopulateInstallFlags(cmd, &options)

	// 	cmd.Flags().StringVarP(&options.Values, "values", "f", filepath.Join(options.Chart, "values.yaml"), "path to Helm values file")

	return cmd
}
