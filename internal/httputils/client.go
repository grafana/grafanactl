package httputils

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/grafana/grafanactl/internal/config"
)

func NewTransport(gCtx *config.Context) *http.Transport {
	tlsSkipVerify := false
	if gCtx.Grafana != nil && gCtx.Grafana.TLS != nil {
		tlsSkipVerify = gCtx.Grafana.TLS.Insecure
	}

	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		//nolint:gosec
		TLSClientConfig: &tls.Config{InsecureSkipVerify: tlsSkipVerify},
	}
}

func NewHTTPClient(gCtx *config.Context) (*http.Client, error) {
	return &http.Client{
		Timeout: 10 * time.Second, // TODO: make this configurable maybe?
		Transport: &LoggedHTTPRoundTripper{
			DecoratedTransport: NewTransport(gCtx),
		},
	}, nil
}
