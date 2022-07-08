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
	"strings"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.

// +kubebuilder:webhook:path=/mutate-frisbee-dev-v1alpha1-scenario,mutating=true,failurePolicy=fail,sideEffects=None,groups=frisbee.dev,resources=scenarios,verbs=create;update,versions=v1alpha1,name=mscenario.kb.io,admissionReviewVersions={v1,v1alpha1}

var _ webhook.Defaulter = &Scenario{}

// +kubebuilder:webhook:path=/validate-frisbee-dev-v1alpha1-scenario,mutating=false,failurePolicy=fail,sideEffects=None,groups=frisbee.dev,resources=scenarios,verbs=create;update,versions=v1alpha1,name=vscenario.kb.io,admissionReviewVersions={v1,v1alpha1}

var _ webhook.Validator = &Scenario{}

// log is for logging in this package.
var scenariolog = logf.Log.WithName("scenario-resource")

func (r *Scenario) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Scenario) Default() {
	scenariolog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Scenario) ValidateCreate() error {
	legitReferences, err := DependencyGraph(r)
	if err != nil {
		return errors.Wrapf(err, "invalid scenario [%s]", r.GetName())
	}

	for _, action := range r.Spec.Actions {
		if err := CheckDependencies(&action, legitReferences); err != nil {
			return errors.Wrapf(err, "dependency error for action [%s]", action.Name)
		}

		if err := CheckAssertions(&action); err != nil {
			return errors.Wrapf(err, "assertion error for action [%s]", action.Name)
		}

		/*
			if err := r.CheckTemplateRef(ctx, scenario, action); err != nil {
				return errors.Wrapf(err, "template reference error for action [%s]", actionName)
			}

		*/

		if err := CheckJobRef(&action, legitReferences); err != nil {
			return errors.Wrapf(err, "job reference error for action [%s]", action.Name)
		}
	}

	return nil
}

func DependencyGraph(scenario *Scenario) (map[string]*Action, error) {

	callIndex := make(map[string]*Action)

	// prepare a dependency graph
	for i, action := range scenario.Spec.Actions {
		// Ensure that the type of action is supported and is correctly set
		if !CheckAction(&action) {
			return nil, errors.Errorf("incorrent spec for type [%s] of action [%s]", action.ActionType, action.Name)
		}

		// Because the action name will be the "matrix" for generating addressable jobs,
		// it must adhere to certain properties.
		if errs := validation.IsDNS1123Subdomain(action.Name); errs != nil {
			err := errors.New(strings.Join(errs, "; "))

			return nil, errors.Wrapf(err, "invalid actioname %s", action.Name)
		}

		_, ok := callIndex[action.Name]
		if ok {
			return nil, errors.Errorf("Duplicate action '%s'", action.Name)
		}

		callIndex[action.Name] = &scenario.Spec.Actions[i]
	}

	return callIndex, nil
}

func CheckAction(act *Action) bool {
	if act == nil || act.EmbedActions == nil {
		return false
	}

	switch act.ActionType {
	case ActionService:
		return act.EmbedActions.Service != nil
	case ActionCluster:
		return act.EmbedActions.Cluster != nil
	case ActionChaos:
		return act.EmbedActions.Chaos != nil
	case ActionCascade:
		return act.EmbedActions.Cascade != nil
	case ActionDelete:
		return act.EmbedActions.Delete != nil
	case ActionCall:
		return act.EmbedActions.Call != nil
	}

	return false
}

func CheckDependencies(action *Action, callIndex map[string]*Action) error {
	if deps := action.DependsOn; deps != nil {
		for _, dep := range deps.Success {
			if _, ok := callIndex[dep]; !ok {
				return errors.Errorf("invalid success dependency [%s]<-[%s]", action.Name, dep)
			}
		}

		for _, dep := range deps.Running {
			if _, ok := callIndex[dep]; !ok {
				return errors.Errorf("invalid running dependency: [%s]<-[%s]", action.Name, dep)
			}
		}
	}

	return nil
}

func CheckAssertions(action *Action) error {
	if assert := action.Assert; !assert.IsZero() {
		if action.Delete != nil {
			return errors.Errorf("Delete job cannot have assertion")
		}

		if assert.HasStateExpr() {
			if _, err := assert.State.Parse(); err != nil {
				return errors.Wrapf(err, "Invalid state expr for action %s", action.Name)
			}
		}

		if assert.HasMetricsExpr() {
			if _, err := assert.Metrics.Parse(); err != nil {
				return errors.Wrapf(err, "Invalid metrics expr for action %s", action.Name)
			}
		}
	}

	return nil
}

func CheckJobRef(action *Action, callIndex map[string]*Action) error {
	switch action.ActionType {
	case ActionDelete:
		// Check that references jobs exist and there are no cycle deletions
		for _, job := range action.Delete.Jobs {
			target, exists := callIndex[job]
			if !exists {
				return errors.Errorf("job [%s] of action [%s] does not exist", job, action.Name)
			}

			if target.ActionType == ActionDelete {
				return errors.Errorf("cycle deletion. job [%s] of action [%s] is a deletion job", job, action.Name)
			}
		}
	}

	/*
		if spec.Type == v1alpha1.FaultKill {
			if action.DependsOn.Success != nil {
				return nil, errors.Errorf("kill is a inject-only chaos. it does not have success. only running")
			}
		}

	*/

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Scenario) ValidateUpdate(old runtime.Object) error {
	scenariolog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Scenario) ValidateDelete() error {
	scenariolog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
