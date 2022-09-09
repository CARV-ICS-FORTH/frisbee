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

package commands

import (
	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/env"
	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubectl-frisbee",
		Short: "Frisbee entrypoint for kubectl plugin",
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()

			err := cmd.Help()
			ui.PrintOnError("Displaying help", err)
		},
	}

	// Add global flags
	env.Settings.AddFlags(cmd)

	ui.SetVerbose(env.Settings.Debug)

	// Add subcommands
	cmd.AddCommand(
		// Platform Installation
		NewInstallCmd(),
		NewUninstallCmd(),

		// Test Management
		NewSubmitCmd(),
		NewGetCmd(),
		NewDeleteCmd(),
		NewInspectCmd(),

		// Analysis Tools
		NewSaveCmd(),
		NewReportCmd(),
	)

	// Check permissions on critical files
	env.Settings.CheckKubePerms()

	return cmd
}
