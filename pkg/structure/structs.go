package structure

import (
	"reflect"
	"strings"
)

func structToLowercase(in interface{}) map[string]interface{} {
	v := reflect.ValueOf(in)
	if v.Kind() != reflect.Struct {
		return nil
	}

	vType := v.Type()

	result := make(map[string]interface{}, v.NumField())

	for i := 0; i < v.NumField(); i++ {
		name := vType.Field(i).Name
		result[strings.ToLower(name)] = v.Field(i).Interface()
	}

	return result
}

func lower(f interface{}) interface{} {
	switch f := f.(type) {
	case []interface{}:
		for i := range f {
			f[i] = lower(f[i])
		}
		return f
	case map[string]interface{}:
		lf := make(map[string]interface{}, len(f))
		for k, v := range f {
			lf[strings.ToLower(k)] = lower(v)
		}
		return lf
	default:
		return f
	}
}
