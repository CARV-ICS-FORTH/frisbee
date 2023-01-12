/*
Copyright 2022-2023 ICS-FORTH.

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

// Package home calculates filesystem paths to Frisbee's configuration, cache and data.
package home

// This helper builds paths to Frisbee's configuration, cache and data paths.
const lp = lazypath("frisbee")

// ConfigPath returns the path where Frisbee stores configuration.
func ConfigPath(elem ...string) string { return lp.configPath(elem...) }

// CachePath returns the path where Frisbee stores cached objects.
func CachePath(elem ...string) string { return lp.cachePath(elem...) }

// DataPath returns the path where Frisbee stores data.
func DataPath(elem ...string) string { return lp.dataPath(elem...) }

// CacheIndexFile returns the path to an index for the given named repository.
func CacheIndexFile(name string) string {
	if name != "" {
		name += "-"
	}
	return name + "index.yaml"
}

// CacheChartsFile returns the path to a text file listing all the charts
// within the given named repository.
func CacheChartsFile(name string) string {
	if name != "" {
		name += "-"
	}
	return name + "charts.txt"
}
