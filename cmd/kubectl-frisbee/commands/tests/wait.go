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
	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/commands/common"
	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewWaitTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "test <testName>",
		Aliases: []string{"w", "watch"},
		Short:   "Wait test until it's in success/error state",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("please pass test name as argument")
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			client := common.GetClient(cmd)

			testName := args[0]

			scenario, err := client.GetTest(testName)
			ui.ExitOnError("getting test "+testName, err)

			if scenario == nil {
				ui.ExitOnError("validate test", errors.Errorf("test '%s' is nil", testName))
			}

			if scenario.Status.Phase.Is(v1alpha1.PhaseSuccess, v1alpha1.PhaseFailed) {
				ui.Completed("test is already finished")
			} else {
				// err := common.WaitTest(context.Background(), client, testName)
				// ui.ExitOnError("waiting test "+testName, err)
			}
		},
	}
}
