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

package utils

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func index(path reflect.Value, idx string) reflect.Value {
	if i, err := strconv.Atoi(idx); err == nil {
		return path.Index(i)
	}

	// reflect.Value.FieldByName cannot be used on map Value
	if path.Kind() == reflect.Map {
		return reflect.Indirect(path)
	}

	return reflect.Indirect(path).FieldByName(idx)
}

func SetField(service *v1alpha1.Service, val v1alpha1.SetField) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("cannot set field [%s]. err: %s", val.Field, r)
		}
	}()

	fieldRef := reflect.ValueOf(&service.Spec).Elem()

	for _, s := range strings.Split(val.Field, ".") {
		fieldRef = index(fieldRef, s)
	}

	var conv interface{}

	// Convert src value to something that may fit to the dst.
	switch fieldRef.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		toInt, err := strconv.Atoi(val.Value)
		if err != nil {
			return errors.Wrapf(err, "convert to Int error")
		}

		conv = toInt

	case reflect.Bool:
		toBool, err := strconv.ParseBool(val.Value)
		if err != nil {
			return errors.Wrapf(err, "convert to Bool error")
		}

		conv = toBool

	case reflect.Map:
		// TODO: Needs to be improved because the map can be of various types
		logrus.Warn("THIS FUNCTION IS NOT WORKING, BUT WE DO NOT WANT TO FAIL EITHER")

		return nil

	default:
		conv = val.Value
	}

	fieldRef.Set(reflect.ValueOf(conv).Convert(fieldRef.Type()))

	return nil
}
