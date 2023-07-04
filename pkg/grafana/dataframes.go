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

package grafana

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/gosimple/slug"
	"github.com/imroc/req/v3"
	"github.com/pkg/errors"
)

func evaluateDashboardVariable(expr *string) {
	// https://prometheus.io/docs/prometheus/latest/querying/basics/#instant-vector-selectors
	*expr = os.Expand(*expr, func(s string) string {
		val, exists := DefaultVariableEvaluation[s]
		if exists {
			return val
		}

		return "$" + s
	})
}

// DownloadData returns data for the given panel.
func (c *Client) DownloadData(ctx context.Context, url *URL, destDir string) error {
	if c == nil {
		panic("empty client was given")
	}

	/*---------------------------------------------------*
	 * Select Dashboard and Timerange
	 *---------------------------------------------------*/
	board, _, err := c.Conn.GetDashboardByUID(ctx, *url.DashboardUID)
	if err != nil {
		return errors.Wrapf(err, "cannot retrieve dashboard %s", *url.DashboardUID)
	}

	dataRange := TimeRange{
		From: url.FromTS.UTC(),
		To:   url.ToTS.UTC(),
		Raw: &RawTimeRange{
			From: url.FromTS.UTC(),
			To:   url.ToTS.UTC(),
		},
	}

	/*---------------------------------------------------*
	 * Download Annotations
	 *---------------------------------------------------*/
	annotationsFilepath := filepath.Join(destDir, "annotations.json")

	if err := downloadAnnotations(c.logger, url, annotationsFilepath); err != nil {
		return errors.Wrapf(err, "failed to download annotations")
	}

	/*---------------------------------------------------*
	 * Download DataFrames
	 *---------------------------------------------------*/
	for _, panel := range board.Panels {
		var queries []interface{}

		// extract queries per panel type
		switch {
		case panel.GraphPanel != nil:
			for _, target := range panel.GraphPanel.Targets {
				queries = append(queries, target)
			}
		case panel.TablePanel != nil:
			for _, target := range panel.TablePanel.Targets {
				evaluateDashboardVariable(&target.Expr)

				queries = append(queries, target)
			}
		case panel.SinglestatPanel != nil:
			for _, target := range panel.SinglestatPanel.Targets {
				evaluateDashboardVariable(&target.Expr)

				queries = append(queries, target)
			}
		case panel.StatPanel != nil:
			for _, target := range panel.StatPanel.Targets {
				evaluateDashboardVariable(&target.Expr)

				queries = append(queries, target)
			}
		case panel.BarGaugePanel != nil:
			for _, target := range panel.BarGaugePanel.Targets {
				evaluateDashboardVariable(&target.Expr)

				queries = append(queries, target)
			}
		case panel.HeatmapPanel != nil:
			for _, target := range panel.HeatmapPanel.Targets {
				evaluateDashboardVariable(&target.Expr)

				queries = append(queries, target)
			}
		case panel.TimeseriesPanel != nil:
			for _, target := range panel.TimeseriesPanel.Targets {
				evaluateDashboardVariable(&target.Expr)

				queries = append(queries, target)
			}
		case panel.CustomPanel != nil:
			c.logger.Info("CustomPanel is not supported. Skip it", "panelTitle", panel.Title)

			continue
		case panel.TextPanel != nil:
			c.logger.Info("TextPanel is not supported. Skip it", "panelTitle", panel.Title)

			continue
		case panel.DashlistPanel != nil:
			c.logger.Info("DashlistPanel is not supported. Skip it", "panelTitle", panel.Title)

			continue
		case panel.PluginlistPanel != nil:
			c.logger.Info("PluginlistPanel is not supported. Skip it", "panelTitle", panel.Title)

			continue
		case panel.RowPanel != nil:
			c.logger.Info("RowPanel is not supported. Skip it", "panelTitle", panel.Title)

			continue
		case panel.AlertlistPanel != nil:
			c.logger.Info("AlertlistPanel is not supported. Skip it", "panelTitle", panel.Title)

			continue
		default:
			c.logger.V(5).Info("Unhandled panel type. skip it",
				"panelTitle", panel.Title,
			)

			continue
		}

		// submit queries
		if len(queries) > 0 {
			dataReq := &DataRequest{
				Queries: queries,
				Range:   dataRange,
				From:    fmt.Sprint(url.FromTS.UnixMilli()),
				To:      fmt.Sprint(url.ToTS.UnixMilli()),
			}

			dataFilepath := filepath.Join(destDir, slug.Make(panel.Title)+".json")

			if err := downloadDataFrame(c.logger, url, dataReq, dataFilepath); err != nil {
				return errors.Wrapf(err, "unable to download csv data")
			}
		}
	}

	return nil
}

func downloadAnnotations(logger logr.Logger, url *URL, dstFile string) error {
	/*---------------------------------------------------*
	 * Fetch annotations from Grafana in JSON
	 *---------------------------------------------------*/
	client := req.NewClient()

	resp, err := client.R().Get(url.AnnotationsQuery())
	if err != nil {
		return errors.Wrapf(err, "GET has failed")
	}

	if !resp.IsSuccessState() {
		return errors.Errorf("unsuccessful response: %s", resp)
	}

	/*---------------------------------------------------*
	 * Store annotations to file
	 *---------------------------------------------------*/
	if err := os.WriteFile(dstFile, resp.Bytes(), 0o600); err != nil {
		return errors.Wrapf(err, "failed to write annotations to '%s'", dstFile)
	}

	logger.Info("Annotations saved.", "file", dstFile)

	return nil
}

// downloadDataFrame downloads raw data without transformations and field config applied.
func downloadDataFrame(logger logr.Logger, url *URL, reqBody *DataRequest, dstFile string) error {
	/*---------------------------------------------------*
	 * Fetch data from Grafana in JSON format
	 *---------------------------------------------------*/
	client := req.NewClient()

	resp, err := client.R().
		SetBodyJsonMarshal(reqBody).
		Post(url.DataSourceQuery())
	if err != nil {
		return errors.Wrapf(err, "POST has failed")
	}

	if !resp.IsSuccessState() {
		return errors.Errorf("unsuccessful response: %s", resp)
	}

	/*---------------------------------------------------*
	 * Store JSON to file
	 *---------------------------------------------------*/
	if err := os.WriteFile(dstFile, resp.Bytes(), 0o600); err != nil {
		return errors.Wrapf(err, "failed to write data to '%s'", dstFile)
	}

	logger.Info("Data saved.", "file", dstFile)

	return nil
}
