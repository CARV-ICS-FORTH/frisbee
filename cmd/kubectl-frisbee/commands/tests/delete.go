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
	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/commands/common"
	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/spf13/cobra"
	"strings"
)

func NewDeleteTestsCmd() *cobra.Command {
	var deleteAll bool
	var selectors []string

	cmd := &cobra.Command{
		Use:     "test <testName>",
		Aliases: []string{"t", "tests"},
		Short:   "Delete Test",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !deleteAll {
				ui.Failf("Pass Test name, --all flag to delete all or labels to delete by labels")
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			if deleteAll {
				err := common.DeleteTests(common.ManagedNamespace, nil)
				ui.ExitOnError("Delete all tests", err)

				return
			}

			if len(args) > 0 {
				err := common.DeleteTests("", args)

				ui.ExitOnError("Delete tests", err)

				return
			}

			if len(selectors) != 0 {
				selectors = append(selectors, common.ManagedNamespace)
				selector := strings.Join(selectors, ",")

				err := common.DeleteTests(selector, nil)
				ui.ExitOnError("Deleting tests by labels: "+selector, err)

				return
			}
		},
	}

	cmd.Flags().BoolVar(&deleteAll, "all", false, "Delete all tests")
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")

	return cmd
}
