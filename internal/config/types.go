package config

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
)

const (
	// DefaultContextName is the name of the default context.
	DefaultContextName = "default"
)

// Config holds the information needed to connect to remote Grafana instances.
type Config struct {
	// Source contains the path to the config file parsed to populate this struct.
	Source string `json:"-" yaml:"-"`

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

// ToRESTConfig returns a REST config for the context.
func (context *Context) ToRESTConfig(ctx context.Context) NamespacedRESTConfig {
	return NewNamespacedRESTConfig(ctx, *context)
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
	// Note: required when targeting an on-prem Grafana instance.
	// See StackID for Grafana Cloud instances.
	OrgID int64 `env:"GRAFANA_ORG_ID" json:"org-id,omitempty" yaml:"org-id,omitempty"`

	// StackID specifies the Grafana Cloud stack targeted by this config.
	// Note: required when targeting a Grafana Cloud instance.
	// See OrgID for on-prem Grafana instances.
	StackID int64 `env:"GRAFANA_STACK_ID" json:"stack-id,omitempty" yaml:"stack-id,omitempty"`

	// TLS contains TLS-related configuration settings.
	TLS *TLS `json:"tls,omitempty" yaml:"tls,omitempty"`

	// OIDC contains OpenID Connect authentication settings.
	// When configured with valid tokens, OIDC takes precedence over APIToken and basic auth.
	OIDC *OIDCConfig `json:"oidc,omitempty" yaml:"oidc,omitempty"`
}

func (grafana GrafanaConfig) validateNamespace(contextName string) error {
	if grafana.OrgID != 0 {
		return nil
	}

	discoveredStackID, discoveryErr := DiscoverStackID(context.Background(), grafana)

	if grafana.StackID == 0 {
		if discoveryErr != nil {
			return ValidationError{
				Path:    fmt.Sprintf("$.contexts.'%s'.grafana", contextName),
				Message: fmt.Sprintf("missing contexts.%[1]s.org-id or contexts.%[1]s.stack-id", contextName),
				Suggestions: []string{
					"Specify the Grafana Org ID for on-prem Grafana",
					"Specify the Grafana Cloud Stack ID for Grafana Cloud",
				},
			}
		}

		return nil
	}

	// If discovery failed but grafana.StackID is set, we proceed with the configured StackID
	//nolint:nilerr // We intentionally ignore the error when StackID is configured
	if discoveryErr != nil {
		return nil
	}

	if discoveredStackID != grafana.StackID {
		return ValidationError{
			Path:    fmt.Sprintf("$.contexts.'%s'.grafana", contextName),
			Message: fmt.Sprintf("mismatched contexts.%[1]s.stack-id, discovered %d - was %d in config", contextName, discoveredStackID, grafana.StackID),
			Suggestions: []string{
				"Specify the correct Grafana Cloud Stack ID for Grafana Cloud or omit the stack-id param",
			},
		}
	}

	return nil
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

	if err := grafana.validateNamespace(contextName); err != nil {
		return err
	}

	return nil
}

func (grafana GrafanaConfig) IsEmpty() bool {
	return grafana == GrafanaConfig{}
}

// OIDCConfig contains OpenID Connect provider settings.
type OIDCConfig struct {
	// IssuerURL is the OIDC provider's issuer URL (e.g., https://your-org.okta.com).
	// Used to discover authorization and token endpoints via .well-known/openid-configuration.
	IssuerURL string `env:"GRAFANA_OIDC_ISSUER" json:"issuer-url,omitempty" yaml:"issuer-url,omitempty"`

	// ClientID is the OAuth2 client ID registered with the OIDC provider.
	ClientID string `env:"GRAFANA_OIDC_CLIENT_ID" json:"client-id,omitempty" yaml:"client-id,omitempty"`

	// ClientSecret is the OAuth2 client secret. Optional for public clients using PKCE.
	ClientSecret string `datapolicy:"secret" env:"GRAFANA_OIDC_CLIENT_SECRET" json:"client-secret,omitempty" yaml:"client-secret,omitempty"`

	// Scopes is a space-separated list of OAuth2 scopes to request.
	// Defaults to "openid profile email" if not set.
	Scopes string `env:"GRAFANA_OIDC_SCOPES" json:"scopes,omitempty" yaml:"scopes,omitempty"`
}

// IsConfigured returns true if OIDC provider settings are present.
func (o *OIDCConfig) IsConfigured() bool {
	return o != nil && o.IssuerURL != "" && o.ClientID != ""
}

// CachedToken holds cached OAuth2 tokens for a context.
type CachedToken struct {
	AccessToken  string `json:"access-token,omitempty" yaml:"access-token,omitempty"`
	RefreshToken string `json:"refresh-token,omitempty" yaml:"refresh-token,omitempty"`
	TokenExpiry  string `json:"token-expiry,omitempty" yaml:"token-expiry,omitempty"`
}

// TokenCache maps context names to their cached tokens.
type TokenCache struct {
	Contexts map[string]*CachedToken `json:"contexts,omitempty" yaml:"contexts,omitempty"`
}

// Get returns the cached token for a context, or nil if none exists.
func (tc *TokenCache) Get(contextName string) *CachedToken {
	if tc == nil || tc.Contexts == nil {
		return nil
	}

	return tc.Contexts[contextName]
}

// Set stores a cached token for a context.
func (tc *TokenCache) Set(contextName string, token *CachedToken) {
	if tc.Contexts == nil {
		tc.Contexts = make(map[string]*CachedToken)
	}

	tc.Contexts[contextName] = token
}

// TLS contains settings to enable transport layer security.
type TLS struct {
	// InsecureSkipTLSVerify disables the validation of the server's SSL certificate.
	// Enabling this will make your HTTPS connections insecure.
	Insecure bool `json:"insecure-skip-verify,omitempty" yaml:"insecure-skip-verify,omitempty"`

	// ServerName is passed to the server for SNI and is used in the client to check server
	// certificates against. If ServerName is empty, the hostname used to contact the
	// server is used.
	ServerName string `json:"server-name,omitempty" yaml:"server-name,omitempty"`

	// CertData holds PEM-encoded bytes (typically read from a client certificate file).
	// Note: this value is base64-encoded in the config file and will be
	// automatically decoded.
	CertData []byte `json:"cert-data,omitempty" yaml:"cert-data,omitempty"`
	// KeyData holds PEM-encoded bytes (typically read from a client certificate key file).
	// Note: this value is base64-encoded in the config file and will be
	// automatically decoded.
	KeyData []byte `datapolicy:"secret" json:"key-data,omitempty" yaml:"key-data,omitempty"`
	// CAData holds PEM-encoded bytes (typically read from a root certificates bundle).
	// Note: this value is base64-encoded in the config file and will be
	// automatically decoded.
	CAData []byte `json:"ca-data,omitempty" yaml:"ca-data,omitempty"`

	// NextProtos is a list of supported application level protocols, in order of preference.
	// Used to populate tls.Config.NextProtos.
	// To indicate to the server http/1.1 is preferred over http/2, set to ["http/1.1", "h2"] (though the server is free to ignore that preference).
	// To use only http/1.1, set to ["http/1.1"].
	NextProtos []string `json:"next-protos,omitempty" yaml:"next-protos,omitempty"`
}

func (cfg *TLS) ToStdTLSConfig() *tls.Config {
	// TODO: CertData, KeyData, CAData
	return &tls.Config{
		//nolint:gosec
		InsecureSkipVerify: cfg.Insecure,
		ServerName:         cfg.ServerName,
		NextProtos:         cfg.NextProtos,
	}
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
