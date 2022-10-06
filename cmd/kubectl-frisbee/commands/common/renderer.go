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
	"io"
	"text/template"

	"github.com/carv-ics-forth/frisbee/cmd/kubectl-frisbee/env"
	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/util/json"
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

func RenderList(obj interface{}, w io.Writer) error {
	switch OutputType(env.Default.OutputType) {
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
		tpl := env.Default.GoTemplate

		list, ok := obj.([]interface{})
		if !ok {
			return fmt.Errorf("can't render, need list type but got: %+v", obj)
		}
		return RenderGoTemplateList(list, w, tpl)
	default:
		return RenderYaml(obj, w)
	}
}
