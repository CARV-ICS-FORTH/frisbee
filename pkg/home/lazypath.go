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

package home

import (
	"os"
	"path/filepath"

	"github.com/carv-ics-forth/frisbee/pkg/home/xdg"
)

const (
	// CacheHomeEnvVar is the environment variable used by Frisbee
	// for the cache directory. When no value is set a default is used.
	CacheHomeEnvVar = "FRISBEE_CACHE_HOME"

	// ConfigHomeEnvVar is the environment variable used by Frisbee
	// for the config directory. When no value is set a default is used.
	ConfigHomeEnvVar = "FRISBEE_CONFIG_HOME"

	// DataHomeEnvVar is the environment variable used by Frisbee
	// for the data directory. When no value is set a default is used.
	DataHomeEnvVar = "FRISBEE_DATA_HOME"
)

// lazypath is a lazy-loaded path buffer for the XDG base directory specification.
type lazypath string

func (l lazypath) path(frisbeeEnvVar, xdgEnvVar string, defaultFn func() string, elem ...string) string {
	// There is an order to check for a path.
	// 1. See if a Frisbee specific environment variable has been set.
	// 2. Check if an XDG environment variable is set
	// 3. Fall back to a default
	base := os.Getenv(frisbeeEnvVar)
	if base != "" {
		return filepath.Join(base, filepath.Join(elem...))
	}
	base = os.Getenv(xdgEnvVar)
	if base == "" {
		base = defaultFn()
	}
	return filepath.Join(base, string(l), filepath.Join(elem...))
}

// cachePath defines the base directory relative to which user specific non-essential data files
// should be stored.
func (l lazypath) cachePath(elem ...string) string {
	return l.path(CacheHomeEnvVar, xdg.CacheHomeEnvVar, cacheHome, filepath.Join(elem...))
}

// configPath defines the base directory relative to which user specific configuration files should
// be stored.
func (l lazypath) configPath(elem ...string) string {
	return l.path(ConfigHomeEnvVar, xdg.ConfigHomeEnvVar, configHome, filepath.Join(elem...))
}

// dataPath defines the base directory relative to which user specific data files should be stored.
func (l lazypath) dataPath(elem ...string) string {
	return l.path(DataHomeEnvVar, xdg.DataHomeEnvVar, dataHome, filepath.Join(elem...))
}
