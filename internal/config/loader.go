package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/goccy/go-yaml"
	"github.com/grafana/grafanactl/internal/fail"
)

const (
	configFilePermissions  = 0o600
	StandardConfigFolder   = "grafanactl"
	StandardConfigFileName = "config.yaml"
)

type Override func(cfg *Config) error

type Source func() (string, error)

func ExplicitConfigFile(path string) Source {
	return func() (string, error) {
		return path, nil
	}
}

func StandardLocation() Source {
	return func() (string, error) {
		file, err := xdg.ConfigFile(filepath.Join(StandardConfigFolder, StandardConfigFileName))
		if err != nil {
			return "", err
		}

		_, err = os.Stat(file)
		// Create an empty config file, to ensure that the loader won't fail.
		if os.IsNotExist(err) {
			if createErr := os.WriteFile(file, []byte(""), configFilePermissions); createErr != nil {
				return "", createErr
			}
		} else if err != nil {
			return "", err
		}

		return file, nil
	}
}

func Load(source Source, overrides ...Override) (Config, error) {
	config := Config{}

	filename, err := source()
	if err != nil {
		return config, handleReadError(filename, nil, err)
	}

	contents, err := os.ReadFile(filename)
	if err != nil {
		return config, handleReadError(filename, contents, err)
	}

	err = yaml.UnmarshalWithOptions(contents, &config, yaml.Strict(), yaml.CustomUnmarshaler[[]byte](func(dest *[]byte, raw []byte) error {
		dst := make([]byte, base64.StdEncoding.DecodedLen(len(raw)))
		_, err := base64.StdEncoding.Decode(dst, raw)
		if err != nil {
			return err
		}

		*dest = dst

		return nil
	}))
	if err != nil {
		return config, UnmarshalError(filename, err)
	}

	for name, ctx := range config.Contexts {
		ctx.Name = name
	}

	for _, override := range overrides {
		if err := override(&config); err != nil {
			return config, handleReadError(filename, contents, err)
		}
	}

	return config, nil
}

func Write(source Source, cfg Config) error {
	filename, err := source()
	if err != nil {
		return handleWriteError(err)
	}

	marshaled, err := yaml.MarshalWithOptions(cfg, yaml.Indent(2), yaml.CustomMarshaler[[]byte](func(data []byte) ([]byte, error) {
		dst := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
		base64.StdEncoding.Encode(dst, data)

		return dst, nil
	}))
	if err != nil {
		return handleWriteError(err)
	}

	return handleWriteError(os.WriteFile(filename, marshaled, configFilePermissions))
}

func handleReadError(filename string, contents []byte, err error) error {
	if err == nil {
		return nil
	}

	pathErr := &fs.PathError{}

	if errors.Is(err, os.ErrNotExist) && errors.As(err, &pathErr) {
		return fail.DetailedError{
			Parent:  err,
			Summary: "File not found",
			Details: fmt.Sprintf("could not read '%s'", pathErr.Path),
			Suggestions: []string{
				"Check for typos in the command's arguments",
			},
		}
	}

	if errors.Is(err, os.ErrPermission) && errors.As(err, &pathErr) {
		return fail.DetailedError{
			Parent:  err,
			Summary: "Permission denied",
			Details: fmt.Sprintf("could not read '%s'", pathErr.Path),
			Suggestions: []string{
				"Check that the configuration file is readable by the current user",
				fmt.Sprintf("On Linux/macOS: chmod %o %s", configFilePermissions, pathErr.Path),
			},
		}
	}

	validationError := ValidationError{}
	if errors.As(err, &validationError) {
		path, err := yaml.PathString(validationError.Path)
		if err != nil {
			return err
		}

		annotatedSource, err := path.AnnotateSource(contents, true)
		if err != nil {
			return err
		}

		detailedErr := InvalidConfiguration(filename, validationError.Error(), string(annotatedSource))
		detailedErr.Suggestions = append(detailedErr.Suggestions, validationError.Suggestions...)

		return detailedErr
	}

	return err
}

func handleWriteError(err error) error {
	if err == nil {
		return nil
	}

	pathErr := &fs.PathError{}

	if errors.Is(err, os.ErrNotExist) && errors.As(err, &pathErr) {
		return fail.DetailedError{
			Parent:  err,
			Summary: "File not found",
			Details: fmt.Sprintf("could not write '%s'", pathErr.Path),
			Suggestions: []string{
				"Check for typos in the command's arguments",
			},
		}
	}

	if errors.Is(err, os.ErrPermission) && errors.As(err, &pathErr) {
		return fail.DetailedError{
			Parent:  err,
			Summary: "Permission denied",
			Details: fmt.Sprintf("could not write '%s'", pathErr.Path),
			Suggestions: []string{
				"Check that the configuration file is writable by the current user",
				fmt.Sprintf("On Linux/macOS: chmod %o %s", configFilePermissions, pathErr.Path),
			},
		}
	}

	return err
}
