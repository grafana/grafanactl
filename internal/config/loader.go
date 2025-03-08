package config

import (
	"os"

	"github.com/adrg/xdg"
	"github.com/goccy/go-yaml"
)

const configFilePermissions = 0o600

type Override func(cfg *Config) error

type Source func() (string, error)

func ExplicitConfigFile(path string) Source {
	return func() (string, error) {
		return path, nil
	}
}

func StandardLocation() Source {
	return func() (string, error) {
		file, err := xdg.ConfigFile("grafana/config.yaml")
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
		return config, err
	}

	contents, err := os.ReadFile(filename)
	if err != nil {
		return config, err
	}

	if err := yaml.UnmarshalWithOptions(contents, &config, yaml.DisallowUnknownField()); err != nil {
		return config, err
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
		return err
	}

	marshaled, err := yaml.MarshalWithOptions(cfg, yaml.Indent(2))
	if err != nil {
		return err
	}

	return os.WriteFile(filename, marshaled, configFilePermissions)
}
