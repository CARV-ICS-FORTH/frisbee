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

func UninstallCmdCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return common.CompleteFlags(cmd, args, toComplete)
}

type UninstallOptions struct {
	RepositoryCache                                           string
	DeleteTests, DeleteOperator, DeleteCRDS, DeleteCache, All bool
}

func UninstallFlags(cmd *cobra.Command, options *UninstallOptions) {
	cmd.Flags().BoolVar(&options.DeleteTests, "tests", false, "delete frisbee tests")
	cmd.Flags().BoolVar(&options.DeleteOperator, "operator", false, "delete frisbee operator")
	cmd.Flags().BoolVar(&options.DeleteCRDS, "crds", false, "delete frisbee crds")
	cmd.Flags().BoolVar(&options.DeleteCache, "cache", false, "delete frisbee cache")

	cmd.Flags().BoolVar(&options.All, "all", false, "delete everything")

	cmd.Flags().StringVar(&options.RepositoryCache, "repository-cache", home.CachePath("repository"), "path to the file containing cached repository indexes")
}

func NewUninstallCmd() *cobra.Command {
	var options UninstallOptions

	cmd := &cobra.Command{
		Use:               "uninstall",
		Short:             "Uninstall Frisbee from current kubectl context",
		Aliases:           []string{"un", "purge"},
		ValidArgsFunction: UninstallCmdCompletion,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			env.Logo()
			ui.SetVerbose(env.Default.Debug)

			if !common.CRDsExist(common.Scenarios) {
				ui.Failf("Frisbee is not installed on the kubernetes cluster.")
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			/*---------------------------------------------------*
			 * Delete Frisbee Tests
			 *---------------------------------------------------*/
			if options.DeleteTests || options.All {
				ui.Info("Deleting Tests. If it takes long time, make sure that Frisbee controller is still running.")

				// Delete namespaces of tests
				err := common.DeleteNamespaces(common.ManagedNamespace)
				ui.ExitOnError("Deleting Tests....", err)
				ui.Success("Tests", "Deleted")
			}

			/*---------------------------------------------------*
			 * Delete Frisbee Operator
			 *---------------------------------------------------*/
			if options.DeleteOperator || options.All {
				// uninstall frisbee operator
				_, err := common.Helm(common.FrisbeeNamespace, "uninstall", "--wait", common.FrisbeeInstallation)
				ui.ExitOnError("Deleting Operator ....", common.HelmIgnoreNotFound(err))
				ui.Success("Operator", "Deleted")

				// delete frisbee namespace
				out, err := common.Kubectl(common.ClusterScope, "delete", "namespace", common.FrisbeeNamespace)
				if !common.ErrNotFound(out) {
					ui.ExitOnError("Deleting namespace ....", err)
				}

				ui.Success("Operator Namespace", "Deleted")
			}

			/*---------------------------------------------------*
			 * Delete Frisbee CRDs
			 *---------------------------------------------------*/
			if options.DeleteCRDS || options.All {
				out, err := common.Kubectl(common.ClusterScope, "delete", "crds", "--wait",
					common.Services, common.Clusters,
					common.Chaos, common.Cascades,
					common.VirtualObjects, common.Calls,
					common.Templates, common.Scenarios,
				)

				if err != nil && !common.ErrNotFound(out) {
					ui.ExitOnError("Deleting CRDs ....", err)
				}

				ui.Success("CRDs", "Deleted")
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
	}
	UninstallFlags(cmd, &options)

	return cmd
}
