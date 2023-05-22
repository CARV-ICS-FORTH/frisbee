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

package v1alpha1

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.

// +kubebuilder:webhook:path=/mutate-frisbee-dev-v1alpha1-cascade,mutating=true,failurePolicy=fail,sideEffects=None,groups=frisbee.dev,resources=cascades,verbs=create;update,versions=v1alpha1,name=mcascade.kb.io,admissionReviewVersions={v1,v1alpha1}

var _ webhook.Defaulter = &Cascade{}

// +kubebuilder:webhook:path=/validate-frisbee-dev-v1alpha1-cascade,mutating=false,failurePolicy=fail,sideEffects=None,groups=frisbee.dev,resources=cascades,verbs=create,versions=v1alpha1,name=vcascade.kb.io,admissionReviewVersions={v1,v1alpha1}

var _ webhook.Validator = &Cascade{}

// log is for logging in this package.
var cascadelog = logf.Log.WithName("cascade-hook")

func (in *Cascade) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// Default implements webhook.Defaulter so a webhook will be registered for the type.
func (in *Cascade) Default() {
	cascadelog.Info("SetDefaults",
		"name", in.GetNamespace()+"/"+in.GetName(),
	)

	// Schedule field
	if schedule := in.Spec.Schedule; schedule != nil {
		if schedule.StartingDeadlineSeconds == nil {
			schedule.StartingDeadlineSeconds = &DefaultStartingDeadlineSeconds
		}
	}
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (in *Cascade) ValidateCreate() error {
	cascadelog.Info("ValidateCreateRequest",
		"name", in.GetNamespace()+"/"+in.GetName(),
	)

	// Set missing values for the template
	if err := in.Spec.GenerateObjectFromTemplate.Prepare(true); err != nil {
		clusterlog.Error(err, "template error")
	}

	// Until field
	if until := in.Spec.SuspendWhen; until != nil {
		if err := ValidateExpr(until); err != nil {
			return errors.Wrapf(err, "SuspendWhen error")
		}
	}

	// Schedule field
	if schedule := in.Spec.Schedule; schedule != nil {
		if in.Spec.MaxInstances < 1 {
			return errors.Errorf("scheduling requires at least one instance")
		}

		if err := ValidateTaskScheduler(schedule); err != nil {
			return errors.Wrapf(err, "schedule error")
		}
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (in *Cascade) ValidateUpdate(runtime.Object) error {
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (in *Cascade) ValidateDelete() error {
	return nil
}
