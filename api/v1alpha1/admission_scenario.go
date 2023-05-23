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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.

// +kubebuilder:webhook:path=/mutate-frisbee-dev-v1alpha1-scenario,mutating=true,failurePolicy=fail,sideEffects=None,groups=frisbee.dev,resources=scenarios,verbs=create;update,versions=v1alpha1,name=mscenario.kb.io,admissionReviewVersions={v1,v1alpha1}

var _ webhook.Defaulter = &Scenario{}

// +kubebuilder:webhook:path=/validate-frisbee-dev-v1alpha1-scenario,mutating=false,failurePolicy=fail,sideEffects=None,groups=frisbee.dev,resources=scenarios,verbs=create,versions=v1alpha1,name=vscenario.kb.io,admissionReviewVersions={v1,v1alpha1}

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

	// Align Inputs with MaxInstances
	for i := 0; i < len(in.Spec.Actions); i++ {
		action := &in.Spec.Actions[i]

		switch action.ActionType {
		case ActionService:
			if err := action.Service.Prepare(false); err != nil {
				scenariolog.Error(err, "definition error", "action", action.Name)
			}

		case ActionCluster:
			if err := action.Cluster.GenerateObjectFromTemplate.Prepare(true); err != nil {
				scenariolog.Error(err, "definition error", "action", action.Name)
			}

		case ActionChaos:
			if err := action.Chaos.Prepare(false); err != nil {
				scenariolog.Error(err, "definition error", "action", action.Name)
			}

		case ActionCascade:
			if err := action.Cascade.GenerateObjectFromTemplate.Prepare(true); err != nil {
				scenariolog.Error(err, "definition error", "action", action.Name)
			}

		case ActionCall, ActionDelete:
			// calls and deletes do not involve templates.
			continue
		}
	}
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (in *Scenario) ValidateCreate() (admission.Warnings, error) {
	legitReferences, err := BuildDependencyGraph(in)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid scenario [%s]", in.GetName())
	}

	for i, action := range in.Spec.Actions {
		// Check that expressions used in the assertions are ok
		if !action.Assert.IsZero() {
			if err := ValidateExpr(action.Assert); err != nil {
				return nil, errors.Wrapf(err, "Invalid expr in assertion")
			}
		}

		// Ensure that the type of action is supported and is correctly set
		if err := CheckAction(&in.Spec.Actions[i], legitReferences); err != nil {
			return nil, errors.Wrapf(err, "incorrent spec for type [%s] of action [%s]", action.ActionType, action.Name)
		}
	}

	if err := CheckForBoundedExecution(legitReferences); err != nil {
		return nil, errors.Wrapf(err, "infinity error")
	}

	return nil, nil
}

// BuildDependencyGraph validates the execution workflow.
// 1. Ensures that action names are qualified (since they are used as generators to jobs)
// 2. Ensures that there are no two actions with the same name.
// 3. Ensure that dependencies point to a valid action.
// 4. Ensure that macros point to a valid action.
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

		// validate references dependencies
		if deps := action.DependsOn; deps != nil {
			for _, dep := range deps.Running {
				if _, exists := callIndex[dep]; !exists {
					return nil, errors.Errorf("invalid running dependency: [%s]<-[%s]", action.Name, dep)
				}
			}

			for _, dep := range deps.Success {
				if _, exists := callIndex[dep]; !exists {
					return nil, errors.Errorf("invalid success dependency: [%s]<-[%s]", action.Name, dep)
				}
			}
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

func CheckForBoundedExecution(callIndex map[string]*Action) error {
	// Use transactions as a means to detect looping containers that never terminate within
	// the lifespan of the scenario. If so, the experiment never ends and waste resources.
	// The idea is find which Actions (Services, Clusters, ...) are not referenced by a
	// terminal dependency condition (e.g, success), and mark as suspects for looping.
	jobCompletionIndex := make(map[string]bool, len(callIndex))

	// Mark every action as uncompleted.
	for _, action := range callIndex {
		jobCompletionIndex[action.Name] = false
	}

	// Do a mockup "run" and mark completed jobs
	for _, action := range callIndex {
		// Successful actions are regarded as completed.
		if deps := action.DependsOn; deps != nil {
			for _, dep := range deps.Success {
				if _, exists := callIndex[dep]; !exists {
					return errors.Errorf("invalid success dependency [%s]<-[%s]", action.Name, dep)
				}

				jobCompletionIndex[dep] = true
			}
		}

		// Deleted actions are regarded as completed.
		if action.ActionType == ActionDelete {
			for _, job := range action.Delete.Jobs {
				completed, exists := jobCompletionIndex[job]
				if !exists {
					return errors.Errorf("internal error. job '%s' does not exist. This should be captured by reference graph", job)
				}

				if completed {
					return errors.Errorf("action.[%s].Delete[%s] deletes an already completed job", action.Name, job)
				}

				// mark the job as completed
				jobCompletionIndex[job] = true
			}

			// If it's a Teardown action, mark it as completed.
			if action.ActionType == ActionDelete && action.Name == "teardown" {
				jobCompletionIndex[action.Name] = true
			}
		}
	}

	// Find jobs are that not completed
	var nonCompleted []string

	for actionName, completed := range jobCompletionIndex {
		if !completed {
			nonCompleted = append(nonCompleted, actionName)
		}
	}

	if len(nonCompleted) > 0 {
		return errors.Errorf("actions '%s' are neither completed nor waited at the end of the scenario", nonCompleted)
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

		var cluster Cluster
		cluster.Spec = *action.EmbedActions.Cluster

		_, err := cluster.ValidateCreate()
		if err != nil {
			return errors.Wrapf(err, "cluster error")
		}

		// validated here because it involves references to other actions.
		if placement := cluster.Spec.Placement; placement != nil {
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

		var cascade Cascade
		cascade.Spec = *action.EmbedActions.Cascade

		_, err := cascade.ValidateCreate()
		return err

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

		var call Call
		call.Spec = *action.EmbedActions.Call

		_, err := call.ValidateCreate()
		return err

	default:
		return errors.Errorf("Unknown action")
	}
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (in *Scenario) ValidateUpdate(runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (in *Scenario) ValidateDelete() (admission.Warnings, error) {
	scenariolog.Info("validate delete", "name", in.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
