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
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	embed "github.com/carv-ics-forth/frisbee"
	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/commands/common"
	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/commands/completion"
	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/env"
	"github.com/carv-ics-forth/frisbee/pkg/grafana"
	"github.com/carv-ics-forth/frisbee/pkg/home"
	"github.com/carv-ics-forth/frisbee/pkg/process"
	"github.com/gosimple/slug"
	"github.com/hashicorp/go-multierror"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/meta"
)

const (
	User = "'':''" // Not really needed since we have no authentication in Grafana.

	Timeout = "24h"
)

var DefaultDashboards = []string{"summary", "singleton"}

func ReportTestCmdCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	switch {
	case len(args) == 0:
		return completion.CompleteScenarios(cmd, args, toComplete)

	case len(args) == 1:
		return nil, cobra.ShellCompDirectiveFilterDirs

	default:
		return completion.CompleteFlags(cmd, args, toComplete)
	}
}

type ReportTestCmdOptions struct {
	// RepositoryCache points to the location where external binaries (i.e, pdf generators) will be stored.
	RepositoryCache string

	// Dashboards select the Grafana dashboards that will be downloaded.
	Dashboards []string

	// PDF generates one PDf per each panel in the selected dashboard.
	PDF bool

	// AggregatePDF generates one PDF for all panels in the selected dashboard.
	AggregatedPDF bool

	// Data downloads data from Grafana
	Data bool

	// Force starts the reporting regardless of the status of the Scenario (data may be inconsistent).
	Force bool

	// Wait blocks until the Scenario is in terminal phase.
	Wait bool
}

func ReportTestCmdFlags(cmd *cobra.Command, options *ReportTestCmdOptions) {
	// RepositoryCache
	cmd.Flags().StringVar(&options.RepositoryCache, "repository-cache", home.CachePath("repository"), "path to the file containing cached repository indexes")

	// Dashboards
	cmd.Flags().StringSliceVar(&options.Dashboards, "dashboard", DefaultDashboards, "The dashboard(s) to generate report from.")

	if err := cmd.RegisterFlagCompletionFunc("dashboard", completion.CompleteServices); err != nil {
		log.Fatal(err)
	}

	// PDF
	cmd.Flags().BoolVar(&options.PDF, "pdf", false, "Generate one PDF for each panel in the dashboard.")

	// Aggregated PDF
	cmd.Flags().BoolVar(&options.AggregatedPDF, "aggregated-pdf", false, "Generate a single PDF for the entire dashboard.")

	// Data
	cmd.Flags().BoolVar(&options.Data, "data", false, "download grafana data as csv (experimental)")

	// Force
	cmd.Flags().BoolVar(&options.Force, "force", false, "Force reporting test data despite test phase.")

	// Wait
	cmd.Flags().BoolVar(&options.Wait, "wait", false, "Block waiting for scenario to be Success.")
}

func NewReportTestCmd() *cobra.Command {
	var options ReportTestCmdOptions

	cmd := &cobra.Command{
		Use:               "test <testName> <dstDir>",
		Aliases:           []string{"tests", "t"},
		Short:             "Generate PDFs for every dashboard in Grafana.",
		ValidArgsFunction: ReportTestCmdCompletion,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				ui.Failf("Pass Test name and destination to store the reports.")
			}

			if options.Wait && options.Force {
				ui.Failf("--wait and --force cannot be used together")
			}

			if !(options.PDF || options.Data || options.AggregatedPDF) {
				ui.Failf("at least one of [--pdf|--aggregated-pdf|--data] flags must be enabled")
			}

			return nil
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			env.Logo()

			if env.Default.NodeJS() == "" || env.Default.NPM() == "" {
				ui.Fail(errors.Errorf("report is disabled. It requires NodeJS and NPM to be installed in your system"))
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			testName, dstDir := args[0], args[1]

			/*---------------------------------------------------*
			 * Inspect the Scenario for Grafana Endpoints.
			 *---------------------------------------------------*/
			scenario, err := env.Default.GetFrisbeeClient().GetScenario(cmd.Context(), testName)
			ui.ExitOnError("Getting test information", err)

			// wait until either all jobs are finished or timeout expired
			if options.Wait {
				ui.Info("Waiting for scenario actions to be completed...")

				err = common.WaitForCondition(testName, v1alpha1.ConditionAllJobsAreCompleted, Timeout)
				ui.ExitOnError("scenario has not been be successful after "+Timeout+". : err", err)
			}

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

			/*-- Connect to Grafana --*/
			grafanaClient, err := grafana.New(cmd.Context(), grafana.WithHTTP(scenario.Status.GrafanaEndpoint))
			ui.ExitOnError("unable to connect to Grafana: err", err)

			/*---------------------------------------------------*
			 * Fix dependencies for PDF Generations
			 *---------------------------------------------------*/
			if options.PDF || options.AggregatedPDF {
				InstallPDFExporter(options.RepositoryCache)

				// needed because the pdf-exporter lives in the installation cache.
				if err := os.Chdir(options.RepositoryCache); err != nil {
					ui.Fail(errors.Wrap(err, "Cannot chdir to Frisbee cache"))
				}
			}

			/*---------------------------------------------------*
			 * Perform Reporting Activities
			 *---------------------------------------------------*/
			for _, dashboardUID := range options.Dashboards {
				// ensure dashboard directory exists
				dashboardDir := filepath.Join(dstDir, dashboardUID)

				err := os.MkdirAll(dashboardDir, os.ModePerm)
				ui.ExitOnError("Destination error: ", err)

				/*---------------------------------------------------*
				 * Save Data
				 *---------------------------------------------------*/
				if options.Data {
					grafanaEndpoint := grafana.NewURL(scenario.Status.GrafanaEndpoint).
						WithDashboard(dashboardUID).
						WithFromTS(time.UnixMilli(fromTS)).
						WithToTS(time.UnixMilli(toTS))

					err = SaveData(cmd.Context(), grafanaClient, grafanaEndpoint, dashboardDir)
					ui.ExitOnError("Saving Data to: "+dashboardDir+" for "+dashboardUID, err)
				}

				/*---------------------------------------------------*
				 * Generate PDFs
				 *---------------------------------------------------*/
				if options.PDF {
					DefaultPDFExport = FastPDFExporter

					grafanaEndpoint := grafana.BuildURL(scenario.Status.GrafanaEndpoint, dashboardUID, fromTS, toTS, "&kiosk")

					err = SavePDFs(cmd.Context(), grafanaClient, grafanaEndpoint, dashboardDir, dashboardUID)
					ui.ExitOnError("Saving PDF to: "+dashboardDir+" for "+dashboardUID, err)
				}

				/*---------------------------------------------------*
				 * Generate Aggregated PDF
				 *---------------------------------------------------*/
				if options.AggregatedPDF {
					DefaultPDFExport = LongPDFExporter

					uri := grafana.BuildURL(scenario.Status.GrafanaEndpoint, dashboardUID, fromTS, toTS, "")

					aggregatedFile := filepath.Join(dashboardDir, "__aggregated__.pdf")

					err = SavePDF(uri, aggregatedFile)
					ui.ExitOnError("Saving Aggregated PDF to: "+dashboardDir, err)
				}
			}
		},
	}

	ReportTestCmdFlags(cmd, &options)

	return cmd
}

// SavePDF extracts the pdf from Grafana and stores it to the destination.
func SavePDF(dashboardURI string, dstFile string) error {
	// 	Validate the URI. This is because if the URI is wrong, the
	// nodejs will block forever.
	if _, err := url.ParseRequestURI(dashboardURI); err != nil {
		return err
	}

	command := []string{
		string(DefaultPDFExport),
		dashboardURI,
		User,
		dstFile,
	}

	_, err := process.Execute(env.Default.NodeJS(), command...)
	if err != nil {
		return err
	}

	ui.Success("Saved pdf", dstFile)

	return err
}

func SavePDFs(ctx context.Context, grafanaClient *grafana.Client, dashboardURI, destDir, dashboardUID string) error {
	/*---------------------------------------------------*
	 * Query Grafana for Available Panels.
	 *---------------------------------------------------*/
	panels, err := grafanaClient.ListPanels(ctx, dashboardUID)
	if err != nil {
		return err
	}

	/*---------------------------------------------------*
	 * Generate PDF for each Panel.
	 *---------------------------------------------------*/
	var merr *multierror.Error

	for i, panel := range panels {
		ui.Debug(fmt.Sprintf("Processing %d/%d", i, len(panels)))

		panelURI := fmt.Sprintf("%s&viewPanel=%d", dashboardURI, panel.ID)
		file := filepath.Join(destDir, slug.Make(panel.Title)+".pdf")

		if err := SavePDF(panelURI, file); err != nil {
			merr = multierror.Append(merr,
				errors.Wrapf(err, "cannot save PDF for panel '%d (%s)'", panel.ID, panel.Title),
			)
		}
	}

	if merr.ErrorOrNil() != nil {
		ui.Warn("Errors", merr.Error())
	}

	return nil
}

func SaveData(ctx context.Context, grafanaClient *grafana.Client, url *grafana.URL, destDir string) error {
	/*---------------------------------------------------*
	 * Download CSV data from each panel
	 *---------------------------------------------------*/
	if err := grafanaClient.DownloadData(ctx, url, destDir); err != nil {
		return errors.Wrapf(err, "failed to download data from Grafana")
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
	FastPDFExporter = PDFExporter(filepath.Join(location, "hack/pdf-exporter/fast-generator.js"))
	LongPDFExporter = PDFExporter(filepath.Join(location, "hack/pdf-exporter/long-dashboards.js"))

	if err := os.Setenv("PATH", os.Getenv("PATH")+":"+location); err != nil {
		log.Fatal(err)
	}

	if err := os.Setenv("NODE_PATH", os.Getenv("NODE_PATH")+":"+location); err != nil {
		log.Fatal(err)
	}

	ui.Success("PDFExporter is installed at ", location)
}

// FindTimeline parses the scenario to find timeline that make sense (formatted into time.UnixMilli).
// ---------------------------------------------------
//	For the starting time we adhere to these rules:
//	 1. If possible, we use the time that the first job was scheduled.
//	 2. Otherwise, we use the Creation time.

//	For the ending time we adhere to these rules:
//	 1. If the scenario is successful, we return the ConditionAllJobsAreCompleted time.
//	 2. If the scenario has failed, we return the Failure time.
//	 3. Otherwise, we report time.Now().
//
// ---------------------------------------------------
func FindTimeline(scenario *v1alpha1.Scenario) (from int64, to int64) {
	// find "From"
	initialized := meta.FindStatusCondition(scenario.Status.Conditions, v1alpha1.ConditionCRInitialized.String())
	if initialized != nil {
		from = initialized.LastTransitionTime.Time.UnixMilli()
	} else {
		from = scenario.GetCreationTimestamp().Time.UnixMilli()
	}

	if scenario.Status.Phase == v1alpha1.PhaseSuccess {
		success := meta.FindStatusCondition(scenario.Status.Conditions, v1alpha1.ConditionAllJobsAreCompleted.String())

		return from, success.LastTransitionTime.Time.UnixMilli()
	}

	if scenario.Status.Phase == v1alpha1.PhaseFailed {
		// Failure may come from various reasons. Unfortunately we have to go through all of them.
		unexpected := meta.FindStatusCondition(scenario.Status.Conditions, v1alpha1.ConditionJobUnexpectedTermination.String())
		if unexpected != nil {
			return from, unexpected.LastTransitionTime.Time.UnixMilli()
		}

		assert := meta.FindStatusCondition(scenario.Status.Conditions, v1alpha1.ConditionAssertionError.String())
		if assert != nil {
			return from, assert.LastTransitionTime.Time.UnixMilli()
		}
	}

	// return a few second in the future to compensate for tardy events
	return from, time.Now().Add(GraceMonitoringPeriod).UnixMilli()
}

// GraceMonitoringPeriod is used to compensate for the misalignment between  the termination time of the container,
// and the next scraping of Prometheus. Normally, it should be twice the scrapping period (which by default is 15s).
const GraceMonitoringPeriod = 2 * 15 * time.Second
