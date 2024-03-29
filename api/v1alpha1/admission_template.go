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
	"k8s.io/apimachinery/pkg/util/json"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/mutate-frisbee-dev-v1alpha1-template,mutating=true,failurePolicy=fail,sideEffects=None,groups=frisbee.dev,resources=templates,verbs=create;update,versions=v1alpha1,name=mtemplate.kb.io,admissionReviewVersions={v1,v1alpha1}

var _ webhook.Defaulter = &Template{}

// +kubebuilder:webhook:path=/validate-frisbee-dev-v1alpha1-template,mutating=false,failurePolicy=fail,sideEffects=None,groups=frisbee.dev,resources=templates,verbs=create,versions=v1alpha1,name=vtemplate.kb.io,admissionReviewVersions={v1,v1alpha1}

var _ webhook.Validator = &Template{}

// log is for logging in this package.
var templatelog = logf.Log.WithName("template-hook")

func (in *Template) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

// Default implements webhook.Defaulter so a webhook will be registered for the type.
func (in *Template) Default() {
	templatelog.Info("SetDefaults",
		"name", in.GetNamespace()+"/"+in.GetName(),
	)
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (in *Template) ValidateCreate() (admission.Warnings, error) {
	templatelog.Info("ValidateCreateRequest",
		"name", in.GetNamespace()+"/"+in.GetName(),
	)

	if err := in.validateTemplateLanguage(); err != nil {
		return nil, errors.Wrapf(err, "erroneous template '%s'", in.GetName())
	}

	return nil, nil
}

func (in *Template) validateTemplateLanguage() error {
	{ // Ensure the template is ok and there are no brackets missing.
		body, err := json.Marshal(in.Spec)
		if err != nil {
			return errors.Wrapf(err, "marshal error")
		}

		// these fields are expected to be set at runtime. use dummy just for the validation.
		var inputs interface{}

		if in.Spec.Inputs != nil {
			inputs = struct {
				Inputs *TemplateInputs `json:"inputs"`
			}{
				in.Spec.Inputs,
			}
		}

		if _, err := ExprState(body).Evaluate(inputs); err != nil {
			return errors.Wrapf(err, "template language error")
		}
	}

	if in.Spec.Service != nil {
		service := Service{
			Spec: *in.Spec.Service,
		}

		_, err := service.ValidateCreate()
		return errors.Wrapf(err, "service definition error")
	}

	if in.Spec.Chaos != nil {
		chaos := Chaos{
			Spec: *in.Spec.Chaos,
		}

		_, err := chaos.ValidateCreate()
		return errors.Wrapf(err, "chaos definition error")
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (in *Template) ValidateUpdate(runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (in *Template) ValidateDelete() (admission.Warnings, error) {
	templatelog.Info("validate delete", "name", in.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
