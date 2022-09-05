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
	embed "github.com/carv-ics-forth/frisbee"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/kubeshop/testkube/pkg/process"
	"github.com/spf13/cobra"
)

const (
	FrisbeeRepo           = "https://carv-ics-forth.github.io/frisbee/charts"
	FrisbeeChartInRepo    = "frisbee/platform"
	FrisbeeChartLocalPath = "charts/platform" // relative to Frisbee root.

	InstallationDir = "/home/fnikol/.frisbee"
)

const (
	JetstackRepo = "https://charts.jetstack.io"
)

/*******************************************************************

			Install The Frisbee Platform

*******************************************************************/

type FrisbeeInstallOptions struct {
	Name, Namespace, Chart, Values string
	NoCertManager, NoPDFExporter   bool
}

func PopulateInstallFlags(cmd *cobra.Command, options *FrisbeeInstallOptions) {
	cmd.Flags().StringVar(&options.Chart, "chart", FrisbeeChartInRepo, "chart name")
	cmd.Flags().StringVar(&options.Name, "name", "frisbee", "installation name")
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "frisbee", "installation namespace")
	cmd.Flags().StringVarP(&options.Values, "values", "f", "", "path to Helm values file")
	cmd.Flags().BoolVar(&options.NoCertManager, "no-cert-manager", false, "don't install cert-manager")
	cmd.Flags().BoolVar(&options.NoPDFExporter, "no-pdf-exporter", false, "don't install pdf exporter")
}

func InstallFrisbeeOnK8s(cmd *cobra.Command, command []string, options *FrisbeeInstallOptions) {
	ui.Info("Helm installing frisbee framework...")

	// Install dependencies
	if !options.NoCertManager {
		err := installCertManager(cmd)
		ui.ExitOnError("Helm install cert-manager", err)
	}

	// Update Frisbee repo
	updateHelmFrisbeeRepo()

	if Verbose(cmd) {
		command = append(command, "--debug")

		_, err := process.LoggedExecuteInDir("", os.Stdout, Helm, command...)
		ui.ExitOnError("Helm install frisbee", err)
	} else {
		_, err := process.Execute(Helm, command...)
		ui.ExitOnError("Helm install frisbee", err)
	}
}

func updateHelmFrisbeeRepo() {
	_, err := process.Execute(Helm, "repo", "add", "frisbee", FrisbeeRepo)
	if err != nil && !strings.Contains(err.Error(), "Error: repository name (frisbee) already exists, please specify a different name") {
		ui.WarnOnError("adding frisbee repo", err)
	}

	_, err = process.Execute(Helm, "repo", "update")
	ui.ExitOnError("Updating helm repositories", err)
}

func installCertManager(cmd *cobra.Command) error {
	_, err := process.Execute(Kubectl, "get", "crds", "certificates.cert-manager.io")
	if err != nil && !strings.Contains(err.Error(), "Error from server (NotFound)") {
		return err
	}

	if err == nil {
		ui.Info("Found existing crd certificates.cert-manager.io. " +
			"Assume that jetstack cert manager is already installed. Skip its installation.")

		return nil
	}

	ui.Info("Helm installing jetstack cert manager.")

	// Update Helm Repo
	_, err = process.Execute(Helm, "repo", "add", "jetstack", JetstackRepo)
	if err != nil && !strings.Contains(err.Error(), "Error: repository name (jetstack) already exists") {
		return err
	}

	_, err = process.Execute(Helm, "repo", "update")
	ui.ExitOnError("Update repo", err)

	// Prepare installation command app
	command := []string{"upgrade", "--install", "--create-namespace",
		"cert-manager", "jetstack/cert-manager",
		"--namespace", "cert-manager",
		"--set", "installCRDs=true",
	}

	if Verbose(cmd) {
		command = append(command, "--debug")
	}

	out, err := process.Execute(Helm, command...)
	if err != nil {
		return err
	}

	ui.Info("Helm install jetstack output", string(out))

	return nil
}

/*
******************************************************************

	Install PDF-Exporter
	This is required for generating pdfs from Grafana.

******************************************************************
*/
const (
	puppeteer   = "puppeteer"
	pdfExporter = "hack/pdf-exporter/long-dashboards.js"
)

var PDFExporter = filepath.Join(InstallationDir, pdfExporter)

func InstallPDFExporter(options *FrisbeeInstallOptions) {
	if options.NoPDFExporter {
		return
	}

	/*
		Install NodeJS dependencies
	*/
	if err := os.Chdir(InstallationDir); err != nil {
		ui.Fail(errors.Wrap(err, "Cannot chdir to Frisbee cache"))
	}
	ui.Info("Installing PDFExporter at", InstallationDir)

	command := []string{
		NPM, "list", InstallationDir,
		"|", "grep", puppeteer, "||",
		NPM, "install", puppeteer, "--package-lock", "--prefix", InstallationDir,
	}

	_, err := process.Execute("sh", "-c", strings.Join(command, " "))
	ui.ExitOnError("Install Puppeteer", err)

	/*
		Copy the embedded pdf exporter in the underlying filesystem.
	*/
	err = embed.CopyLocallyIfNotExists(embed.Hack, InstallationDir)
	ui.ExitOnError("Install PDF Renderer", err)
}
