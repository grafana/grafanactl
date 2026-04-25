package config

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/goccy/go-yaml"
	"github.com/grafana/grafana-app-sdk/logging"
	"github.com/grafana/grafanactl/internal/format"
)

const (
	configFilePermissions  = 0o600
	StandardConfigFolder   = "grafanactl"
	StandardConfigFileName = "config.yaml"
	tokenCacheFileName     = "tokens.yaml"
	ConfigFileEnvVar       = "GRAFANACTL_CONFIG"

	defaultEmptyConfigFile = `
contexts:
  default: {}
current-context: default
`
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
		// Check if GRAFANACTL_CONFIG environment variable is set
		if envPath := os.Getenv(ConfigFileEnvVar); envPath != "" {
			return envPath, nil
		}

		file, err := xdg.ConfigFile(filepath.Join(StandardConfigFolder, StandardConfigFileName))
		if err != nil {
			return "", err
		}

		_, err = os.Stat(file)
		// Create an empty config file, to ensure that the loader won't fail.
		if os.IsNotExist(err) {
			if createErr := os.WriteFile(file, []byte(defaultEmptyConfigFile), configFilePermissions); createErr != nil {
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

	codec := &format.YAMLCodec{BytesAsBase64: true}
	if err := codec.Decode(bytes.NewBuffer(contents), &config); err != nil {
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

	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, configFilePermissions)
	if err != nil {
		return err
	}
	defer file.Close()

	codec := &format.YAMLCodec{BytesAsBase64: true}
	return codec.Encode(file, cfg)
}

func tokenCachePath() (string, error) {
	return xdg.CacheFile(filepath.Join(StandardConfigFolder, tokenCacheFileName))
}

// LoadTokenCache loads the token cache from $XDG_CACHE_HOME/grafanactl/tokens.yaml.
// Returns an empty cache if the file doesn't exist.
func LoadTokenCache(ctx context.Context) (TokenCache, error) {
	cache := TokenCache{}

	cachePath, err := tokenCachePath()
	if err != nil {
		return cache, err
	}

	logging.FromContext(ctx).Debug("Loading token cache", slog.String("filename", cachePath))

	contents, err := os.ReadFile(cachePath)
	if os.IsNotExist(err) {
		return cache, nil
	}
	if err != nil {
		return cache, err
	}

	codec := &format.YAMLCodec{}
	if err := codec.Decode(bytes.NewBuffer(contents), &cache); err != nil {
		return cache, err
	}

	return cache, nil
}

// WriteTokenCache writes the token cache to $XDG_CACHE_HOME/grafanactl/tokens.yaml.
func WriteTokenCache(ctx context.Context, cache TokenCache) error {
	cachePath, err := tokenCachePath()
	if err != nil {
		return err
	}

	logging.FromContext(ctx).Debug("Writing token cache", slog.String("filename", cachePath))

	file, err := os.OpenFile(cachePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, configFilePermissions)
	if err != nil {
		return err
	}
	defer file.Close()

	codec := &format.YAMLCodec{}
	return codec.Encode(file, cache)
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
