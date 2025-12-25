package config

import (
	"context"

	authlib "github.com/grafana/authlib/types"
	"k8s.io/client-go/rest"
)

// NamespacedRESTConfig is a REST config with a namespace.
// TODO: move to app SDK?
type NamespacedRESTConfig struct {
	rest.Config
	Namespace string
}

// NewNamespacedRESTConfig creates a new namespaced REST config.
func NewNamespacedRESTConfig(ctx context.Context, cfg Context) NamespacedRESTConfig {
	rcfg := rest.Config{
		// TODO add user agent
		// UserAgent: cfg.UserAgent.ValueString(),
		Host:            cfg.Grafana.Server,
		APIPath:         "/apis",
		TLSClientConfig: rest.TLSClientConfig{},
		// TODO: make configurable
		QPS:   50,
		Burst: 100,
	}

	if cfg.Grafana.TLS != nil {
		// Kubernetes really is wonderful, huh.
		// tl;dr it has its own TLSClientConfig,
		// and it's not compatible with the one from the "crypto/tls" package.
		rcfg.TLSClientConfig = rest.TLSClientConfig{
			Insecure:   cfg.Grafana.TLS.Insecure,
			ServerName: cfg.Grafana.TLS.ServerName,
			CertData:   cfg.Grafana.TLS.CertData,
			KeyData:    cfg.Grafana.TLS.KeyData,
			CAData:     cfg.Grafana.TLS.CAData,
			NextProtos: cfg.Grafana.TLS.NextProtos,
		}
	}

	// Authentication
	switch {
	case cfg.Grafana.APIToken != "":
		rcfg.BearerToken = cfg.Grafana.APIToken
	case cfg.Grafana.User != "":
		rcfg.Username = cfg.Grafana.User
		rcfg.Password = cfg.Grafana.Password
	}

	// Namespace
	var namespace string

	discoveredStackID, err := DiscoverStackID(ctx, *cfg.Grafana)

	if err == nil {
		// even if cfg.Grafana.OrgID was set - we ignore it, discoveredStackID takes precedent
		namespace = authlib.CloudNamespaceFormatter(discoveredStackID)
	} else {
		if cfg.Grafana.OrgID != 0 {
			namespace = authlib.OrgNamespaceFormatter(cfg.Grafana.OrgID)
		} else {
			namespace = authlib.CloudNamespaceFormatter(cfg.Grafana.StackID)
		}
	}

	return NamespacedRESTConfig{
		Config:    rcfg,
		Namespace: namespace,
	}
}
