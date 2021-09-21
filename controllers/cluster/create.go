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

package cluster

import (
	"context"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
)

func (r *Reconciler) create(ctx context.Context, cluster *v1alpha1.Cluster, serviceList v1alpha1.SList) error {

	for instance := range serviceList.Yield(ctx, cluster.Spec.Schedule) {
		if err := r.GetClient().Create(ctx, instance); err != nil {
			return errors.Wrapf(err, "cannot create service %s", instance.GetName())
		}
	}

	return nil
}
