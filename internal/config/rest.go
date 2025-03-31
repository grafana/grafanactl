package config

import (
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

	// TODO: add auth parsing
	// userInfo, orgID, apiKey, err := parseAuth(cfg)
	// if err != nil {
	// 	return err
	// }
	// switch {
	// case apiKey != "":
	// 	if orgID > 1 {
	// 		return fmt.Errorf("org_id is only supported with basic auth. API keys are already org-scoped")
	// 	}
	// 	rcfg.BearerToken = apiKey
	// case userInfo != nil:
	// 	rcfg.Username = userInfo.Username()
	// 	if p, ok := userInfo.Password(); ok {
	// 		rcfg.Password = p
	// 	}
	// }
	rcfg.BearerToken = cfg.Grafana.Token

	// TODO: add proper namespace parsing once we have orgID / stackID
	// 	ns = claims.OrgNamespaceFormatter(cfg.Grafana.OrgID)
	// switch {
	// case cfg.Grafana.OrgID > 0:
	// 	ns = claims.OrgNamespaceFormatter(cfg.Grafana.OrgID)
	// case cfg.Grafana.StackID > 0:
	// 	ns = claims.CloudNamespaceFormatter(cfg.Grafana.StackID)
	// }
	namespace := claims.OrgNamespaceFormatter(1)

	return NamespacedRESTConfig{
		Config:    rcfg,
		Namespace: namespace,
	}, nil
}
