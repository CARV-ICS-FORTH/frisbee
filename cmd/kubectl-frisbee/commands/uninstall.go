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

package commands

import (
	"os"

	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/commands/common"
	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/env"
	"github.com/carv-ics-forth/frisbee/pkg/home"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

type PlatformUninstallOptions struct {
	Namespace, Name, RepositoryCache              string
	All, DeleteCRDS, DeleteCache, DeleteNamespace bool
}

func PopulatePlatformUninstallFlags(cmd *cobra.Command, options *PlatformUninstallOptions) {
	cmd.Flags().StringVar(&options.Namespace, "namespace", "frisbee", "namespace where to install")

	cmd.Flags().StringVar(&options.Name, "name", "frisbee", "installation name")

	cmd.Flags().BoolVar(&options.DeleteCache, "delete-cache", false, "delete frisbee cache")
	cmd.Flags().BoolVar(&options.DeleteCRDS, "delete-crds", false, "delete frisbee crds")
	cmd.Flags().BoolVar(&options.DeleteNamespace, "delete-namespace", false, "delete the installation namespace")
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
			env.Logo()
			ui.SetVerbose(env.Default.Debug)

			if !common.CRDsExist(common.Scenarios) {
				ui.Failf("Frisbee is not installed on the kubernetes cluster.")
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			/*---------------------------------------------------*
			 * Delete Tests
			 *---------------------------------------------------*/
			ui.Info("Deleting Tests. If it takes long time, make sure that Frisbee controller is still running.")
			{
				// Delete Manifests
				err := common.DeleteNamespaces(common.ManagedNamespace)
				ui.ExitOnError("Deleting Tests....", err)
				ui.Success("Tests", "Deleted")

				_, err = common.Helm(options.Namespace, "uninstall", "--wait", options.Name)
				ui.ExitOnError("Deleting Helm charts ....", common.HelmIgnoreNotFound(err))
				ui.Success("Charts", "Deleted")
			}

			/*---------------------------------------------------*
			 * Delete CRDs
			 *---------------------------------------------------*/
			if options.DeleteCRDS || options.All {
				out, err := common.Kubectl("", "delete", "crds", "--wait",
					common.Scenarios, common.Clusters, common.Services, common.Cascades, common.Chaos,
					common.Calls, common.VirtualObjects, common.Templates)

				if err != nil && !common.ErrNotFound(out) {
					ui.ExitOnError("Deleting CRDs ....", err)
				}

				ui.Success("CRDs", "Deleted")
			}

			/*---------------------------------------------------*
			 * Delete Namespace
			 *---------------------------------------------------*/
			if options.DeleteNamespace || options.All {
				out, err := common.Kubectl("", "delete", "namespace", options.Namespace)
				if !common.ErrNotFound(out) {
					ui.ExitOnError("Deleting namespace ....", err)
				}

				ui.Success("Namespace", "Deleted")
			}

			/*---------------------------------------------------*
			 * Delete Cache
			 *---------------------------------------------------*/
			if options.DeleteCache || options.All {
				err := os.RemoveAll(options.RepositoryCache)
				if err != nil && !os.IsNotExist(err) {
					ui.ExitOnError("Deleting Cache ....", err)
				}

				ui.Success("Cache", "Deleted")
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
