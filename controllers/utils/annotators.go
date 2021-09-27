// Licensed to FORTH/ICS under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. FORTH/ICS licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package utils

import (
	"context"

	"github.com/grafana-tools/sdk"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
)

type Annotator interface {
	Insert(sdk.CreateAnnotationRequest) (id uint)
	Patch(reqID uint, ga sdk.PatchAnnotationRequest)
}

// ///////////////////////////////////////////
//		Grafana Annotator
// ///////////////////////////////////////////

const (
	statusAnnotationAdded = "Annotation added"

	statusAnnotationPatched = "Annotation patched"
)

type GrafanaAnnotator struct {
	ctx context.Context

	*sdk.Client
}

// Insert inserts a new annotation to Grafana
func (c *GrafanaAnnotator) Insert(ga sdk.CreateAnnotationRequest) (reqID uint) {
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()

	// submit
	gaResp, err := c.Client.CreateAnnotation(ctx, ga)
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "annotation failed"))

		return
	}

	if gaResp.Message == nil {
		runtime.HandleError(errors.Wrapf(err, "empty annotation response"))
	} else if *gaResp.Message != statusAnnotationAdded {
		runtime.HandleError(errors.Wrapf(err, "expected message '%s', but got '%s'", statusAnnotationAdded, *gaResp.Message))
	}

	return *gaResp.ID
}

// Patch updates an existing annotation to Grafana
func (c *GrafanaAnnotator) Patch(reqID uint, ga sdk.PatchAnnotationRequest) {
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()

	// submit
	gaResp, err := c.Client.PatchAnnotation(ctx, reqID, ga)
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "annotation failed"))

		return
	}

	if gaResp.Message == nil {
		runtime.HandleError(errors.Wrapf(err, "empty annotation response"))
	} else if *gaResp.Message != statusAnnotationPatched {
		runtime.HandleError(errors.Wrapf(err, "expected message '%s', but got '%s'", statusAnnotationPatched, *gaResp.Message))
	}
}
