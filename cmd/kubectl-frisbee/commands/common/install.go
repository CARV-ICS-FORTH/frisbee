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

package common

import (
	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/kubeshop/testkube/pkg/process"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

const (
	FrisbeeRepo  = "https://carv-ics-forth.github.io/frisbee/charts"
	JetstackRepo = "https://charts.jetstack.io"
)

type HelmUpgradeOrInstallFrisbeeOptions struct {
	Name, Namespace, Chart, Values string
	NoJetstack                     bool
	Verbose                        bool
}

func PopulateUpgradeInstallFlags(cmd *cobra.Command, options *HelmUpgradeOrInstallFrisbeeOptions) {
	cmd.Flags().StringVar(&options.Chart, "chart", "frisbee/platform", "chart name")
	cmd.Flags().StringVar(&options.Name, "name", "frisbee", "installation name")
	cmd.Flags().StringVar(&options.Namespace, "namespace", "frisbee", "namespace where to install")
	cmd.Flags().StringVar(&options.Values, "values", "", "path to Helm values file")
	cmd.Flags().BoolVar(&options.NoJetstack, "no-jetstack", false, "don't install Jetstack")
}

func UpdateHelmRepo() {
	_, err := process.Execute(Helm, "repo", "add", "frisbee", FrisbeeRepo)
	if err != nil && !strings.Contains(err.Error(), "Error: repository name (frisbee) already exists, please specify a different name") {
		ui.WarnOnError("adding frisbee repo", err)
	}

	_, err = process.Execute(Helm, "repo", "update")
	ui.ExitOnError("Updating helm repositories", err)
}

func HelmInstall(command []string, options *HelmUpgradeOrInstallFrisbeeOptions) {
	// Install dependencies
	if !options.NoJetstack {
		err := installCertManager(Helm, options)
		ui.ExitOnError("Helm install cert-manager", err)
	}

	// Install Frisbee
	if options.Values != "" {
		command = append(command, "--values", options.Values)
	}

	if options.Verbose {
		command = append(command, "--debug")

		ui.Info(Helm, command...)

		out, err := process.LoggedExecuteInDir("", os.Stdout, Helm, command...)
		ui.ExitOnError("Helm install frisbee", err)

		ui.Info("Helm install frisbee output", string(out))
	} else {
		ui.Info(Helm, command...)

		out, err := process.Execute(Helm, command...)
		ui.ExitOnError("Helm install frisbee", err)

		ui.Info("Helm install frisbee output", string(out))
	}
}

func installCertManager(helmPath string, options *HelmUpgradeOrInstallFrisbeeOptions) error {
	_, err := process.Execute(Kubectl, "get", "crds", "certificates.cert-manager.io")
	if err != nil && !strings.Contains(err.Error(), "Error from server (NotFound)") {
		return err
	}

	if err != nil {
		ui.Info("Helm installing jetstack cert manager.")
		_, err = process.Execute(helmPath, "repo", "add", "jetstack", JetstackRepo)
		if err != nil && !strings.Contains(err.Error(), "Error: repository name (jetstack) already exists") {
			return err
		}

		_, err = process.Execute(helmPath, "repo", "update")
		if err != nil {
			return err
		}

		command := []string{"upgrade", "--install",
			"cert-manager", "jetstack/cert-manager",
			"--namespace", "cert-manager",
			"--create-namespace",
			"--set", "installCRDs=true",
		}

		if options.Verbose {
			command = append(command, "--debug")

			ui.Info(helmPath, command...)

			out, err := process.LoggedExecuteInDir("", os.Stdout, helmPath, command...)
			if err != nil {
				return err
			}

			ui.Info("Helm install jetstack output", string(out))
		} else {
			out, err := process.Execute(helmPath, command...)
			if err != nil {
				return err
			}

			ui.Info("Helm install jetstack output", string(out))
		}

	} else {
		ui.Info("Found existing crd certificates.cert-manager.io. Assume that jetstack cert manager is already installed. " +
			"Skip its installation")
	}

	return nil
}
