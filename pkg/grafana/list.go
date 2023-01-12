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

package grafana

import (
	"context"

	"github.com/pkg/errors"
)

type Panel struct {
	Title string
	ID    uint
}

// ListPanelsWithData returns a list of Panels ID with  a Grafana dashboard.
func (c *Client) ListPanelsWithData(ctx context.Context, dashboardUID string) ([]Panel, error) {
	if c == nil {
		panic("empty client was given")
	}

	board, _, err := c.Conn.GetDashboardByUID(ctx, dashboardUID)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot retrieve dashboard %s", dashboardUID)
	}

	panels := make([]Panel, 0, len(board.Panels))

	for _, panel := range board.Panels {
		panels = append(panels, Panel{
			Title: panel.Title,
			ID:    panel.ID,
		})
	}

	return panels, nil
}
