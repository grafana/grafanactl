package config

import (
	"errors"
)

// Config holds the information needed to connect to remote Grafana instances.
type Config struct {
	// Contexts is a map of context configurations, indexed by name.
	Contexts map[string]*Context `json:"contexts" yaml:"contexts"`

	// CurrentContext is the name of the context currently in use.
	CurrentContext string `json:"current-context" yaml:"current-context"`
}

// Context holds the information required to connect to a remote Grafana instance.
type Context struct {
	Name string `json:"-" yaml:"-"`

	Grafana *GrafanaConfig `json:"grafana,omitempty" yaml:"grafana,omitempty"`
}

type GrafanaConfig struct {
	// Server is the address of the Grafana server (https://hostname:port/path).
	Server string `env:"GRAFANA_SERVER" json:"server,omitempty" yaml:"server,omitempty"`

	User  string `env:"GRAFANA_USER" json:"user,omitempty" yaml:"user,omitempty"`
	Token string `datapolicy:"secret" env:"GRAFANA_TOKEN" json:"token,omitempty" yaml:"token,omitempty"`

	// InsecureSkipTLSVerify disables the validation of the server's SSL certificate.
	// Enabling this will make your HTTPS connections insecure.
	InsecureSkipTLSVerify bool `json:"insecure-skip-tls-verify,omitempty" yaml:"insecure-skip-tls-verify,omitempty"`
}

func (grafana GrafanaConfig) IsEmpty() bool {
	return grafana == GrafanaConfig{}
}

// Minify returns a trimmed down version of the given configuration containing
// only the current context and the relevant options it directly depends on.
func Minify(config Config) (Config, error) {
	minified := config

	if config.CurrentContext == "" {
		return Config{}, errors.New("current-context must be defined in order to minify")
	}

	minified.Contexts = make(map[string]*Context, 1)
	for _, ctx := range config.Contexts {
		if ctx.Name == minified.CurrentContext {
			minified.Contexts[ctx.Name] = ctx
		}
	}

	return minified, nil
}
