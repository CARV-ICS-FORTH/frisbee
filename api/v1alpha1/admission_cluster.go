/*
Copyright 2021 ICS-FORTH.

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
)

// +kubebuilder:webhook:path=/mutate-frisbee-dev-v1alpha1-cluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=frisbee.dev,resources=clusters,verbs=create;update,versions=v1alpha1,name=mcluster.kb.io,admissionReviewVersions={v1,v1alpha1}

var _ webhook.Defaulter = &Cluster{}

// +kubebuilder:webhook:path=/validate-frisbee-dev-v1alpha1-cluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=frisbee.dev,resources=clusters,verbs=create;update,versions=v1alpha1,name=vcluster.kb.io,admissionReviewVersions={v1,v1alpha1}

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
	clusterlog.Info("default", "name", in.Name)

	/*
		Make the inputs consistent with MaxInstances.
	*/
	if err := in.Spec.GenerateObjectFromTemplate.Prepare(true); err != nil {
		clusterlog.Error(err, "template error")
	}

	// Schedule field
	if schedule := in.Spec.Schedule; schedule != nil {
		if schedule.StartingDeadlineSeconds == nil {
			schedule.StartingDeadlineSeconds = &DefaultStartingDeadlineSeconds
		}
	}
	// TODO(user): fill in your defaulting logic.
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (in *Cluster) ValidateCreate() error {
	clusterlog.Info("validate create", "name", in.Name)

	// Set missing values for the template
	if err := in.Spec.GenerateObjectFromTemplate.Prepare(true); err != nil {
		clusterlog.Error(err, "template error")
	}

	// Resources field
	if resources := in.Spec.Resources; resources != nil {
		if in.Spec.Until != nil {
			return errors.Errorf("resource distribution conflicts with Until conditions")
		}

		if in.Spec.MaxInstances < 1 {
			return errors.Errorf("resource distribution requires at least one services")
		}

		if err := ValidateDistribution(resources.DistributionSpec); err != nil {
			return errors.Wrapf(err, "distribution error")
		}
	}

	// TestData field
	if testdata := in.Spec.TestData; testdata != nil {
		clusterlog.Info("TestData validation is missing.")
		// todo: add conditions
	}

	// Tolerate field
	if tolerate := in.Spec.Tolerate; tolerate != nil {
		if err := ValidateTolerate(tolerate); err != nil {
			return errors.Wrapf(err, "tolerate error")
		}
	}

	// Until field
	if until := in.Spec.Until; until != nil {
		if err := ValidateExpr(until); err != nil {
			return errors.Wrapf(err, "until error")
		}
	}

	// Schedule field
	if schedule := in.Spec.Schedule; schedule != nil {
		if err := ValidateScheduler(in.Spec.MaxInstances, schedule); err != nil {
			return errors.Wrapf(err, "schedule error")
		}
	}

	// Suspend Field
	if suspend := in.Spec.Suspend; suspend != nil {
		if *suspend {
			return errors.Errorf("Cannot create a cluster that is already suspended")
		}
	}

	// Placement Field
	// -- Validated in the scenario, because it involves references to other actions

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (in *Cluster) ValidateUpdate(runtime.Object) error {
	clusterlog.Info("validate update", "name", in.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (in *Cluster) ValidateDelete() error {
	clusterlog.Info("validate delete", "name", in.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
