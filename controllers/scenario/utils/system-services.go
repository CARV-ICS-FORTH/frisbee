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

package utils

import (
	"context"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	serviceutils "github.com/carv-ics-forth/frisbee/controllers/service/utils"
	"github.com/carv-ics-forth/frisbee/pkg/configuration"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func DeployDataviewer(ctx context.Context, reconciler common.Reconciler, scenario *v1alpha1.Scenario) error {
	// Ensure the claim exists, and we do not wait indefinitely.
	if scenario.Spec.TestData != nil {
		claimName := scenario.Spec.TestData.Claim.ClaimName
		key := client.ObjectKey{Namespace: scenario.GetNamespace(), Name: claimName}

		var claim corev1.PersistentVolumeClaim

		if err := reconciler.GetClient().Get(ctx, key, &claim); err != nil {
			return errors.Wrapf(err, "cannot verify existence of testdata claim '%s'", claimName)
		}
	}

	// Now we can use it to create the data viewer
	var job v1alpha1.Service

	job.SetName(common.DefaultDataviewerName)

	// set labels
	v1alpha1.SetScenarioLabel(&job.ObjectMeta, scenario.GetName())
	v1alpha1.SetComponentLabel(&job.ObjectMeta, v1alpha1.ComponentSys)

	{ // spec
		spec, err := serviceutils.GetServiceSpec(ctx, reconciler.GetClient(), scenario, v1alpha1.GenerateObjectFromTemplate{
			TemplateRef:  configuration.DataviewerTemplate,
			MaxInstances: 1,
			Inputs:       nil,
		})
		if err != nil {
			return errors.Wrapf(err, "cannot get spec")
		}

		spec.DeepCopyInto(&job.Spec)

		// the dataviewer is the only service that has complete access to the volume's content.
		serviceutils.AttachTestDataVolume(&job, scenario.Spec.TestData, false)
	}

	if err := common.Create(ctx, reconciler, scenario, &job); err != nil {
		return errors.Wrapf(err, "cannot create %s", job.GetName())
	}

	scenario.Status.DataviewerEndpoint = common.ExternalEndpoint(common.DefaultDataviewerName, scenario.GetNamespace())

	return nil
}

func DeployPrometheus(ctx context.Context, reconciler common.Reconciler, scenario *v1alpha1.Scenario) error {
	var job v1alpha1.Service

	job.SetName(common.DefaultPrometheusName)

	// set labels
	v1alpha1.SetScenarioLabel(&job.ObjectMeta, scenario.GetName())
	v1alpha1.SetComponentLabel(&job.ObjectMeta, v1alpha1.ComponentSys)

	{ // spec
		spec, err := serviceutils.GetServiceSpec(ctx, reconciler.GetClient(), scenario, v1alpha1.GenerateObjectFromTemplate{
			TemplateRef:  configuration.PrometheusTemplate,
			MaxInstances: 1,
			Inputs:       nil,
		})
		if err != nil {
			return errors.Wrapf(err, "cannot get spec")
		}

		spec.DeepCopyInto(&job.Spec)

		// NOTICE: Prometheus does not support NFS or other distributed filesystems. It returns
		// panic: Unable to create mmap-ed active query log
		// We have this line here commented, just to make the point of **DO NOT UNCOMMENT IT**.
		// job.AttachTestDataVolume(scenario.Spec.TestData, true)
	}

	if err := common.Create(ctx, reconciler, scenario, &job); err != nil {
		return errors.Wrapf(err, "cannot create %s", job.GetName())
	}

	scenario.Status.PrometheusEndpoint = common.ExternalEndpoint(common.DefaultPrometheusName, scenario.GetNamespace())

	return nil
}

func DeployGrafana(ctx context.Context, reconciler common.Reconciler, scenario *v1alpha1.Scenario, agentRefs []string) error {
	var job v1alpha1.Service

	job.SetName(common.DefaultGrafanaServiceName)

	v1alpha1.SetScenarioLabel(&job.ObjectMeta, scenario.GetName())
	v1alpha1.SetComponentLabel(&job.ObjectMeta, v1alpha1.ComponentSys)

	{ // spec
		spec, err := serviceutils.GetServiceSpec(ctx, reconciler.GetClient(), scenario, v1alpha1.GenerateObjectFromTemplate{
			TemplateRef:  configuration.GrafanaTemplate,
			MaxInstances: 1,
			Inputs:       nil,
		})
		if err != nil {
			return errors.Wrapf(err, "cannot get spec")
		}

		spec.DeepCopyInto(&job.Spec)

		serviceutils.AttachTestDataVolume(&job, scenario.Spec.TestData, true)

		if err := InstallGrafanaDashboards(ctx, reconciler, scenario, &job.Spec, agentRefs); err != nil {
			return errors.Wrapf(err, "import dashboards")
		}
	}

	if err := common.Create(ctx, reconciler, scenario, &job); err != nil {
		return errors.Wrapf(err, "cannot create %s", job.GetName())
	}

	scenario.Status.GrafanaEndpoint = common.ExternalEndpoint(common.DefaultGrafanaServiceName, scenario.GetNamespace())

	return nil
}
