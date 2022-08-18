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
	"fmt"
	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/spf13/cobra"
	"os"
)

var (
	verbose bool
)

func init() {
	// Installation
	RootCmd.AddCommand(NewInstallCmd())
	RootCmd.AddCommand(NewUninstallCmd())

	// New commands
	RootCmd.AddCommand(NewSubmitCmd())
	RootCmd.AddCommand(NewGetCmd())
	RootCmd.AddCommand(NewDeleteCmd())

	RootCmd.AddCommand(NewInspectCmd())
}

var RootCmd = &cobra.Command{
	Use:   "kubectl-frisbee",
	Short: "Frisbee entrypoint for kubectl plugin",

	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		ui.SetVerbose(verbose)
	},

	Run: func(cmd *cobra.Command, args []string) {
		ui.Logo()
		err := cmd.Usage()
		ui.PrintOnError("Displaying usage", err)
		cmd.DisableAutoGenTag = true
	},
}

func Execute() {
	ui.SetVerbose(true)
	/*
		cfg, err := config.Load()
		ui.WarnOnError("loading config", err)

		defaultNamespace := "testkube"
		if cfg.Namespace != "" {
			defaultNamespace = cfg.Namespace
		}

		apiURI := "http://localhost:8088"
		if cfg.APIURI != "" {
			apiURI = cfg.APIURI
		}

		if os.Getenv("TESTKUBE_API_URI") != "" {
			apiURI = os.Getenv("TESTKUBE_API_URI")
		}

		RootCmd.PersistentFlags().BoolVarP(&telemetryEnabled, "telemetry-enabled", "", cfg.TelemetryEnabled, "enable collection of anonumous telemetry data")
		RootCmd.PersistentFlags().StringVarP(&client, "client", "c", "proxy", "client used for connecting to Testkube API one of proxy|direct")
		RootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "", defaultNamespace, "Kubernetes namespace, default value read from config if set")
		RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "", false, "show additional debug messages")
		RootCmd.PersistentFlags().StringVarP(&apiURI, "api-uri", "a", apiURI, "api uri, default value read from config if set")
		RootCmd.PersistentFlags().BoolVarP(&oauthEnabled, "oauth-enabled", "", cfg.OAuth2Data.Enabled, "enable oauth")
	*/

	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "", false, "show additional debug messages")

	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
