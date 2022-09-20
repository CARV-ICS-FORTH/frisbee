/*
Copyright 2022 ICS-FORTH.

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

package manifest

import (
	"bufio"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

// ReadManifest reads from stdin, a single file/url, or a list of files and/or urls.
func ReadManifest(manifestPaths ...string) ([][]byte, error) {
	var manifestContents [][]byte
	var err error
	if len(manifestPaths) == 1 && manifestPaths[0] == "-" {
		body, err := ReadFromStdin()
		if err != nil {
			return [][]byte{}, err
		}
		manifestContents = append(manifestContents, body)
	} else {
		manifestContents, err = ReadFromFilePathsOrUrls(manifestPaths...)
		if err != nil {
			return [][]byte{}, err
		}
	}
	return manifestContents, err
}

// ReadFromStdin reads the manifest from standard input.
func ReadFromStdin() ([]byte, error) {
	reader := bufio.NewReader(os.Stdin)
	body, err := io.ReadAll(reader)
	if err != nil {
		return []byte{}, err
	}

	return body, err
}

// ReadFromFilePathsOrUrls reads the content of a single or a list of file paths and/or urls.
func ReadFromFilePathsOrUrls(filePathsOrUrls ...string) ([][]byte, error) {
	var fileContents [][]byte
	var body []byte
	var err error
	for _, filePathOrUrl := range filePathsOrUrls {
		if IsURL(filePathOrUrl) {
			body, err = ReadFromUrl(filePathOrUrl)
			if err != nil {
				return [][]byte{}, err
			}
		} else {
			body, err = os.ReadFile(filepath.Clean(filePathOrUrl))
			if err != nil {
				return [][]byte{}, err
			}
		}

		fileContents = append(fileContents, body)
	}

	return fileContents, err
}

// ReadFromUrl reads the content of a URL.
func ReadFromUrl(url string) ([]byte, error) {
	response, err := http.Get(url) //nolint:gosec
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil {
		return nil, err
	}

	return body, err
}

// IsURL returns whether a string is a URL.
func IsURL(u string) bool {
	var parsedURL *url.URL
	var err error

	parsedURL, err = url.ParseRequestURI(u)
	if err == nil {
		if parsedURL != nil && parsedURL.Host != "" {
			return true
		}
	}
	return false
}
