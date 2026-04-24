package config

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"

	"github.com/adrg/xdg"
	"github.com/goccy/go-yaml"
	"github.com/grafana/grafana-app-sdk/logging"
	"github.com/grafana/grafanactl/internal/format"
	"github.com/grafana/grafanactl/internal/secrets"
)

const (
	configFilePermissions  = 0o600
	StandardConfigFolder   = "grafanactl"
	StandardConfigFileName = "config.yaml"
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

	// Compute a length-preserving redacted copy of the file contents once.
	// Sensitive scalar values (fields tagged datapolicy:"secret") are replaced
	// with "**REDACTED**" padded to the same byte length. This copy is used
	// exclusively for error rendering so that goccy/go-yaml's Token.Origin
	// references redacted bytes rather than the raw secrets.
	redactedContents := secrets.RedactYAMLSecrets(contents, reflect.TypeFor[Context]())

	codec := &format.YAMLCodec{BytesAsBase64: true}
	if err := codec.Decode(bytes.NewBuffer(contents), &config); err != nil {
		// Re-parse with the redacted buffer to obtain an error whose Token
		// metadata references safe (redacted) source bytes. Because redaction
		// is length-preserving and does not alter YAML syntax, the redacted
		// decode hits the same structural error at the same position.
		var throwaway Config
		redactedErr := err
		if rerr := codec.Decode(bytes.NewBuffer(redactedContents), &throwaway); rerr != nil {
			redactedErr = rerr
		}
		return config, UnmarshalError{File: filename, Err: redactedErr}
	}

	for name, ctx := range config.Contexts {
		ctx.Name = name
	}

	for _, override := range overrides {
		if err := override(&config); err != nil {
			return config, annotateErrorWithSource(filename, redactedContents, err)
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

// annotateErrorWithSource wraps a ValidationError with a source snippet from
// the config file. redactedContents must be the length-preserving redacted
// copy of the file so that the annotated excerpt never reveals sensitive
// scalar values (fields tagged datapolicy:"secret").
func annotateErrorWithSource(filename string, redactedContents []byte, err error) error {
	if err == nil {
		return nil
	}

	validationError := ValidationError{}
	if errors.As(err, &validationError) {
		path, err := yaml.PathString(validationError.Path)
		if err != nil {
			return err
		}

		// The **REDACTED** sentinel starts with '*', which goccy/go-yaml's raw
		// YAML parser (used internally by path.AnnotateSource via
		// parser.ParseBytes) treats as an alias indicator and fails to resolve.
		// Replace with the YAML-safe equivalent of the same byte length so
		// that AnnotateSource can parse the source without errors while still
		// hiding the sensitive value.
		annotationContents := bytes.ReplaceAll(
			redactedContents,
			[]byte("**REDACTED**"),
			[]byte("__REDACTED__"),
		)
		annotatedSource, err := path.AnnotateSource(annotationContents, true)
		if err != nil {
			return err
		}

		validationError.File = filename
		validationError.AnnotatedSource = string(annotatedSource)

		return validationError
	}

	return err
}
