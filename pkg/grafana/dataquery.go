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
	"k8s.io/apimachinery/pkg/util/json"
)

type DataQuery struct {
	Datasource interface{} `json:"datasource"`
	Expr       string      `json:"expr"`
}

type DataRequest struct {
	Queries []DataQuery `json:"queries"`

	Range TimeRange `json:"range"`
	From  string    `json:"from"`
	To    string    `json:"to"`
}

// DownloadData returns data for the given panel.
func (c *Client) DownloadData(ctx context.Context, url *URL, destDir string) error {
	if c == nil {
		panic("empty client was given")
	}

	// select the dashboard.
	board, _, err := c.Conn.GetDashboardByUID(ctx, *url.DashboardUID)
	if err != nil {
		return errors.Wrapf(err, "cannot retrieve dashboard %s", *url.DashboardUID)
	}

	// set the time-range we are interested in.
	dataRange := TimeRange{
		From: url.FromTS.UTC(),
		To:   url.ToTS.UTC(),
		Raw: &RawTimeRange{
			From: url.FromTS.UTC(),
			To:   url.ToTS.UTC(),
		},
	}

	// iterate the panels of the dashboard.
	for _, panel := range board.Panels {
		var queries []DataQuery

		// iterate the metrics of the panel.
		switch {
		case panel.TimeseriesPanel != nil:
			for _, target := range panel.TimeseriesPanel.Targets {
				queries = append(queries, DataQuery{
					Datasource: target.Datasource,
					Expr:       target.Expr,
				})
			}
		default:
			c.logger.V(5).Info("Skip panel. Data can be extracted only from Timeseries",
				"panelTitle", panel.Title,
			)
		}

		// submit the query
		if len(queries) > 0 {
			req := &DataRequest{
				Queries: queries,
				Range:   dataRange,
				From:    fmt.Sprint(url.FromTS.UnixMilli()),
				To:      fmt.Sprint(url.ToTS.UnixMilli()),
			}

			file := filepath.Join(destDir, slug.Make(panel.Title)+".json")

			if err := downloadData(c.logger, url, req, file); err != nil {
				return errors.Wrapf(err, "unable to download csv data")
			}
		}
	}

	return nil
}

func downloadData(logger logr.Logger, url *URL, reqBody *DataRequest, dstFile string) error {
	reqBodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return errors.Wrapf(err, "failed to create request")
	}

	/*---------------------------------------------------*
	 * Fetch data from Grafana in JSON format
	 *---------------------------------------------------*/
	client := req.NewClient()

	resp, err := client.R().
		SetBodyJsonBytes(reqBodyJSON).
		Post(url.APIQuery())
	if err != nil {
		return errors.Wrapf(err, "POST has failed")
	}

	if !resp.IsSuccessState() {
		return errors.Errorf("bad response status: %s", resp.Status)
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
