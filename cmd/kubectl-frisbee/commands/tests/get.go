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

package tests

import (
	"os"

	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/commands/common"
	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/env"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewGetTestsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "test <testName>",
		Aliases:           []string{"tests", "t"},
		Short:             "Get all available tests",
		Long:              `Getting all available tests from given namespace - if no namespace given "frisbee" namespace is used`,
		ValidArgsFunction: common.NoArgs,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				ui.Failf("To get information for a test use: `kubectl frisbee inspect test <testName>`")
			}

			return nil
		},

		Run: func(cmd *cobra.Command, args []string) {
			tests, err := env.Default.GetFrisbeeClient().ListScenarios(cmd.Context(), common.ManagedNamespace)
			ui.PrintOnError("Getting all tests ", err)

			err = common.RenderList(&tests, os.Stdout)
			ui.PrintOnError("Rendering list", err)
		},
	}

	return cmd
}
