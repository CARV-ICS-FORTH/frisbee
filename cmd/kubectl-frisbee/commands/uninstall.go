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
	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/kubeshop/testkube/pkg/process"
	"github.com/spf13/cobra"
)

func NewUninstallCmd() *cobra.Command {
	var name, namespace string

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Frisbee from current kubectl context",
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()

			ui.Verbose = true

			_, err := process.Execute("helm", "uninstall", "--namespace", namespace, name)
			ui.PrintOnError("uninstalling frisbee", err)
		},
	}

	cmd.Flags().StringVar(&name, "name", "frisbee", "installation name")
	cmd.Flags().StringVar(&namespace, "namespace", "frisbee", "namespace where to install")

	return cmd
}
