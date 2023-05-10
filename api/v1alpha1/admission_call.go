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
	calllog.Info("SetDefaults",
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
func (in *Call) ValidateCreate() error {
	calllog.Info("-> ValidateCreate", "obj", in.GetNamespace()+"/"+in.GetName())
	defer calllog.Info("<- ValidateCreate", "obj", in.GetNamespace()+"/"+in.GetName())

	// Expect field
	if expect := in.Spec.Expect; expect != nil {
		if len(expect) != len(in.Spec.Services) {
			return errors.Errorf("Expect '%d' outputs for '%d' services", len(expect), len(in.Spec.Services))
		}
	}

	// Tolerate field
	if err := ValidateTolerate(in.Spec.Tolerate); err != nil {
		return errors.Wrapf(err, "tolerate error")
	}

	// SuspendWhen field
	if err := ValidateExpr(in.Spec.SuspendWhen); err != nil {
		return errors.Wrapf(err, "SuspendWhen error")
	}

	// Schedule field
	if schedule := in.Spec.Schedule; schedule != nil {
		if len(in.Spec.Services) < 1 {
			return errors.Errorf("scheduling requires at least one instance")
		}

		if err := ValidateTaskScheduler(schedule); err != nil {
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
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (in *Call) ValidateDelete() error {
	calllog.Info("validate delete", "name", in.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
