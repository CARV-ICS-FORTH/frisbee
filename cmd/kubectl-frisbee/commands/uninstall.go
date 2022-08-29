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
	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/commands/common"
	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/kubeshop/testkube/pkg/process"
	"github.com/spf13/cobra"
)

type PlatformUninstallOptions struct {
	Namespace, Name string
	CRDS            bool
}

func PopulatePlatformUninstallFlags(cmd *cobra.Command, options *PlatformUninstallOptions) {
	cmd.Flags().StringVar(&options.Namespace, "namespace", "frisbee", "namespace where to install")

	cmd.Flags().StringVar(&options.Name, "name", "frisbee", "installation name")

	cmd.Flags().BoolVar(&options.CRDS, "crds", false, "delete frisbee crds")
}

func NewUninstallCmd() *cobra.Command {
	var options PlatformUninstallOptions

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Frisbee from current kubectl context",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			ui.SetVerbose(verbose)
		},
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()

			ui.Verbose = true

			err := common.DeleteNamespaces(common.ManagedNamespace)
			ui.ExitOnError("Deleting test-cases", err)

			_, err = process.Execute("helm", "uninstall", "--wait",
				"--namespace", options.Namespace, options.Name)
			ui.ExitOnError("Uninstalling  platform", err)

			if options.CRDS {
				_, err = process.Execute("kubectl", "delete", "crds", "--wait",
					common.Scenarios, common.Clusters, common.Services, common.Cascades, common.Chaos,
					common.Calls, common.VirtualObjects, common.Templates)
				ui.ExitOnError("Uninstalling crds", err)
			}
		},
	}

	PopulatePlatformUninstallFlags(cmd, &options)

	return cmd
}
