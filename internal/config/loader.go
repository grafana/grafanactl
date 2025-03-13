package config

import (
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
		return config, handleReadError(err)
	}

	contents, err := os.ReadFile(filename)
	if err != nil {
		return config, handleReadError(err)
	}

	if err := yaml.UnmarshalWithOptions(contents, &config, yaml.DisallowUnknownField()); err != nil {
		return config, InvalidConfiguration(filename, err)
	}

	for name, ctx := range config.Contexts {
		ctx.Name = name
	}

	for _, override := range overrides {
		if err := override(&config); err != nil {
			return config, err
		}
	}

	return config, nil
}

func Write(source Source, cfg Config) error {
	filename, err := source()
	if err != nil {
		return handleWriteError(err)
	}

	marshaled, err := yaml.MarshalWithOptions(cfg, yaml.Indent(2))
	if err != nil {
		return handleWriteError(err)
	}

	return handleWriteError(os.WriteFile(filename, marshaled, configFilePermissions))
}

func handleReadError(err error) error {
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
