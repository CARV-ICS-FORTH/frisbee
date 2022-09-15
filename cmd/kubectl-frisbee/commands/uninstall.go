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
	"os"

	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/commands/common"
	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/env"
	"github.com/carv-ics-forth/frisbee/pkg/home"
	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/spf13/cobra"
)

type PlatformUninstallOptions struct {
	Namespace, Name, RepositoryCache string
	All, CRDS, Cache                 bool
}

func PopulatePlatformUninstallFlags(cmd *cobra.Command, options *PlatformUninstallOptions) {
	cmd.Flags().StringVar(&options.Namespace, "namespace", "frisbee", "namespace where to install")

	cmd.Flags().StringVar(&options.Name, "name", "frisbee", "installation name")

	cmd.Flags().BoolVar(&options.Cache, "cache", false, "delete frisbee cache")
	cmd.Flags().BoolVar(&options.CRDS, "crds", false, "delete frisbee crds")
	cmd.Flags().BoolVar(&options.All, "all", false, "delete everything")

	cmd.Flags().StringVar(&options.RepositoryCache, "repository-cache", home.CachePath("repository"), "path to the file containing cached repository indexes")
}

func NewUninstallCmd() *cobra.Command {
	var options PlatformUninstallOptions

	cmd := &cobra.Command{
		Use:     "uninstall",
		Short:   "Uninstall Frisbee from current kubectl context",
		Aliases: []string{"un", "purge"},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			ui.Logo()

			env.Settings.CheckKubePerms()

			ui.Info("Using config:", env.Settings.KubeConfig)
		},
		Run: func(cmd *cobra.Command, args []string) {
			// Delete Tests
			if common.CRDsExist(common.Scenarios) {
				ui.Info("Deleting Tests. If it takes long time, make sure that Frisbee controller is still running.")

				err := common.DeleteNamespaces(common.ManagedNamespace)
				ui.ExitOnError("Deleting Tests....", err)

				ui.Success("Tests deleted")
			}

			// Delete Helm Charts
			{
				command := []string{
					"uninstall", "--wait",
					options.Name,
				}

				if env.Settings.Debug {
					command = append(command, "--debug")
				}

				_, err := common.Helm(options.Namespace, command...)
				ui.ExitOnError("Deleting Helm charts ....", common.HelmIgnoreNotFound(err))

				ui.Success("Charts deleted")
			}

			// Delete crds
			if options.CRDS || options.All {
				out, err := common.Kubectl("", "delete", "crds", "--wait",
					common.Scenarios, common.Clusters, common.Services, common.Cascades, common.Chaos,
					common.Calls, common.VirtualObjects, common.Templates)

				if err != nil && !common.ErrNotFound(out) {
					ui.ExitOnError("Deleting CRDs ....", err)
				}

				ui.Success("CRDS  deleted")

			}

			// Delete cache
			if options.Cache || options.All {
				err := os.RemoveAll(options.RepositoryCache)
				if err != nil && !os.IsNotExist(err) {
					ui.ExitOnError("Deleting Cache ....", err)
				}

				ui.Success("Cache  deleted")
			}
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			ui.NL()
			ui.Success("Frisbee has been uninstalled")
			ui.NL()
		},
	}
	PopulatePlatformUninstallFlags(cmd, &options)

	return cmd
}
