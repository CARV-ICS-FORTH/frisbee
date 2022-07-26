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
	"k8s.io/apimachinery/pkg/util/json"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// +kubebuilder:webhook:path=/mutate-frisbee-dev-v1alpha1-template,mutating=true,failurePolicy=fail,sideEffects=None,groups=frisbee.dev,resources=templates,verbs=create;update,versions=v1alpha1,name=mtemplate.kb.io,admissionReviewVersions={v1,v1alpha1}

var _ webhook.Defaulter = &Template{}

// +kubebuilder:webhook:path=/validate-frisbee-dev-v1alpha1-template,mutating=false,failurePolicy=fail,sideEffects=None,groups=frisbee.dev,resources=templates,verbs=create;update,versions=v1alpha1,name=vtemplate.kb.io,admissionReviewVersions={v1,v1alpha1}

var _ webhook.Validator = &Template{}

// log is for logging in this package.
var templatelog = logf.Log.WithName("template-hook")

func (in *Template) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *Template) Default() {
	templatelog.V(5).Info("default", "name", in.Name)

	// TODO(user): fill in your defaulting logic.
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *Template) ValidateCreate() error {
	templatelog.V(5).Info("validate create", "name", in.Name)

	{ // Ensure the template is ok and there are no brackets missing.
		specBody, err := json.Marshal(in.Spec)
		if err != nil {
			return errors.Wrapf(err, "marshal error")
		}

		if _, err := ExprState(specBody).Evaluate(Scheme{
			Scenario:  "",
			Namespace: "",
			Inputs:    in.Spec.Inputs,
		}); err != nil {
			return errors.Wrapf(err, "Invalid template definition")
		}
	}

	if in.Spec.Service != nil {
		v := Service{
			Spec: *in.Spec.Service,
		}

		return v.ValidateCreate()
	}

	if in.Spec.Chaos != nil {
		v := Chaos{
			Spec: *in.Spec.Chaos,
		}

		return v.ValidateCreate()
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *Template) ValidateUpdate(old runtime.Object) error {
	templatelog.V(5).Info("validate update", "name", in.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *Template) ValidateDelete() error {
	templatelog.V(5).Info("validate delete", "name", in.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
