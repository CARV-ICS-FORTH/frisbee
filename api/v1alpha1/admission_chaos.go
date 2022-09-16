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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// +kubebuilder:webhook:path=/mutate-frisbee-dev-v1alpha1-chaos,mutating=true,failurePolicy=fail,sideEffects=None,groups=frisbee.dev,resources=chaos,verbs=create;update,versions=v1alpha1,name=mchaos.kb.io,admissionReviewVersions={v1,v1alpha1}

var _ webhook.Defaulter = &Chaos{}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:path=/validate-frisbee-dev-v1alpha1-chaos,mutating=true,failurePolicy=fail,sideEffects=None,groups=frisbee.dev,resources=chaos,verbs=create;update,versions=v1alpha1,name=vchaos.kb.io,admissionReviewVersions={v1,v1alpha1}

var _ webhook.Validator = &Chaos{}

// log is for logging in this package.
var chaoslog = logf.Log.WithName("chaos-hook")

func (in *Chaos) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// Default implements webhook.Defaulter so a webhook will be registered for the type.
func (in *Chaos) Default() {
	chaoslog.Info("default", "name", in.Name)
	// TODO(user): fill in your defaulting logic.
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (in *Chaos) ValidateCreate() error {
	chaoslog.Info("validate create", "name", in.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (in *Chaos) ValidateUpdate(runtime.Object) error {
	chaoslog.Info("validate update", "name", in.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (in *Chaos) ValidateDelete() error {
	chaoslog.Info("validate delete", "name", in.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
