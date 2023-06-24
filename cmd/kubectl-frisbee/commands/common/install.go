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

package common

import (
	"github.com/pkg/errors"
	"log"
	"os"
	"path/filepath"
	"strings"

	embed "github.com/carv-ics-forth/frisbee"
	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/env"
	"github.com/carv-ics-forth/frisbee/pkg/process"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

/*******************************************************************

			Install The Frisbee Platform

*******************************************************************/

type FrisbeeInstallOptions struct {
	NoCertManager bool
}

func PopulateInstallFlags(cmd *cobra.Command, options *FrisbeeInstallOptions) {
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

	if env.Default.Debug {
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

	ui.ExitOnError("Query Kubernetes for CRDs", err)

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
	command := []string{
		"upgrade", "--install", "--create-namespace",
		"cert-manager", "jetstack/cert-manager",
		"--set", "installCRDs=true",
	}

	if env.Default.Debug {
		command = append(command, "--debug")
	}

	out, err := Helm("cert-manager", command...)
	if err != nil {
		return err
	}

	ui.Info("Helm install jetstack output", string(out))

	return nil
}

/*---------------------------------------------------*
 	Install PDF-Exporter.
	This is required for generating pdfs from Grafana.
 *---------------------------------------------------*/

const (
	puppeteer = "puppeteer@19.11.0"
)

type PDFExporter string

var (
	// DefaultPDFExport points to either FastPDFExporter or LongPDFExporter.
	DefaultPDFExport PDFExporter

	// FastPDFExporter is fast on individual panels, but does not render dashboard with many panels.
	FastPDFExporter PDFExporter

	// LongPDFExporter can render dashboards with many panels, but it's a bit slow.
	LongPDFExporter PDFExporter
)

func InstallPDFExporter(location string) {
	/*---------------------------------------------------*
	 * Ensure that the Cache Dir exists.
	 *---------------------------------------------------*/
	_, err := os.Open(location)
	if err != nil && !os.IsNotExist(err) {
		ui.Failf("failed to open cache directory " + location)
	}

	err = os.MkdirAll(location, os.ModePerm)
	ui.ExitOnError("create cache directory:"+location, err)

	/*---------------------------------------------------*
	 * Install NodeJS dependencies
	 *---------------------------------------------------*/
	ui.Info("Installing PDFExporter ...")

	oldPwd, _ := os.Getwd()

	err = os.Chdir(location)
	ui.ExitOnError("Installing PDFExporter ", err)

	command := []string{
		env.Default.NPM(), "list", location,
		"|", "grep", puppeteer, "||",
		env.Default.NPM(), "install", puppeteer, "--package-lock", "--prefix", location,
	}

	_, err = process.Execute("sh", "-c", strings.Join(command, " "))
	ui.ExitOnError(" --> Installing Puppeteer", err)

	ui.Success("PDFExporter is installed at ", location)

	err = os.Chdir(oldPwd)
	ui.ExitOnError("Returning to "+oldPwd, err)
}

func LoadPDFExporter(cacheLocation string) {
	/*---------------------------------------------------*
	 * Copy the embedded pdf exporter into fs
	 *---------------------------------------------------*/
	err := embed.UpdateLocalFiles(embed.Hack, cacheLocation)
	ui.ExitOnError(" --> Install PDF Renderer", err)

	/*---------------------------------------------------*
	 * Update path to the pdf-exporter binary
	 *---------------------------------------------------*/
	FastPDFExporter = PDFExporter(filepath.Join(cacheLocation, "hack/pdf-exporter/fast-generator.js"))
	LongPDFExporter = PDFExporter(filepath.Join(cacheLocation, "hack/pdf-exporter/long-dashboards.js"))

	if err := os.Setenv("PATH", os.Getenv("PATH")+":"+cacheLocation); err != nil {
		log.Fatal(err)
	}

	if err := os.Setenv("NODE_PATH", os.Getenv("NODE_PATH")+":"+cacheLocation); err != nil {
		log.Fatal(err)
	}

	// needed because the pdf-exporter lives in the installation cache.
	if err := os.Chdir(cacheLocation); err != nil {
		ui.Fail(errors.Wrap(err, "Cannot chdir to Frisbee cache"))
	}
}
