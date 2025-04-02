package config

import (
	"errors"
	"fmt"
)

const (
	// DefaultContextName is the name of the default context.
	DefaultContextName = "default"
)

// Config holds the information needed to connect to remote Grafana instances.
type Config struct {
	// Contexts is a map of context configurations, indexed by name.
	Contexts map[string]*Context `json:"contexts" yaml:"contexts"`

	// CurrentContext is the name of the context currently in use.
	CurrentContext string `json:"current-context" yaml:"current-context"`
}

func (config *Config) HasContext(name string) bool {
	return config.Contexts[name] != nil
}

// GetCurrentContext returns the current context.
// If the current context is not set, it returns an error.
func (config *Config) GetCurrentContext() *Context {
	return config.Contexts[config.CurrentContext]
}

// SetContext adds a new context to the Grafana config.
// If a context with the same name already exists, it is overwritten.
func (config *Config) SetContext(name string, makeCurrent bool, context Context) {
	if config.Contexts == nil {
		config.Contexts = make(map[string]*Context)
	}

	config.Contexts[name] = &context

	if makeCurrent {
		config.CurrentContext = name
	}
}

// Context holds the information required to connect to a remote Grafana instance.
type Context struct {
	Name string `json:"-" yaml:"-"`

	Grafana *GrafanaConfig `json:"grafana,omitempty" yaml:"grafana,omitempty"`
}

func (context *Context) Validate() error {
	if context.Grafana == nil || context.Grafana.IsEmpty() {
		return ValidationError{
			Path:    fmt.Sprintf("$.contexts.'%s'", context.Name),
			Message: "grafana config is required",
		}
	}

	return context.Grafana.Validate(context.Name)
}

type GrafanaConfig struct {
	// Server is the address of the Grafana server (https://hostname:port/path).
	// Required.
	Server string `env:"GRAFANA_SERVER" json:"server,omitempty" yaml:"server,omitempty"`

	// User to authenticate as with basic authentication.
	// Optional.
	User string `env:"GRAFANA_USER" json:"user,omitempty" yaml:"user,omitempty"`
	// Password to use when using with basic authentication.
	// Optional.
	Password string `datapolicy:"secret" env:"GRAFANA_PASSWORD" json:"password,omitempty" yaml:"password,omitempty"`

	// APIToken is a service account token.
	// See https://grafana.com/docs/grafana/latest/administration/service-accounts/#add-a-token-to-a-service-account-in-grafana
	// Note: if defined, the API Token takes precedence over basic auth credentials.
	// Optional.
	APIToken string `datapolicy:"secret" env:"GRAFANA_TOKEN" json:"token,omitempty" yaml:"token,omitempty"`

	// OrgID specifies the organization targeted by this config.
	// Note: only used when relying on basic authentication as API keys are
	// already org-scoped.
	// Default: 1
	// Optional.
	OrgID int64 `env:"GRAFANA_ORG_ID" json:"org_id,omitempty" yaml:"org_id,omitempty"`

	// StackID specifies the Grafana Cloud stack targeted by this config.
	// Note: required when targeting a Grafana Cloud instance.
	StackID int64 `env:"GRAFANA_STACK_ID" json:"stack_id,omitempty" yaml:"stack_id,omitempty"`

	// InsecureSkipTLSVerify disables the validation of the server's SSL certificate.
	// Enabling this will make your HTTPS connections insecure.
	InsecureSkipTLSVerify bool `json:"insecure-skip-tls-verify,omitempty" yaml:"insecure-skip-tls-verify,omitempty"`
}

func (grafana GrafanaConfig) Validate(contextName string) error {
	if grafana.Server == "" {
		return ValidationError{
			Path:    fmt.Sprintf("$.contexts.'%s'.grafana", contextName),
			Message: "server is required",
			Suggestions: []string{
				"Set the address of the Grafana server to connect to",
			},
		}
	}

	// Auth: API Token
	if grafana.APIToken != "" {
		if grafana.OrgID != 0 {
			return ValidationError{
				Path:    fmt.Sprintf("$.contexts.'%s'.grafana.org_id", contextName),
				Message: "org_id is only supported with basic auth. API keys are already org-scoped",
				Suggestions: []string{
					"Remove the `org_id` setting",
				},
			}
		}
	}

	return nil
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
	for name, ctx := range config.Contexts {
		if name == minified.CurrentContext {
			minified.Contexts[name] = ctx
		}
	}

	return minified, nil
}
