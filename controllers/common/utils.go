package common

import (
	"fmt"
	"reflect"
)

// Reflect if an interface is either a struct or a pointer to a struct
// and has the defined member method. If error is nil, it means
// the MethodName is accessible with reflect.
func ReflectStructMethod(Iface interface{}, MethodName string) error {
	ValueIface := reflect.ValueOf(Iface)

	// Check if the passed interface is a pointer
	if ValueIface.Type().Kind() != reflect.Ptr {
		// Create a new type of Iface, so we have a pointer to work with
		ValueIface = reflect.New(reflect.TypeOf(Iface))
	}

	// Get the method by name
	Method := ValueIface.MethodByName(MethodName)
	if !Method.IsValid() {
		return fmt.Errorf("Couldn't find method `%s` in interface `%s`, is it Exported?", MethodName, ValueIface.Type())
	}
	return nil
}

// Reflect if an interface is either a struct or a pointer to a struct
// and has the defined member field, if error is nil, the given
// FieldName exists and is accessible with reflect.
func ReflectStructField(Iface interface{}, FieldName string) error {
	ValueIface := reflect.ValueOf(Iface)

	// Check if the passed interface is a pointer
	if ValueIface.Type().Kind() != reflect.Ptr {
		// Create a new type of Iface's Type, so we have a pointer to work with
		ValueIface = reflect.New(reflect.TypeOf(Iface))
	}

	// 'dereference' with Elem() and get the field by name
	Field := ValueIface.Elem().FieldByName(FieldName)
	if !Field.IsValid() {
		return fmt.Errorf("Interface `%s` does not have the field `%s`", ValueIface.Type(), FieldName)
	}
	return nil
}
