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

package install

import (
	"fmt"
	"net"
	"os"

	"github.com/kubeshop/testkube/pkg/process"

	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/commands/common"
	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/env"
	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/spf13/cobra"
)

const FrisbeeChartLocalPath = "charts/platform" // relative to Frisbee root.

func NewInstallDevelopmentCmd() *cobra.Command {
	var (
		options   common.FrisbeeInstallOptions
		chartPath string
		publicIP  net.IP
		values    string
	)

	cmd := &cobra.Command{
		Use:   "development <FrisbeePath> <PublicIP>",
		Short: "Install Frisbee in development mode.",
		Long:  "Install all Frisbee components, except for the controller which will run externally.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				ui.Failf("please pass project path and public ip as argument")
			}

			// Check Project Path
			chartPath = fmt.Sprintf("%s/charts/platform", args[0])
			_, err := os.Stat(chartPath + "/Chart.yaml")
			ui.ExitOnError("Check Helm Chart", err)

			// Check Public IP
			publicIP = net.ParseIP(args[1])

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
				"--namespace", options.Namespace, "--create-namespace",
				"--set", fmt.Sprintf("operator.enabled=%t", false),
				"--set", fmt.Sprintf("operator.advertisedHost=%s", publicIP),
				"--values", values,
				options.Name, chartPath,
			}

			common.InstallFrisbeeOnK8s(command, &options)
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			ui.NL()
			ui.Success("Frisbee installed in development mode. Run it with: ",
				fmt.Sprintf("KUBECONFIG=%s FRISBEE_NAMESPACE=%s make run",
					env.Settings.KubeConfig,
					options.Namespace))
			ui.NL()

			ui.Success(" Happy Testing! 🚀")
		},
	}

	cmd.Flags().StringVarP(&values, "values", "f", FrisbeeChartLocalPath+"/values.yaml", "helm values file")

	common.PopulateInstallFlags(cmd, &options)

	// 	cmd.Flags().StringVarP(&options.Values, "values", "f", filepath.Join(options.Chart, "values.yaml"), "path to Helm values file")

	return cmd
}
