package structure

import (
	"fmt"
	"reflect"
	"strings"
)

func KeysFromMap(m interface{}) (keys []string) {
	v := reflect.ValueOf(m)
	if v.Kind() != reflect.Map {
		panic("input type not a map")
	}

	for _, k := range v.MapKeys() {
		keys = append(keys, k.String())
	}
	return keys
}

func MergeMap(src map[string]string, newer map[string]string) map[string]string {
	if len(src) == 0 {
		src = map[string]string{}
	}

	for key, value := range newer {
		src[key] = value
	}

	return src
}

// Label is the label field in metadata
type Label map[string]string

// String converts label to a string
func (l Label) String() string {
	var arr []string

	for k, v := range l {
		if len(k) == 0 {
			continue
		}

		arr = append(arr, fmt.Sprintf("%s=%s", k, v))
	}

	return strings.Join(arr, ",")
}
