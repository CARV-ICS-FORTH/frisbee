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
	"os"
	"strings"
)

func NewGetTestsCmd() *cobra.Command {
	var (
		selectors []string
	)

	cmd := &cobra.Command{
		Use:     "test <testName>",
		Aliases: []string{"tests", "t"},
		Short:   "Get all available tests",
		Long:    `Getting all available tests from given namespace - if no namespace given "frisbee" namespace is used`,
		Run: func(cmd *cobra.Command, args []string) {
			client := common.GetClient(cmd)

			if len(args) == 0 { // All tests
				tests, err := client.ListTests(strings.Join(selectors, ","))
				ui.ExitOnError("Getting all tests ", err)

				err = common.RenderList(cmd, tests, os.Stdout)
				ui.PrintOnError("Rendering list", err)

				return
			}

			for _, testName := range args { // Specific test
				test, err := client.GetTest(testName)
				if test != nil {
					ui.NL()
					ui.Info("Test:", testName)

					ui.Table(test.Status, os.Stdout)
				}

				ui.ExitOnError("Getting test "+testName, err)
			}
		},
	}

	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")

	return cmd
}
