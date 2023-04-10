/*
Copyright 2021-2023 ICS-FORTH.

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

package cluster

import (
	"context"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	clusterutils "github.com/carv-ics-forth/frisbee/controllers/cluster/utils"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	serviceutils "github.com/carv-ics-forth/frisbee/controllers/service/utils"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

func (r *Controller) runJob(ctx context.Context, cluster *v1alpha1.Cluster, jobIndex int) error {
	var job v1alpha1.Service

	// Populate the job
	job.SetName(common.GenerateName(cluster, jobIndex))
	v1alpha1.PropagateLabels(&job, cluster)

	// modulo is needed to re-iterate the job list, required for the implementation of "Until".
	jobSpec := cluster.Status.QueuedJobs[jobIndex%len(cluster.Status.QueuedJobs)]

	jobSpec.DeepCopyInto(&job.Spec)

	serviceutils.AttachTestDataVolume(&job, cluster.Spec.TestData, true)

	if err := common.Create(ctx, r, cluster, &job); err != nil {
		return err
	}

	r.GetEventRecorderFor(cluster.GetName()).Event(cluster, corev1.EventTypeNormal, "Scheduled", job.GetName())

	return nil
}

func (r *Controller) constructJobSpecList(ctx context.Context, cluster *v1alpha1.Cluster) ([]v1alpha1.ServiceSpec, error) {
	serviceSpecs, err := serviceutils.GetServiceSpecList(ctx, r.GetClient(), cluster, cluster.Spec.GenerateObjectFromTemplate)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get serviceSpecs")
	}

	clusterutils.SetPlacement(cluster, serviceSpecs)

	clusterutils.SetResources(cluster, serviceSpecs)

	clusterutils.SetTimeline(cluster)

	return serviceSpecs, nil
}
