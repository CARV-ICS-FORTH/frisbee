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
	"fmt"
	"os"

	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/spf13/cobra"
)

var (
	verbose bool
)

func init() {
	// Platform Installation
	RootCmd.AddCommand(NewInstallCmd())
	RootCmd.AddCommand(NewUninstallCmd())

	// Test Management
	RootCmd.AddCommand(NewSubmitCmd())
	RootCmd.AddCommand(NewGetCmd())
	RootCmd.AddCommand(NewDeleteCmd())

	// Test Runtime
	RootCmd.AddCommand(NewInspectCmd())
	RootCmd.AddCommand(NewSaveCmd())
}

var RootCmd = &cobra.Command{
	Use:   "kubectl-frisbee",
	Short: "Frisbee entrypoint for kubectl plugin",
	Run: func(cmd *cobra.Command, args []string) {
		ui.Logo()
		err := cmd.Usage()
		ui.PrintOnError("Displaying usage", err)
		cmd.DisableAutoGenTag = true
	},
}

func Execute() {
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", true, "show additional debug messages")
	RootCmd.PersistentFlags().Bool("hints", true, "show hints related to the specific operations")

	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
