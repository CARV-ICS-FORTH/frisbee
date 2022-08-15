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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"strings"
)

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.

// +kubebuilder:webhook:path=/mutate-frisbee-dev-v1alpha1-service,mutating=true,failurePolicy=fail,sideEffects=None,groups=frisbee.dev,resources=services,verbs=create;update,versions=v1alpha1,name=mservice.kb.io,admissionReviewVersions={v1,v1alpha1}

var _ webhook.Defaulter = &Service{}

// +kubebuilder:webhook:path=/validate-frisbee-dev-v1alpha1-service,mutating=false,failurePolicy=fail,sideEffects=None,groups=frisbee.dev,resources=services,verbs=create;update,versions=v1alpha1,name=vservice.kb.io,admissionReviewVersions={v1,v1alpha1}

var _ webhook.Validator = &Service{}

// log is for logging in this package.
var servicelog = logf.Log.WithName("service-hook")

func (in *Service) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *Service) Default() {
	servicelog.V(5).Info("default", "name", in.Name)

	// TODO(user): fill in your defaulting logic.
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *Service) ValidateCreate() error {
	servicelog.V(5).Info("validate create", "name", in.GetName())

	for _, container := range in.Spec.Containers {
		if container.Name == MainContainerName { // Validate Main container(s)
			if err := in.validateMainContainer(&container); err != nil {
				return errors.Wrapf(err, "error in service template '%s'", in.GetName())
			}
		} else { // Validate Sidecar container(s)
			if err := in.validateSidecarContainer(&container); err != nil {
				return errors.Wrapf(err, "error in service template '%s'", in.GetName())
			}
		}
	}

	return nil
}

func (in *Service) validateMainContainer(c *corev1.Container) error {
	// Ensure that are no other main containers
	if len(in.Spec.Containers) > 1 {
		return errors.Errorf("Only one container can defined in the template of a Main container")
	}

	// Ensure that there are no sidecar decorations
	if _, exists := in.Spec.Decorators.Annotations[SidecarTelemetry]; exists {
		return errors.Errorf("Unclear if it's a main container or a telemetry sidecar")
	}

	return nil
}

func (in *Service) validateSidecarContainer(c *corev1.Container) error {
	if in.Spec.Decorators.Annotations == nil {
		return errors.Errorf("Follow either the Main container or the Sidecar container rules.")
	}

	// Ensure that there are no sidecar decorations
	if value, exists := in.Spec.Decorators.Annotations[SidecarTelemetry]; exists {
		if value == MainContainerName {
			return errors.Errorf("Follow either the Main container or the Sidecar container rules. Not both")
		}

		if value != c.Name {
			return errors.Errorf("Invalid annotation. Expected '%s:%s' but got '%s:%s'",
				SidecarTelemetry, c.Name,
				SidecarTelemetry, value)
		}

		// Ensure that container is discoverable by Prometheus
		for _, port := range c.Ports {
			if !strings.HasPrefix(port.Name, PrometheusDiscoverablePort) {
				return errors.Errorf(`Because container '%s' is defined as a Telemetry Agent,
									Port '%s' should be prefixed by '%s'`, c.Name, port.Name, PrometheusDiscoverablePort)
			}
		}

		return nil
	}

	return errors.Errorf("no sidecar annotations where found")
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *Service) ValidateUpdate(old runtime.Object) error {
	servicelog.V(5).Info("validate update", "name", in.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *Service) ValidateDelete() error {
	servicelog.V(5).Info("validate delete", "name", in.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
