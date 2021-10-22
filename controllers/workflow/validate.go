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

package workflow

import (
	"context"
	"net/http"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type workflowValidator struct {
	Client  client.Client
	decoder *admission.Decoder
}

// workflowValidator admints a workflow if the DAG is valid
func (v *workflowValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	w := &v1alpha1.Workflow{}

	if err := v.decoder.Decode(req, w); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	logrus.Warn("YEEEEEEEHA I am a valid validator")

	return admission.Allowed("all quiet in the western front")
}

// InjectDecoder injects the decoder.
func (v *workflowValidator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d

	return nil
}
