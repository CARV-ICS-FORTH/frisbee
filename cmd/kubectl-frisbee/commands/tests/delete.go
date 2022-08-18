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
	"fmt"
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
		Run: func(cmd *cobra.Command, args []string) {
			client := common.GetClient(cmd)

			if deleteAll {
				tests, err := client.DeleteTests("")
				ui.ExitOnError("delete all tests", err)
				ui.SuccessAndExit("Successfully deleted all tests", fmt.Sprint(tests))
			}

			if len(args) > 0 {
				name := args[0]
				err := client.DeleteTest(name)
				ui.ExitOnError("delete test "+name, err)
				ui.SuccessAndExit("Successfully deleted test", name)
			}

			if len(selectors) != 0 {
				selector := strings.Join(selectors, ",")
				tests, err := client.DeleteTests(selector)
				ui.ExitOnError("deleting tests by labels: "+selector, err)
				ui.SuccessAndExit("Successfully deleted tests by labels", selector, "tests", fmt.Sprint(tests))
			}

			ui.Failf("Pass Test name, --all flag to delete all or labels to delete by labels")
		},
	}

	cmd.Flags().BoolVar(&deleteAll, "all", false, "Delete all tests")
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")

	return cmd
}
