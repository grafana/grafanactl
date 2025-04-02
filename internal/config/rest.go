package config

import (
	"errors"

	"github.com/grafana/authlib/claims"
	"k8s.io/client-go/rest"
)

// NamespacedRESTConfig is a REST config with a namespace.
// TODO: move to app SDK?
type NamespacedRESTConfig struct {
	rest.Config
	Namespace string
}

// NewNamespacedRESTConfig creates a new namespaced REST config.
func NewNamespacedRESTConfig(cfg Context) (NamespacedRESTConfig, error) {
	rcfg := rest.Config{
		// TODO add user agent
		// UserAgent: cfg.UserAgent.ValueString(),
		Host:    cfg.Grafana.Server,
		APIPath: "/apis",
		TLSClientConfig: rest.TLSClientConfig{
			// always true for now because we don't have TLS config parsing yet
			// k8s requires TLS or InsecureSkipTLSVerify=true
			Insecure: true,
		},
		// TODO: make configurable
		QPS:   50,
		Burst: 100,
	}

	// TODO: add TLS config
	// tlsClientConfig, err := parseTLSconfig(cfg)
	// if err != nil {
	// 	return err
	// }

	// // Kubernetes really is wonderful, huh.
	// // tl;dr it has it's own TLSClientConfig,
	// // and it's not compatible with the one from the "crypto/tls" package.
	// rcfg.TLSClientConfig = rest.TLSClientConfig{
	// 	Insecure: tlsClientConfig.InsecureSkipVerify,
	// }

	// if len(tlsClientConfig.CertData) > 0 {
	// 	rcfg.CertData = tlsClientConfig.CertData
	// }

	// if len(tlsClientConfig.KeyData) > 0 {
	// 	rcfg.KeyData = tlsClientConfig.KeyData
	// }

	// if len(tlsClientConfig.CAData) > 0 {
	// 	rcfg.CAData = tlsClientConfig.CAData
	// }

	// Authentication
	switch {
	case cfg.Grafana.APIToken != "":
		if cfg.Grafana.OrgID != 0 {
			return NamespacedRESTConfig{}, errors.New("org_id is only supported with basic auth. API keys are already org-scoped")
		}
		rcfg.BearerToken = cfg.Grafana.APIToken
	case cfg.Grafana.User != "":
		rcfg.Username = cfg.Grafana.User
		rcfg.Password = cfg.Grafana.Password
	}

	// Namespace
	namespace := claims.OrgNamespaceFormatter(1)
	if cfg.Grafana.OrgID != 0 {
		namespace = claims.OrgNamespaceFormatter(cfg.Grafana.OrgID)
	} else if cfg.Grafana.StackID != 0 {
		namespace = claims.CloudNamespaceFormatter(cfg.Grafana.StackID)
	}

	return NamespacedRESTConfig{
		Config:    rcfg,
		Namespace: namespace,
	}, nil
}
