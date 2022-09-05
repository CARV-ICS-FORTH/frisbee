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

	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/commands/common"
	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewInstallDevelopmentCmd() *cobra.Command {
	var options common.FrisbeeInstallOptions
	var chartPath string
	var publicIP net.IP

	cmd := &cobra.Command{
		Use:   "development <FrisbeePath> <PublicIP>",
		Short: "Install Frisbee in development mode.",
		Long:  "Install all Frisbee components, except for the controller which will run externally.",
		Args: func(cmd *cobra.Command, args []string) error {
			ui.Logo()

			if len(args) < 2 {
				return errors.New("please pass project path and public ip as argument")
			}

			// Check Project Path
			chartPath = fmt.Sprintf("%s/%s", args[0], common.FrisbeeChartLocalPath)
			_, err := os.Stat(chartPath + "/Chart.yaml")
			ui.ExitOnError("Check Helm Chart", err)

			// Check Public IP
			publicIP = net.ParseIP(args[1])

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			command := []string{"upgrade", "--install", "--wait",
				"--namespace", options.Namespace, "--create-namespace",
				"--set", fmt.Sprintf("operator.enabled=%t", false),
				"--set", fmt.Sprintf("operator.advertisedHost=%s", publicIP),
				"--values", options.Values,
				options.Name, chartPath,
			}

			common.InstallFrisbeeOnK8s(cmd, command, &options)

			common.InstallPDFExporter(&options)
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			ui.NL()
			ui.Info("Frisbee runs in development mode. You must use: ",
				fmt.Sprintf("FRISBEE_NAMESPACE=%s make run", options.Namespace))
			ui.NL()

			ui.Success(" Happy Testing! 🚀")
		},
	}

	common.PopulateInstallFlags(cmd, &options)

	return cmd
}
