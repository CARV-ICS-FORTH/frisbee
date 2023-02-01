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
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	embed "github.com/carv-ics-forth/frisbee"
	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/commands/completion"
	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/env"
	"github.com/carv-ics-forth/frisbee/pkg/grafana"
	"github.com/carv-ics-forth/frisbee/pkg/home"
	"github.com/carv-ics-forth/frisbee/pkg/process"
	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/meta"
)

const (
	User = "'':''" // Not really needed since we have no authentication in Grafana.

	SummaryDashboardUID = "summary"
)

type TestReportOptions struct {
	Force, Aggregate bool
	DashboardUID     string
	RepositoryCache  string

	Data bool
}

func ReportTestCmdCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	switch {
	case len(args) == 0:
		return completion.CompleteScenarios(cmd, args, toComplete)

	default:
		return completion.CompleteFlags(cmd, args, toComplete)
	}
}

func PopulateReportTestFlags(cmd *cobra.Command, options *TestReportOptions) {
	cmd.Flags().BoolVar(&options.Force, "force", false, "Force reporting test data despite test phase.")

	cmd.Flags().StringVar(&options.DashboardUID, "dashboard", SummaryDashboardUID, "The dashboard to generate report from.")

	cmd.Flags().BoolVar(&options.Data, "data", false, "download grafana data as csv (experimental)")

	cmd.Flags().BoolVar(&options.Aggregate, "aggregate", false, "Generate a single PDF for the entire dashboard.")

	cmd.Flags().StringVar(&options.RepositoryCache, "repository-cache", home.CachePath("repository"), "path to the file containing cached repository indexes")
}

func NewReportTestsCmd() *cobra.Command {
	var options TestReportOptions

	cmd := &cobra.Command{
		Use:               "test <testName> <destination>",
		Aliases:           []string{"tests", "t"},
		Short:             "Generate PDFs for every dashboard in Grafana.",
		ValidArgsFunction: ReportTestCmdCompletion,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				ui.Failf("Pass Test name and destination to store the reports.")
			}

			return nil
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			ui.Logo()

			if env.Default.NodeJS() == "" || env.Default.NPM() == "" {
				ui.Fail(errors.Errorf("report is disabled. It requires NodeJS and NPM to be installed in your system"))
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			testName, destination := args[0], args[1]

			InstallPDFExporter(options.RepositoryCache)

			// needed because the pdf-exporter lives in the installation cache.
			if err := os.Chdir(options.RepositoryCache); err != nil {
				ui.Fail(errors.Wrap(err, "Cannot chdir to Frisbee cache"))
			}

			/*---------------------------------------------------*
			 * Inspect the Scenario for Grafana Endpoints.
			 *---------------------------------------------------*/
			scenario, err := env.Default.GetFrisbeeClient().GetScenario(cmd.Context(), testName)
			ui.ExitOnError("Getting test information", err)

			switch {
			case scenario == nil:
				ui.Failf("test '%s' was not found", testName)
			case scenario.Status.GrafanaEndpoint == "":
				ui.Failf("Telemetry is not enabled for this test. ")
			case !scenario.Status.Phase.Is(v1alpha1.PhaseSuccess, v1alpha1.PhaseFailed):
				// Abort getting data from a non-completed test, unless --force is used
				if !options.Force {
					ui.Failf("Unsafe operation. The test is not completed yet. Use --force")
				}
			}

			/*-- Filter time to the beginning/ending of the scenario. --*/
			fromTS, toTS := FindTimeline(scenario)

			/*---------------------------------------------------*
			 * Download data as CSV for the specific timeline
			 *---------------------------------------------------*/
			if options.Data {
				url := grafana.NewURL(scenario.Status.GrafanaEndpoint).
					WithDashboard(options.DashboardUID).
					WithFromTS(time.UnixMilli(fromTS)).
					WithToTS(time.UnixMilli(toTS))

				err = SaveData(url, destination)
				ui.ExitOnError("Saving metrics to: "+destination, err)
			}

			/*---------------------------------------------------*
			 * Generate PDFs for the specific timeline
			 *---------------------------------------------------*/
			if options.Aggregate {
				url := grafana.BuildURL(scenario.Status.GrafanaEndpoint, options.DashboardUID, fromTS, toTS, "")

				err = SavePDF(&options, url, destination)
				ui.ExitOnError("Saving aggregated report to: "+destination, err)
			} else {
				uri := grafana.BuildURL(scenario.Status.GrafanaEndpoint, options.DashboardUID, fromTS, toTS, "&kiosk")

				err = SavePDFs(&options, uri, destination, scenario.Status.GrafanaEndpoint, options.DashboardUID)
				ui.ExitOnError("Saving reports to: "+destination, err)
			}
		},
	}

	PopulateReportTestFlags(cmd, &options)

	return cmd
}

// SavePDF extracts the pdf from Grafana and stores it to the destinatio.
func SavePDF(options *TestReportOptions, dashboardURI string, destination string) error {
	/*
		Validate the URI. This is because if the URI is wrong, the
		nodejs will block forever.
	*/
	_, err := url.ParseRequestURI(dashboardURI)
	if err != nil {
		return err
	}

	exporter := FastPDFExporter
	if options.Aggregate {
		exporter = LongPDFExporter
	}

	command := []string{
		exporter,
		dashboardURI,
		User,
		destination,
	}

	ui.Info("Saving report to", destination)

	_, err = process.LoggedExecuteInDir("", os.Stdout, env.Default.NodeJS(), command...)

	return err
}

var (
	nonAlphanumericRegex  = regexp.MustCompile(`[^a-zA-Z0-9]+`)
	removeDuplicatesRegex = regexp.MustCompile(`/_{2,}/g`)
)

func SavePDFs(options *TestReportOptions, dashboardURI, destDir string, grafanaEndpoint string, dashboardUID string) error {
	/*---------------------------------------------------*
	 * Ensure destination exists
	 *---------------------------------------------------*/
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return errors.Wrapf(err, "destination error")
	}

	/*---------------------------------------------------*
	 * Query Grafna for Available Pannels.
	 *---------------------------------------------------*/
	ctx := context.Background()

	grafanaClient, err := grafana.New(ctx, grafana.WithHTTP(grafanaEndpoint))
	if err != nil {
		return errors.Wrapf(err, "cannot get Grafana client")
	}

	panels, err := grafanaClient.ListPanels(ctx, dashboardUID)
	if err != nil {
		return err
	}

	/*---------------------------------------------------*
	 * Generate PDF for each Panel.
	 *---------------------------------------------------*/
	for i, panel := range panels {
		panelURI := fmt.Sprintf("%s&viewPanel=%d", dashboardURI, panel.ID)

		/*-- replace special characters with underscore --*/
		title := nonAlphanumericRegex.ReplaceAllString(panel.Title, "_")
		title = removeDuplicatesRegex.ReplaceAllString(title, "_")

		ui.Debug(fmt.Sprintf("Processing %d/%d", i, len(panels)))

		if err := SavePDF(options, panelURI, filepath.Join(destDir, title)+".pdf"); err != nil {
			return errors.Wrapf(err, "cannot save panel '%d (%s)'", panel.ID, panel.Title)
		}
	}

	return nil
}

func SaveData(url *grafana.URL, dstDir string) error {
	/*---------------------------------------------------*
	 * Ensure destination exists
	 *---------------------------------------------------*/
	if err := os.MkdirAll(dstDir, os.ModePerm); err != nil {
		return errors.Wrapf(err, "destination error")
	}

	/*---------------------------------------------------*
	 * APIQuery Grafana for Available Panels.
	 *---------------------------------------------------*/
	ctx := context.Background()

	grafanaClient, err := grafana.New(ctx, grafana.WithHTTP(url.Endpoint))
	if err != nil {
		return errors.Wrapf(err, "cannot get Grafana client")
	}

	/*---------------------------------------------------*
	 * Download CSV data from each panel
	 *---------------------------------------------------*/
	if err := grafanaClient.DownloadData(ctx, url, dstDir); err != nil {
		return errors.Wrapf(err, "failed to get Grafana data")
	}

	return nil
}

/*---------------------------------------------------*
 	Install PDF-Exporter.
	This is required for generating pdfs from Grafana.
 *---------------------------------------------------*/

const (
	puppeteer = "puppeteer"
)

var (
	// FastPDFExporter is fast on individual panels, but does not render dashboard with many panels.
	FastPDFExporter string

	// LongPDFExporter can render dashboards with many panels, but it's a bit slow.
	LongPDFExporter string
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

	/*---------------------------------------------------*
	 * Copy the embedded pdf exporter into fs
	 *---------------------------------------------------*/
	err = embed.CopyLocallyIfNotExists(embed.Hack, location)
	ui.ExitOnError(" --> Install PDF Renderer", err)

	err = os.Chdir(oldPwd)
	ui.ExitOnError("Returning to "+oldPwd, err)

	/*---------------------------------------------------*
	 * Update path to the pdf-exporter binary
	 *---------------------------------------------------*/
	FastPDFExporter = filepath.Join(location, "hack/pdf-exporter/fast-generator.js")
	LongPDFExporter = filepath.Join(location, "hack/pdf-exporter/long-dashboards.js")

	os.Setenv("PATH", os.Getenv("PATH")+":"+location)
	os.Setenv("NODE_PATH", os.Getenv("NODE_PATH")+":"+location)

	ui.Success("PDFExporter is installed at ", location)
}

// FindTimeline parses the scenario to find timeline that make sense (formatted into time.UnixMilli).
/*---------------------------------------------------*
	For the starting time we adhere to these rules:
	 1. If possible, we use the time that the first job was scheduled.
	 2. Otherwise, we use the Creation time.

	For the ending time we adhere to these rules:
	 1. If the scenario is successful, we return the ConditionAllJobsAreCompleted time.
	 2. If the scenario has failed, we return the Failure time.
	 3. Otherwise, we report time.Now().
 *---------------------------------------------------*/
func FindTimeline(scenario *v1alpha1.Scenario) (from int64, to int64) {
	initialized := meta.FindStatusCondition(scenario.Status.Conditions, v1alpha1.ConditionCRInitialized.String())
	if initialized != nil {
		from = initialized.LastTransitionTime.Time.UnixMilli()
	} else {
		from = scenario.GetCreationTimestamp().Time.UnixMilli()
	}

	to = time.Now().UnixMilli()

	switch scenario.Status.Phase {
	case v1alpha1.PhaseSuccess:
		success := meta.FindStatusCondition(scenario.Status.Conditions, v1alpha1.ConditionAllJobsAreCompleted.String())
		to = success.LastTransitionTime.Time.UnixMilli()

	case v1alpha1.PhaseFailed:
		// Failure may come from various reasons. Unfortunately we have to go through all of them.
		unexpected := meta.FindStatusCondition(scenario.Status.Conditions, v1alpha1.ConditionJobUnexpectedTermination.String())
		if unexpected != nil {
			to = unexpected.LastTransitionTime.Time.UnixMilli()

			break
		}

		assert := meta.FindStatusCondition(scenario.Status.Conditions, v1alpha1.ConditionAssertionError.String())
		if assert != nil {
			to = assert.LastTransitionTime.Time.UnixMilli()

			break
		}
	}

	return from, to
}
