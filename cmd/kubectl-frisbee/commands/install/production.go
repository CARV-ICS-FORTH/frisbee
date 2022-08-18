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

func NewInstallProductionCmd() *cobra.Command {
	var options common.HelmUpgradeOrInstallFrisbeeOptions

	cmd := &cobra.Command{
		Use:   "production",
		Short: "Install Frisbee in production mode.",
		Long:  "Install all Frisbee components, including the controller.",
		Run: func(cmd *cobra.Command, args []string) {
			ui.Info("Helm installing frisbee framework")

			common.UpdateHelmRepo()

			command := []string{"upgrade", "--install", "--wait", "--create-namespace", "--namespace", options.Namespace}
			command = append(command, options.Name, options.Chart)

			common.HelmInstall(command, &options)
		},
	}

	common.PopulateUpgradeInstallFlags(cmd, &options)

	return cmd
}
