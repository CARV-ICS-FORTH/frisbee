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

package v1alpha1

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/mutate-frisbee-dev-v1alpha1-cluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=frisbee.dev,resources=clusters,verbs=create;update,versions=v1alpha1,name=mcluster.kb.io,admissionReviewVersions={v1,v1alpha1}

var _ webhook.Defaulter = &Cluster{}

// +kubebuilder:webhook:path=/validate-frisbee-dev-v1alpha1-cluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=frisbee.dev,resources=clusters,verbs=create,versions=v1alpha1,name=vcluster.kb.io,admissionReviewVersions={v1,v1alpha1}

var _ webhook.Validator = &Cluster{}

// log is for logging in this package.
var clusterlog = logf.Log.WithName("cluster-hook")

func (in *Cluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

// Default implements webhook.Defaulter so a webhook will be registered for the type.
func (in *Cluster) Default() {
	clusterlog.Info("SetDefaults",
		"name", in.GetNamespace()+"/"+in.GetName(),
	)

	// Schedule field
	if schedule := in.Spec.Schedule; schedule != nil {
		if schedule.StartingDeadlineSeconds == nil {
			schedule.StartingDeadlineSeconds = &DefaultStartingDeadlineSeconds
		}
	}

	if in.Spec.DefaultDistributionSpec != nil {
		in.Spec.DefaultDistributionSpec = &DistributionSpec{Name: DistributionConstant}
	}
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (in *Cluster) ValidateCreate() (admission.Warnings, error) {
	clusterlog.Info("ValidateCreateRequest",
		"name", in.GetNamespace()+"/"+in.GetName(),
	)

	// Set missing values for the template
	if err := in.Spec.GenerateObjectFromTemplate.Prepare(true); err != nil {
		clusterlog.Error(err, "template error")
	}

	// TestData field
	if testdata := in.Spec.TestData; testdata != nil {
		clusterlog.Info("TestData validation is missing.")
		// todo: add conditions
	}

	// Tolerate field
	if tolerate := in.Spec.Tolerate; tolerate != nil {
		if err := ValidateTolerate(tolerate); err != nil {
			return nil, errors.Wrapf(err, "tolerate error")
		}
	}

	// Until field
	if until := in.Spec.SuspendWhen; until != nil {
		if err := ValidateExpr(until); err != nil {
			return nil, errors.Wrapf(err, "SuspendWhen error")
		}
	}

	// Schedule field
	if schedule := in.Spec.Schedule; schedule != nil {
		if in.Spec.MaxInstances < 1 {
			return nil, errors.Errorf("scheduling requires at least one instance")
		}

		if err := ValidateTaskScheduler(schedule); err != nil {
			return nil, errors.Wrapf(err, "schedule error")
		}
	}

	// Suspend Field
	if suspend := in.Spec.Suspend; suspend != nil {
		if *suspend {
			return nil, errors.Errorf("Cannot create a cluster that is already suspended")
		}
	}

	// Resources field
	// if distributionSpec is nil, Default() will set it to constant.
	if resources := in.Spec.Resources; resources != nil {
		if in.Spec.SuspendWhen != nil {
			return nil, errors.Errorf("resource distribution conflicts with SuspendWhen conditions")
		}

		if in.Spec.MaxInstances < 1 {
			return nil, errors.Errorf("resource distribution requires at least one services")
		}

		if err := resources.Validate(); err != nil {
			return nil, errors.Wrapf(err, "cluster.resources")
		}
	}

	// Placement Field
	// -- Validated in the scenario, because it involves references to other actions

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (in *Cluster) ValidateUpdate(runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (in *Cluster) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}
