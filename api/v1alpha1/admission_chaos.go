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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/mutate-frisbee-dev-v1alpha1-chaos,mutating=true,failurePolicy=fail,sideEffects=None,groups=frisbee.dev,resources=chaos,verbs=create;update,versions=v1alpha1,name=mchaos.kb.io,admissionReviewVersions={v1,v1alpha1}

var _ webhook.Defaulter = &Chaos{}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:path=/validate-frisbee-dev-v1alpha1-chaos,mutating=false,failurePolicy=fail,sideEffects=None,groups=frisbee.dev,resources=chaos,verbs=create,versions=v1alpha1,name=vchaos.kb.io,admissionReviewVersions={v1,v1alpha1}

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
	chaoslog.Info("SetDefaults",
		"name", in.GetNamespace()+"/"+in.GetName(),
	)
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (in *Chaos) ValidateCreate() (admission.Warnings, error) {
	chaoslog.Info("ValidateCreateRequest",
		"name", in.GetNamespace()+"/"+in.GetName(),
	)

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (in *Chaos) ValidateUpdate(runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (in *Chaos) ValidateDelete() (admission.Warnings, error) {
	chaoslog.Info("validate delete", "name", in.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
