/*
Copyright 2022 ICS-FORTH.

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

// +kubebuilder:webhook:path=/mutate-frisbee-dev-v1alpha1-call,mutating=true,failurePolicy=fail,sideEffects=None,groups=frisbee.dev,resources=calls,verbs=create;update,versions=v1alpha1,name=mcall.kb.io,admissionReviewVersions={v1,v1alpha1}

var _ webhook.Defaulter = &Call{}

// +kubebuilder:webhook:path=/validate-frisbee-dev-v1alpha1-call,mutating=false,failurePolicy=fail,sideEffects=None,groups=frisbee.dev,resources=calls,verbs=create,versions=v1alpha1,name=vcall.kb.io,admissionReviewVersions={v1,v1alpha1}

var _ webhook.Validator = &Call{}

// log is for logging in this package.
var calllog = logf.Log.WithName("call-hook")

func (in *Call) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// Default implements webhook.Defaulter so a webhook will be registered for the type.
func (in *Call) Default() {
	calllog.V(5).Info("default", "name", in.Name)

	// Schedule field
	if schedule := in.Spec.Schedule; schedule != nil {
		if schedule.StartingDeadlineSeconds == nil {
			schedule.StartingDeadlineSeconds = &DefaultStartingDeadlineSeconds
		}
	}

	// TODO(user): fill in your defaulting logic.
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (in *Call) ValidateCreate() error {
	calllog.V(5).Info("validate create", "name", in.Name)

	// Expect field
	if expect := in.Spec.Expect; expect != nil {
		if len(expect) != len(in.Spec.Services) {
			return errors.Errorf("Expect '%d' outputs for '%d' services", len(expect), len(in.Spec.Services))
		}
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
		if err := ValidateScheduler(len(in.Spec.Services), schedule); err != nil {
			return errors.Wrapf(err, "schedule error")
		}
	}

	// Suspend Field
	if suspend := in.Spec.Suspend; suspend != nil {
		if *suspend {
			return errors.Errorf("Cannot create a call that is already suspended")
		}
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (in *Call) ValidateUpdate(runtime.Object) error {
	calllog.Info("validate update", "name", in.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (in *Call) ValidateDelete() error {
	calllog.Info("validate delete", "name", in.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
