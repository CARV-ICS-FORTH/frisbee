/*
Copyright 2022 ICS-FORTH.

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

package common

import (
	"fmt"
	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"io"
	"k8s.io/apimachinery/pkg/util/json"
	"text/template"
)

type OutputType string

const (
	OutputGoTemplate OutputType = "go"
	OutputJSON       OutputType = "json"
	OutputYAML       OutputType = "yaml"
	OutputPretty     OutputType = "pretty"
)

type CliObjRenderer func(ui *ui.UI, obj interface{}) error

func RenderJSON(obj interface{}, w io.Writer) error {
	return json.NewEncoder(w).Encode(obj)
}

func RenderYaml(obj interface{}, w io.Writer) error {
	return yaml.NewEncoder(w).Encode(obj)
}

func RenderGoTemplate(item interface{}, w io.Writer, tpl string) error {
	tmpl, err := template.New("result").Parse(tpl)
	if err != nil {
		return err
	}

	return tmpl.Execute(w, item)
}

func RenderGoTemplateList(list []interface{}, w io.Writer, tpl string) error {
	tmpl, err := template.New("result").Parse(tpl)
	if err != nil {
		return err
	}

	for _, item := range list {
		err := tmpl.Execute(w, item)
		if err != nil {
			return err
		}
	}

	return nil
}

func RenderPrettyList(obj ui.TableData, w io.Writer) error {
	ui.NL()
	ui.Table(obj, w)
	ui.NL()
	return nil
}

func ExecutionRenderer(ui *ui.UI, obj interface{}) error {
	test, ok := obj.(v1alpha1.ReconcileStatusAware)
	if !ok {
		return fmt.Errorf("can't use '%T' as testkube.Test in RenderObj for test", obj)
	}

	ui.Warn("Name:     ", test.GetName())
	ui.Warn("Namespace:", test.GetNamespace())
	ui.Warn("Created:  ", test.GetCreationTimestamp().String())
	ui.Warn("Status:  ", test.GetReconcileStatus().Phase.String())

	return nil
}

func RenderObject(obj interface{}) error {
	test, ok := obj.(v1alpha1.ReconcileStatusAware)
	if !ok {
		return errors.Errorf("can't use '%T' as v1alpha1.ReconcileStatusAware in RenderObj for test", obj)
	}

	ui.Warn("Name:     ", test.GetName())
	ui.Warn("Namespace:", test.GetNamespace())
	ui.Warn("Created:  ", test.GetCreationTimestamp().String())
	ui.Warn("Status:  ", test.GetReconcileStatus().Phase.String())

	/*
		if len(test.Labels) > 0 {
			ui.NL()
			ui.Warn("Labels:   ", testkube.MapToString(test.Labels))
		}
		if test.Schedule != "" {
			ui.NL()
			ui.Warn("Schedule: ", test.Schedule)
		}

		if test.Content != nil {
			ui.NL()
			ui.Info("Content")
			ui.Warn("Type", test.Content.Type_)
			if test.Content.Uri != "" {
				ui.Warn("Uri: ", test.Content.Uri)
			}

			if test.Content.Repository != nil {
				ui.Warn("Repository: ")
				ui.Warn("  Uri:      ", test.Content.Repository.Uri)
				ui.Warn("  Branch:   ", test.Content.Repository.Branch)
				ui.Warn("  Commit:   ", test.Content.Repository.Commit)
				ui.Warn("  Path:     ", test.Content.Repository.Path)
				ui.Warn("  Username: ", test.Content.Repository.Username)
				ui.Warn("  Token:    ", test.Content.Repository.Token)
			}

			if test.Content.Data != "" {
				ui.Warn("Data: ", "\n", test.Content.Data)
			}
		}

		if test.ExecutionRequest != nil {
			ui.Warn("Execution request: ")
			if test.ExecutionRequest.Name != "" {
				ui.Warn("  Name:        ", test.ExecutionRequest.Name)
			}

			if len(test.ExecutionRequest.Variables) > 0 {
				renderer.RenderVariables(test.ExecutionRequest.Variables)
			}

			if len(test.ExecutionRequest.Args) > 0 {
				ui.Warn("  Args:        ", test.ExecutionRequest.Args...)
			}

			if len(test.ExecutionRequest.Envs) > 0 {
				ui.NL()
				ui.Warn("  Envs:        ", testkube.MapToString(test.ExecutionRequest.Envs))
			}

			if len(test.ExecutionRequest.SecretEnvs) > 0 {
				ui.NL()
				ui.Warn("  Secret Envs: ", testkube.MapToString(test.ExecutionRequest.SecretEnvs))
			}

			if test.ExecutionRequest.HttpProxy != "" {
				ui.Warn("  Http proxy:  ", test.ExecutionRequest.HttpProxy)
			}

			if test.ExecutionRequest.HttpsProxy != "" {
				ui.Warn("  Https proxy: ", test.ExecutionRequest.HttpsProxy)
			}
		}

	*/

	return nil

}

func RenderList(cmd *cobra.Command, obj interface{}, w io.Writer) error {
	outputType := OutputType(cmd.Flag("output").Value.String())

	switch outputType {
	case OutputPretty:
		list, ok := obj.(ui.TableData)
		if !ok {
			return fmt.Errorf("can't render, need list of type ui.TableData but got: %T (%+v)", obj, obj)
		}
		return RenderPrettyList(list, w)
	case OutputYAML:
		return RenderYaml(obj, w)
	case OutputJSON:
		return RenderJSON(obj, w)
	case OutputGoTemplate:
		tpl := cmd.Flag("go-template").Value.String()
		list, ok := obj.([]interface{})
		if !ok {
			return fmt.Errorf("can't render, need list type but got: %+v", obj)
		}
		return RenderGoTemplateList(list, w, tpl)
	default:
		return RenderYaml(obj, w)
	}
}
