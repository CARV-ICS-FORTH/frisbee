package structure

import (
	"reflect"
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

func MergeMap(src, newer map[string]string) map[string]string {
	if len(src) == 0 {
		src = map[string]string{}
	}

	for key, value := range newer {
		src[key] = value
	}

	return src
}

func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func Intersect(as, bs []string) []string {
	i := make([]string, 0, Max(len(as), len(bs)))
	for _, a := range as {
		for _, b := range bs {
			if a == b {
				i = append(i, a)
			}
		}
	}
	return i
}

// Contains checks if all keys of the subset are included in the ref, and the values are the same.
func Contains(ref, subset map[string]string) bool {
	if len(ref) < len(subset) {
		return false
	}

	for k, v := range subset {
		if w, ok := ref[k]; !ok || v != w {
			return false
		}
	}

	return true
}
