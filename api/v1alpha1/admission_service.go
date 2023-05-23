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
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.

// +kubebuilder:webhook:path=/mutate-frisbee-dev-v1alpha1-service,mutating=true,failurePolicy=fail,sideEffects=None,groups=frisbee.dev,resources=services,verbs=create;update,versions=v1alpha1,name=mservice.kb.io,admissionReviewVersions={v1,v1alpha1}

var _ webhook.Defaulter = &Service{}

// +kubebuilder:webhook:path=/validate-frisbee-dev-v1alpha1-service,mutating=false,failurePolicy=fail,sideEffects=None,groups=frisbee.dev,resources=services,verbs=create,versions=v1alpha1,name=vservice.kb.io,admissionReviewVersions={v1,v1alpha1}

var _ webhook.Validator = &Service{}

// log is for logging in this package.
var servicelog = logf.Log.WithName("service-hook")

func (in *Service) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// Default implements webhook.Defaulter so a webhook will be registered for the type.
func (in *Service) Default() {
	servicelog.Info("SetDefaults",
		"name", in.GetNamespace()+"/"+in.GetName(),
	)
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (in *Service) ValidateCreate() (admission.Warnings, error) {
	servicelog.Info("ValidateCreateRequest",
		"name", in.GetNamespace()+"/"+in.GetName(),
	)

	for i := range in.Spec.Containers {
		container := in.Spec.Containers[i]

		if container.Name == MainContainerName { // Validate Main container(s)
			if err := in.validateMainContainer(&container); err != nil {
				return nil, errors.Wrapf(err, "error in service template '%s'", in.GetName())
			}
		} else { // Validate Sidecar container(s)
			if err := in.validateSidecarContainer(&container); err != nil {
				return nil, errors.Wrapf(err, "service '%s' definition error", in.GetName())
			}
		}
	}

	return nil, nil
}

func (in *Service) validateMainContainer(container *corev1.Container) error {
	// Ensure that there are no sidecar decorations
	if _, exists := in.Spec.Decorators.Annotations[SidecarTelemetry]; exists {
		return errors.Errorf("unclear if it's a main container or a telemetry sidecar")
	}

	return nil
}

func (in *Service) validateSidecarContainer(container *corev1.Container) error {
	// Ensure that there are no sidecar decorations
	if value, exists := in.Spec.Decorators.Annotations[SidecarTelemetry]; exists {
		if value == MainContainerName {
			return errors.Errorf("conflict. Main Container has been marked as sidecar in the decorators.annotation")
		}

		if value != container.Name {
			return errors.Errorf("Invalid annotation. Expected: '%s:%s' but got '%s:%s'",
				SidecarTelemetry, container.Name,
				SidecarTelemetry, value)
		}

		// Ensure that container is discoverable by Prometheus
		for _, port := range container.Ports {
			if !strings.HasPrefix(port.Name, PrometheusDiscoverablePort) {
				return errors.Errorf(`Because container '%s' is defined as a Telemetry Agent,
									Port '%s' should be prefixed by '%s'`, container.Name, port.Name, PrometheusDiscoverablePort)
			}
		}

		return nil
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (in *Service) ValidateUpdate(_ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (in *Service) ValidateDelete() (admission.Warnings, error) {
	servicelog.Info("validate delete", "name", in.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
