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

package workflow

import (
	"context"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/telemetry/grafana"
	"github.com/carv-ics-forth/frisbee/controllers/utils"
	"github.com/carv-ics-forth/frisbee/controllers/utils/expressions"
	notifier "github.com/golanghelper/grafana-webhook"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *Controller) getJob(ctx context.Context, w *v1alpha1.Workflow, action v1alpha1.Action) (client.Object, error) {
	logrus.Warn("Handle job ", action.Name)

	switch action.ActionType {
	case "Service":
		return r.service(ctx, w, action)

	case "Cluster":
		return r.cluster(ctx, w, action)

	case "Chaos":
		return r.chaos(ctx, w, action)

	case "Cascade":
		return r.cascade(ctx, w, action)

	default:
		return nil, errors.Errorf("unknown action %s", action.ActionType)
	}
}

func (r *Controller) service(ctx context.Context, w *v1alpha1.Workflow, action v1alpha1.Action) (client.Object, error) {
	if err := expandInput(ctx, r, w.GetNamespace(), &action.Service.Inputs); err != nil {
		return nil, errors.Wrapf(err, "input error")
	}

	// get the service template
	if err := action.Service.Validate(false); err != nil {
		return nil, errors.Wrapf(err, "template validation")
	}

	spec, err := r.serviceControl.GetServiceSpec(ctx, w.GetNamespace(), *action.Service)
	if err != nil {
		return nil, errors.Wrapf(err, "service spec")
	}

	var service v1alpha1.Service

	spec.DeepCopyInto(&service.Spec)
	service.SetName(action.Name)

	return &service, nil
}

func (r *Controller) cluster(ctx context.Context, w *v1alpha1.Workflow, action v1alpha1.Action) (client.Object, error) {
	if err := expandInput(ctx, r, w.GetNamespace(), &action.Cluster.Inputs); err != nil {
		return nil, errors.Wrapf(err, "input error")
	}

	var cluster v1alpha1.Cluster

	action.Cluster.DeepCopyInto(&cluster.Spec)
	cluster.SetName(action.Name)

	return &cluster, nil
}

func (r *Controller) chaos(ctx context.Context, w *v1alpha1.Workflow, action v1alpha1.Action) (client.Object, error) {
	if err := expandInput(ctx, r, w.GetNamespace(), &action.Chaos.Inputs); err != nil {
		return nil, errors.Wrapf(err, "input error")
	}

	logrus.Warn("INPUTS ", action.Chaos.Inputs)

	// get the service template
	if err := action.Chaos.Validate(false); err != nil {
		return nil, errors.Wrapf(err, "template validation")
	}

	spec, err := r.chaosControl.GetChaosSpec(ctx, w.GetNamespace(), *action.Chaos)
	if err != nil {
		return nil, errors.Wrapf(err, "service spec")
	}

	if spec.Type == v1alpha1.FaultKill {
		if action.DependsOn.Success != nil {
			return nil, errors.Errorf("kill is a inject-only chaos. it does not have success. only running")
		}
	}

	var chaos v1alpha1.Chaos

	spec.DeepCopyInto(&chaos.Spec)
	chaos.SetName(action.Name)

	return &chaos, nil
}

func (r *Controller) cascade(ctx context.Context, w *v1alpha1.Workflow, action v1alpha1.Action) (client.Object, error) {
	if err := expandInput(ctx, r, w.GetNamespace(), &action.Cascade.Inputs); err != nil {
		return nil, errors.Wrapf(err, "input error")
	}

	var cascade v1alpha1.Cascade

	action.Cascade.DeepCopyInto(&cascade.Spec)
	cascade.SetName(action.Name)

	return &cascade, nil
}

func (r *Controller) ConnectToGrafana(ctx context.Context, cr *v1alpha1.Workflow) error {
	endpoint := utils.DefaultConfiguration.GrafanaEndpoint

	return grafana.NewGrafanaClient(ctx, r, endpoint,
		// Set a callback that will be triggered when there is Grafana alert.
		// Through this channel we can get informed for SLA violations.
		grafana.WithNotifyOnAlert(func(b *notifier.Body) {
			r.Logger.Info("Grafana Alert", "body", b)

			// when Grafana fires an alert, this alert is captured by the Webhook.
			// The webhook must someone notify the appropriate controller.
			// To do that, it adds information of the fired alert to the object's metadata
			// and updates (patches) the object.
			if err := expressions.DispatchAlert(ctx, r, b); err != nil {
				r.Logger.Error(err, "unable to inform CR for SLA violation", "cr", cr.GetName())
			}
		}),
	)
}
