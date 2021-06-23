package load

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Config describes external configurations required by the object service.
type Config struct {
	// File is the name of configuration file
	File string
	// DstDir is the directory where File will be written
	DstDir string
	// Payload is the content that will be written in File
	Payload []byte
}

// ConfigDir scans the given directory for "/config" dir and loads all into contents
// into the Frisbee configuration format.
// Any variable is replaced by the MapperFunc.
func ConfigDir(path string) []Config {
	var configs []Config

	dir := filepath.Join(path, "/config")
	err := filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			data, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			configs = append(configs, Config{
				File:    info.Name(),
				DstDir:  strings.TrimPrefix(path, dir),
				Payload: data, // os.Expand(string(data), load.MapperFunc),
			})
			return nil
		})
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		panic(err)
	}

	return configs
}
