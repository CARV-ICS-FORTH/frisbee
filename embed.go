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

/*
Package embed is used to embed the various required scripts into the Frisbee Terminal.
This allows to execute the Terminal from any path. For more info see https://zetcode.com/golang/embed/
*/
package embed

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

//go:embed hack
var Hack embed.FS

// UpdateLocalFiles duplicates the structure of embedded fs into the installation dir.
func UpdateLocalFiles(static embed.FS, installationDir string) error {
	root := "."

	copyLocally := func(path string) error {
		data, err := fs.ReadFile(static, path)
		if err != nil {
			return errors.Wrapf(err, "cannot read embedded file '%s'", path)
		}

		return os.WriteFile(filepath.Join(installationDir, path), data, os.ModePerm)
	}

	return fs.WalkDir(static, root, func(path string, d fs.DirEntry, _ error) error {
		if path == root {
			// ignore the root
			return nil
		}

		/*
			Open and inspect embedded file.
		*/
		f, err := static.Open(path)
		if err != nil {
			return errors.Wrapf(err, "cannot open embedded file '%s'", path)
		}

		fInfo, err := f.Stat()
		if err != nil {
			return errors.Wrapf(err, "cannot stat embedded file '%s'", path)
		}

		/*
			Duplicate the embedded file into installation dir.
		*/
		switch {
		case fInfo.Mode().IsRegular():
			localInfo, err := os.Stat(path)

			if err != nil && os.IsNotExist(err) {
				return errors.Wrapf(err, "cannot stat installation path '%s'", path)
			}

			if !localInfo.Mode().IsRegular() {
				return errors.Errorf("expected '%s' to be a file, but it's '%s'.", path, localInfo.Mode().Type())
			}

			// Copy the file locally
			return copyLocally(path)

		case fInfo.IsDir():
			ufInfo, err := os.Stat(path)
			switch {
			case os.IsNotExist(err):
				err := os.MkdirAll(filepath.Join(installationDir, path), os.ModePerm)
				return errors.Wrapf(err, "cannot create dir '%s' in the installation fs", path)
			case err != nil:
				return errors.Wrapf(err, "cannot stat installation path '%s'", path)
			case !ufInfo.IsDir():
				return errors.Errorf("Expected '%s' to be a dir, but it's '%s'.", path, ufInfo.Mode().Type())
			default:
				return nil
			}
		default:
			return errors.Errorf("Filemode '%s' is not supported", fInfo.Mode().Type())
		}
	})
}
