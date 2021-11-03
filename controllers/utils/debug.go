/*
Copyright 2021 ICS-FORTH.

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
	"fmt"
	"path"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"
)

func GetCallerInfo(skip int) (fileName, funcName string, line int) {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		logrus.Warn("get info failed")

		return
	}

	fileName = path.Base(file)
	funcName = runtime.FuncForPC(pc).Name()

	return
}

func GetCallerLine() string {
	fileName, funcName, line := GetCallerInfo(3)

	_ = fileName

	return fmt.Sprintf("%s:%d", filepath.Base(funcName), line)
}
