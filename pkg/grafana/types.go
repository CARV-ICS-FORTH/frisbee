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
	"time"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var defaultLogger = zap.New(zap.UseDevMode(true)).WithName("default.grafana")

type RawTimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type TimeRange struct {
	From time.Time     `json:"from"`
	To   time.Time     `json:"to"`
	Raw  *RawTimeRange `json:"raw"`
}

// Tag is used as tag for Grafana Annotations
type Tag = string

const (
	TagCreated = "create"
	TagDeleted = "delete"
	TagFailed  = "failed"
	TagChaos   = "chaos"
)

// Annotation provides a way to mark points on the graph with rich events.
type Annotation interface {
	// Add  pushes an annotation to grafana indicating that a new component has joined the experiment.
	Add(obj client.Object)

	// Delete pushes an annotation to grafana indicating that a new component has left the experiment.
	Delete(obj client.Object)
}

// DataRequest is used to ask DataFrame from Grafana.
type DataRequest struct {
	Queries []interface{} `json:"queries"`

	Range TimeRange `json:"range"`
	From  string    `json:"from"`
	To    string    `json:"to"`
}

// DefaultVariableEvaluation contains the default replacement of environment variables in Grafana expressions.
var DefaultVariableEvaluation = map[string]string{
	"Instance": ".+",
	"Node":     ".+",
}

const (
	respAnnotationAddOK = "Annotation added"

	respAnnotationAddError = "Failed to save annotation"

	respAnnotationPatchOK = "Annotation patched"

	respAnnotationPatchError = "Failed to update annotation"

	respUnauthorizedError = "Unauthorized"

	respAlertSuccess = "success"
)

var healthError = errors.New("Grafana does not seam healthy")

var Timeout = 2 * time.Minute
