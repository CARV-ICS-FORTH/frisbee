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
	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/commands/common"
	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/spf13/cobra"
)

const FrisbeeChartInRepo = "frisbee/platform"

func NewInstallProductionCmd() *cobra.Command {
	var (
		options   common.FrisbeeInstallOptions
		chartPath string
		values    string
	)

	cmd := &cobra.Command{
		Use:   "production",
		Short: "Install Frisbee in production mode.",
		Long:  "Install all Frisbee components, including the controller.",
		Run: func(cmd *cobra.Command, args []string) {
			command := []string{
				"upgrade", "--install", "--wait",
				"--namespace", options.Namespace, "--create-namespace",
			}

			if values != "" {
				command = append(command, "--values", values)
			}

			command = append(command, options.Name, chartPath)

			common.InstallFrisbeeOnK8s(command, &options)
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			ui.NL()
			ui.Success(" Happy Testing! 🚀")
			ui.NL()
		},
	}

	cmd.Flags().StringVar(&chartPath, "chart", FrisbeeChartInRepo, "chart file to install")
	cmd.Flags().StringVarP(&values, "values", "f", "", "helm values file")
	common.PopulateInstallFlags(cmd, &options)

	return cmd
}
