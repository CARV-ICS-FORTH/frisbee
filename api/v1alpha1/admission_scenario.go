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

// +kubebuilder:webhook:path=/validate-frisbee-dev-v1alpha1-scenario,mutating=true,failurePolicy=fail,sideEffects=None,groups=frisbee.dev,resources=scenarios,verbs=create;update,versions=v1alpha1,name=vscenario.kb.io,admissionReviewVersions={v1,v1alpha1}

var _ webhook.Validator = &Scenario{}

// log is for logging in this package.
var scenariolog = logf.Log.WithName("scenario-hook")

func (in *Scenario) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

// Default implements webhook.Defaulter so a webhook will be registered for the type.
func (in *Scenario) Default() {
	scenariolog.Info("default", "name", in.Name)
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (in *Scenario) ValidateCreate() error {
	legitReferences, err := BuildDependencyGraph(in)
	if err != nil {
		return errors.Wrapf(err, "invalid scenario [%s]", in.GetName())
	}

	// Use transactions as a means to detect looping containers that never terminate within
	// the lifespan of the scenario. If so, the experiment never ends and waste resources.
	// The idea is find which Actions (Services, Clusters, ...) are not referenced by a
	// terminal dependency condition (e.g, success), and mark as suspects for looping.
	txIndex := make(map[string]bool, len(legitReferences))

	for _, action := range in.Spec.Actions {
		// Check the referenced dependencies are ok
		if err := CheckDependencyGraph(action, legitReferences, txIndex); err != nil {
			return errors.Wrapf(err, "dependency error for action [%s]", action.Name)
		}

		// Check that expressions used in the assertions are ok
		if !action.Assert.IsZero() {
			if err := ValidateExpr(action.Assert); err != nil {
				return errors.Wrapf(err, "Invalid expr in assertion")
			}
		}

		// Ensure that the type of action is supported and is correctly set
		if err := CheckAction(&action, legitReferences); err != nil {
			return errors.Wrapf(err, "incorrent spec for type [%s] of action [%s]", action.ActionType, action.Name)
		}

		/*
			if err := r.CheckTemplateRef(ctx, scenario, action); err != nil {
				return errors.Wrapf(err, "template reference error for action [%s]", actionName)
			}

		*/
	}

	// raise a warning if there are opened looping services that remain active after all actions are scheduled.
	for actionName, completed := range txIndex {
		if !completed {
			return errors.Errorf("Action '%s' remains unbounded at the end of the scenario.", actionName)
		}
	}

	return nil
}

func BuildDependencyGraph(scenario *Scenario) (map[string]*Action, error) {
	// callIndex maintains a map of all the action in the scenario
	callIndex := make(map[string]*Action, len(scenario.Spec.Actions))

	// prepare a dependency graph
	for i, action := range scenario.Spec.Actions {
		// Because the action name will be the "matrix" for generating addressable jobs,
		// it must adhere to certain properties.
		if errs := validation.IsDNS1123Subdomain(action.Name); errs != nil {
			err := errors.New(strings.Join(errs, "; "))

			return nil, errors.Wrapf(err, "invalid actioname %s", action.Name)
		}

		// update calling map
		if _, exists := callIndex[action.Name]; !exists {
			callIndex[action.Name] = &scenario.Spec.Actions[i]
		} else {
			return nil, errors.Errorf("Duplicate action '%s'", action.Name)
		}
	}

	return callIndex, nil
}

func CheckDependencyGraph(action Action, callIndex map[string]*Action, txIndex map[string]bool) error {
	// find invalid dependency and update txIndex with completed actions.
	if deps := action.DependsOn; deps != nil {
		for _, dep := range deps.Running {
			if _, exists := callIndex[dep]; !exists {
				return errors.Errorf("invalid running dependency: [%s]<-[%s]", action.Name, dep)
			}

			// In general, we assume that an Action A with a "Running" dependency to Action B,
			// will leave the Action B unchanged, after Action A is complete. Exception to this is "Delete" Action,
			// which will delete (and therefore 'complete') the Action B.
			if action.ActionType != ActionDelete {
				// mark the action as opened
				txIndex[dep] = false
			} else {
				// mark the action as completed
				txIndex[dep] = true
			}
		}

		for _, dep := range deps.Success {
			if _, exists := callIndex[dep]; !exists {
				return errors.Errorf("invalid success dependency [%s]<-[%s]", action.Name, dep)
			}

			// mark the action as completed.
			txIndex[dep] = true
		}
	}

	return nil
}

func CheckAction(action *Action, references map[string]*Action) error {
	if action == nil || action.EmbedActions == nil {
		return errors.Errorf("empty definition")
	}

	switch action.ActionType {
	case ActionService:
		if action.EmbedActions.Service == nil {
			return errors.Errorf("empty service definition")
		}

		return nil

	case ActionCluster:
		if action.EmbedActions.Cluster == nil {
			return errors.Errorf("empty cluster definition")
		}

		v := &Cluster{
			Spec: *action.EmbedActions.Cluster,
		}

		if err := v.ValidateCreate(); err != nil {
			return errors.Wrapf(err, "cluster error")
		}

		if placement := v.Spec.Placement; placement != nil {
			if err := ValidatePlacement(placement, references); err != nil {
				return errors.Wrapf(err, "placement error")
			}
		}

		return nil

	case ActionChaos:
		if action.EmbedActions.Chaos == nil {
			return errors.Errorf("empty chaos definition")
		}

		/*
			if spec.Type == v1alpha1.FaultKill {
				if action.DependsOn.Success != nil {
					return nil, errors.Errorf("kill is a inject-only chaos. it does not have success. only running")
				}
			}

		*/

		return nil

	case ActionCascade:
		if action.EmbedActions.Cascade == nil {
			return errors.Errorf("empty cascade definition")
		}

		v := &Cascade{
			Spec: *action.EmbedActions.Cascade,
		}

		return v.ValidateCreate()
	case ActionDelete:
		if action.EmbedActions.Delete == nil {
			return errors.Errorf("empty delete definition")
		}

		// Check that references jobs exist and there are no cycle deletions
		for _, job := range action.EmbedActions.Delete.Jobs {
			target, exists := references[job]
			if !exists {
				return errors.Errorf("referenced job '%s' does not exist", job)
			}

			if target.ActionType == ActionDelete {
				return errors.Errorf("cycle deletion. referected job '%s' should not be a deletion job", job)
			}
		}

		return nil
	case ActionCall:
		if action.EmbedActions.Call == nil {
			return errors.Errorf("empty call definition")
		}

		v := &Call{
			Spec: *action.EmbedActions.Call,
		}

		return v.ValidateCreate()

	default:
		return errors.Errorf("Unknown action")
	}
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (in *Scenario) ValidateUpdate(runtime.Object) error {
	scenariolog.Info("validate update", "name", in.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (in *Scenario) ValidateDelete() error {
	scenariolog.Info("validate delete", "name", in.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
