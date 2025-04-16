package config

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/goccy/go-yaml"
	"github.com/grafana/grafana-app-sdk/logging"
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

func Load(ctx context.Context, source Source, overrides ...Override) (Config, error) {
	config := Config{}

	filename, err := source()
	if err != nil {
		return config, err
	}

	logging.FromContext(ctx).Debug("Loading config", slog.String("filename", filename))
	config.Source = filename

	contents, err := os.ReadFile(filename)
	if err != nil {
		return config, err
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
		return config, UnmarshalError{File: filename, Err: err}
	}

	for name, ctx := range config.Contexts {
		ctx.Name = name
	}

	for _, override := range overrides {
		if err := override(&config); err != nil {
			return config, annotateErrorWithSource(filename, contents, err)
		}
	}

	return config, nil
}

func Write(ctx context.Context, source Source, cfg Config) error {
	filename, err := source()
	if err != nil {
		return err
	}

	logging.FromContext(ctx).Debug("Writing config", slog.String("filename", filename))

	marshaled, err := yaml.MarshalWithOptions(
		cfg,
		yaml.Indent(2),
		yaml.IndentSequence(true),
		yaml.CustomMarshaler[[]byte](func(data []byte) ([]byte, error) {
			dst := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
			base64.StdEncoding.Encode(dst, data)

			return dst, nil
		}),
	)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, marshaled, configFilePermissions)
}

func annotateErrorWithSource(filename string, contents []byte, err error) error {
	if err == nil {
		return nil
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

		validationError.File = filename
		validationError.AnnotatedSource = string(annotatedSource)

		return validationError
	}

	return err
}
