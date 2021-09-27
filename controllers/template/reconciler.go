// Licensed to FORTH/ICS under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. FORTH/ICS licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

/*
Package template provides a lookup for service definition.

It also provides a way of creating random values at every call, for example to create services that listen
on different ports.
*/
package template

import (
	"context"
	"reflect"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/template/helpers"
	"github.com/fnikolai/frisbee/controllers/utils"
	"github.com/fnikolai/frisbee/controllers/utils/lifecycle"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewController(mgr ctrl.Manager, logger logr.Logger) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Template{}).
		Named("template").
		Complete(&Reconciler{
			Manager: mgr,
			Logger:  logger.WithName("template"),
		})
}

// Reconciler reconciles a Templates object
type Reconciler struct {
	ctrl.Manager
	logr.Logger
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var obj v1alpha1.Template

	var ret bool
	result, err := utils.Reconcile(ctx, r, req, &obj, &ret)
	if ret {
		return result, err
	}

	// if the template is already registered, there is nothing else to do.
	if obj.Status.IsRegistered {
		return utils.Stop()
	}

	// validate services
	for name, spec := range obj.Spec.Services {
		if _, err := helpers.GenerateServiceSpec(&spec); err != nil {
			return lifecycle.Failed(ctx, r, &obj, errors.Wrapf(err, "service template %s error", name))
		}
	}

	// validate monitors
	for name, spec := range obj.Spec.Monitors {
		if _, err := helpers.GenerateMonitorSpec(&spec); err != nil {
			return lifecycle.Failed(ctx, r, &obj, errors.Wrapf(err, "monitor template %s error", name))
		}
	}

	r.Logger.Info("Import Template",
		"name", req.NamespacedName,
		"services", GetServiceNames(obj.Spec),
		"monitor", GetMonitorNames(obj.Spec),
	)

	obj.Status.IsRegistered = true

	return lifecycle.Running(ctx, r, &obj, "all templates are loaded")
}

func (r *Reconciler) Finalizer() string {
	return "templates.frisbee.io/finalizer"
}

func (r *Reconciler) Finalize(obj client.Object) error {
	r.Logger.Info("Finalize", "kind", reflect.TypeOf(obj), "name", obj.GetName())

	return nil
}

func GetServiceNames(t v1alpha1.TemplateSpec) []string {
	names := make([]string, 0, len(t.Services))

	for name := range t.Services {
		names = append(names, name)
	}

	return names
}

func GetMonitorNames(t v1alpha1.TemplateSpec) []string {
	names := make([]string, 0, len(t.Monitors))

	for name := range t.Monitors {
		names = append(names, name)
	}

	return names
}
