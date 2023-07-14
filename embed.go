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
func UpdateLocalFiles(embeddedFS embed.FS, installationDir string) error {
	root := "."

	copyLocally := func(sourceFilePath string, hostFilePath string) error {
		data, err := fs.ReadFile(embeddedFS, sourceFilePath)
		if err != nil {
			return errors.Wrapf(err, "cannot read embeddedFS file '%s'", sourceFilePath)
		}

		if err := os.WriteFile(hostFilePath, data, os.ModePerm); err != nil {
			return errors.Wrapf(err, "cannot write file '%s'", hostFilePath)
		}

		return nil
	}

	return fs.WalkDir(embeddedFS, root, func(relPath string, d fs.DirEntry, _ error) error {
		if relPath == root {
			// ignore the root
			return nil
		}

		/*---------------------------------------------------
		 * Open and inspect embedded file.
		 *---------------------------------------------------*/
		embeddedFile, err := embeddedFS.Open(relPath)
		if err != nil {
			return errors.Wrapf(err, "cannot open embeddedFS file '%s'", relPath)
		}

		embeddedFileInfo, err := embeddedFile.Stat()
		if err != nil {
			return errors.Wrapf(err, "cannot stat embeddedFS file '%s'", relPath)
		}

		/*---------------------------------------------------
		 * Duplicate the embedded file into installation dir.
		 *---------------------------------------------------*/
		hostpath := filepath.Join(installationDir, relPath)

		switch {
		case embeddedFileInfo.Mode().IsRegular():
			hostFileInfo, err := os.Stat(hostpath)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					// Copy the file locally
					return copyLocally(relPath, hostpath)
				}

				return errors.Wrapf(err, "cannot stat host path '%s'", hostpath)
			}

			if !hostFileInfo.Mode().IsRegular() {
				return errors.Errorf("expected '%s' to be a file, but it's '%s'.", relPath, hostFileInfo.Mode().Type())
			}

			// Copy the file locally
			return copyLocally(relPath, hostpath)

		case embeddedFileInfo.IsDir():
			hostFileInfo, err := os.Stat(hostpath)
			switch {
			case os.IsNotExist(err):
				if err := os.MkdirAll(hostpath, os.ModePerm); err != nil {
					return errors.Wrapf(err, "cannot create dir '%s' in the host fs", hostpath)
				}

				return nil
			case err != nil:
				return errors.Wrapf(err, "cannot stat host path '%s'", hostpath)
			case !hostFileInfo.IsDir():
				return errors.Errorf("expected '%s' to be a dir, but it's '%s'", hostpath, hostFileInfo.Mode().Type())
			default:
				return nil
			}
		default:
			return errors.Errorf("Filemode '%s' is not supported", embeddedFileInfo.Mode().Type())
		}
	})
}
