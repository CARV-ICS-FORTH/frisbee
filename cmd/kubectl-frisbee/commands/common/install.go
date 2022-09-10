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
	"strings"

	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/env"
	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/spf13/cobra"
)

const (
	FrisbeeRepo = "https://carv-ics-forth.github.io/frisbee/charts"
)

const (
	JetstackRepo = "https://charts.jetstack.io"
)

/*******************************************************************

			Install The Frisbee Platform

*******************************************************************/

type FrisbeeInstallOptions struct {
	Name, Namespace string
	NoCertManager   bool
}

func PopulateInstallFlags(cmd *cobra.Command, options *FrisbeeInstallOptions) {
	cmd.Flags().StringVar(&options.Name, "name", "frisbee", "installation name")
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "frisbee", "installation namespace")

	cmd.Flags().BoolVar(&options.NoCertManager, "no-cert-manager", false, "don't install cert-manager")
}

func InstallFrisbeeOnK8s(command []string, options *FrisbeeInstallOptions) {
	// Install dependencies
	if !options.NoCertManager {
		err := installCertManager()
		ui.ExitOnError("Helm install cert-manager", err)
	}

	// Update Frisbee repo
	updateHelmFrisbeeRepo()

	ui.Info("Installing Frisbee platform...")

	if env.Settings.Debug {
		command = append(command, "--debug")

		_, err := LoggedHelm("", command...)
		ui.ExitOnError("Installing Helm Charts", err)
	} else {
		_, err := Helm("", command...)
		ui.ExitOnError("Installing Helm Charts", err)
	}
}

func updateHelmFrisbeeRepo() {
	_, err := Helm("", "repo", "add", "frisbee", FrisbeeRepo)
	if err != nil && !strings.Contains(err.Error(), "Error: repository name (frisbee) already exists, please specify a different name") {
		ui.WarnOnError("adding frisbee repo", err)
	}

	_, err = Helm("", "repo", "update")
	ui.ExitOnError("Updating helm repositories", err)
}

func CRDsExist(apiresource string) bool {
	out, err := Kubectl("", "get", "crds", apiresource)
	if ErrNotFound(out) {
		return false
	}

	ui.ExitOnError("cannot query kubernetes api for crds", err)

	return true
}

func installCertManager() error {
	ui.Info("Installing cert manager...")

	if CRDsExist("certificates.cert-manager.io") {
		ui.Success("Found existing crds for jetstack cert-manager. Skip installation",
			"certificates.cert-manager.io")

		return nil
	}

	ui.Info("Helm installing jetstack cert manager.")

	// Update Helm Repo
	_, err := Helm("", "repo", "add", "jetstack", JetstackRepo)
	if err != nil && !strings.Contains(err.Error(), "Error: repository name (jetstack) already exists") {
		return err
	}

	_, err = Helm("", "repo", "update")
	ui.ExitOnError("Update repo", err)

	// Prepare installation command app
	command := []string{"upgrade", "--install", "--create-namespace",
		"cert-manager", "jetstack/cert-manager",
		"--set", "installCRDs=true",
	}

	if env.Settings.Debug {
		command = append(command, "--debug")
	}

	out, err := Helm("cert-manager", command...)
	if err != nil {
		return err
	}

	ui.Info("Helm install jetstack output", string(out))

	return nil
}
