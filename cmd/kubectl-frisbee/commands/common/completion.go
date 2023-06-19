/*
Copyright 2023 ICS-FORTH.

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

package common

import (
	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/env"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NoArgs(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}

// CompleteScenarios list the available test-cases
func CompleteScenarios(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	list, err := env.Default.GetFrisbeeClient().ListScenarios(cmd.Context(), ManagedNamespace)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	return list.TestNames(), cobra.ShellCompDirectiveDefault
}

// CompleteServices list the available services. Assumes that args[0] is the namespace
func CompleteServices(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	list, err := env.Default.GetFrisbeeClient().ListServices(cmd.Context(), args[0])
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	return list.Names(), cobra.ShellCompDirectiveDefault
}

func CompleteFlags(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	var flags []string

	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		flags = append(flags, "--"+flag.Name)
	})

	return flags, cobra.ShellCompDirectiveNoFileComp
}
