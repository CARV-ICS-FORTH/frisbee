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

package tests

import (
	"strings"

	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/commands/common"
	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/spf13/cobra"
)

type TestDeleteOptions struct {
	DeleteAll, Force bool
	Selectors        []string
}

func PopulateTestDeleteFlags(cmd *cobra.Command, options *TestDeleteOptions) {
	cmd.Flags().BoolVar(&options.DeleteAll, "all", false, "Delete all tests")
	cmd.Flags().StringSliceVarP(&options.Selectors, "label", "l", nil, "label key value pair: --label key1=value1")

	cmd.Flags().BoolVar(&options.Force, "force", false, "Force delete a stalled test")
}

func NewDeleteTestsCmd() *cobra.Command {
	var options TestDeleteOptions

	cmd := &cobra.Command{
		Use:     "test <testName>",
		Aliases: []string{"t", "tests"},
		Short:   "Delete Test",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !options.DeleteAll {
				ui.Failf("Pass Test name, --all flag to delete all or labels to delete by labels.")
			}

			if options.DeleteAll && options.Force {
				ui.Failf("Choose only one of --all or --force.")
			}

			if len(args) > 1 && options.Force {
				ui.Failf("To prevent intended deletions, --force is applicable at one test at a time.")
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {

			switch {
			case options.Force:
				testName := args[0]

				ui.Info("Deleting test: ", testName)
				err := common.ForceDelete(testName)
				ui.ExitOnError("Force Delete "+testName, err)

			case options.DeleteAll:
				ui.Info("Deleting all tests with label: ", common.ManagedNamespace)

				err := common.DeleteNamespaces(common.ManagedNamespace)
				ui.ExitOnError("Delete all tests", err)

			case len(args) > 0:
				ui.Info("Deleting tests: ", args...)

				err := common.DeleteNamespaces("", args...)
				ui.ExitOnError("Delete tests", err)

			case len(options.Selectors) != 0:
				options.Selectors = append(options.Selectors, common.ManagedNamespace)
				selector := strings.Join(options.Selectors, ",")

				ui.Info("Deleting all tests with labels: ", common.ManagedNamespace)

				err := common.DeleteNamespaces(selector)
				ui.ExitOnError("Deleting tests by labels: "+selector, err)
			default:
				cmd.Help()
			}
		},
	}

	PopulateTestDeleteFlags(cmd, &options)

	return cmd
}
