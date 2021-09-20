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

package dataport

import (
	"context"
	"time"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func matchPorts(ctx context.Context, r *Reconciler, criteria *metav1.LabelSelector) v1alpha1.DataPortList {
	var matches v1alpha1.DataPortList

	selector, err := metav1.LabelSelectorAsSelector(criteria)
	if err != nil {
		r.Logger.Error(err, "selector conversion error")

		return matches
	}

	// TODO: Find a way for continuous watching

	time.Sleep(10 * time.Second)

	listOptions := client.ListOptions{LabelSelector: selector}

	if err := r.Client.List(ctx, &matches, &listOptions); err != nil {
		r.Logger.Error(err, "select error")

		return matches
	}

	return matches
}
