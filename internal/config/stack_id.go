package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	authlib "github.com/grafana/authlib/types"
)

var errBootdataNonOK = errors.New("bootdata request failed")

// discoverStackId attempts to discover a Grafana Cloud stack namespace via the /bootdata endpoint.
// It returns the parsed stack ID when the response matches the expected format.
func discoverStackId(ctx context.Context, cfg GrafanaConfig) (stackID int64, ok bool, err error) {
	bootdataURL, err := buildBootdataURL(cfg.Server)
	if err != nil {
		return 0, false, err
	}

	client := newBootdataHTTPClient(cfg)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, bootdataURL.String(), nil)
	if err != nil {
		return 0, false, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, false, fmt.Errorf("%w: status %d", errBootdataNonOK, resp.StatusCode)
	}

	var payload struct {
		Settings struct {
			Namespace string `json:"namespace"`
		} `json:"settings"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, false, err
	}

	namespace := strings.TrimSpace(payload.Settings.Namespace)
	if namespace == "" {
		return 0, false, nil
	}

	ns, err := authlib.ParseNamespace(namespace)
	if err != nil {
		return 0, false, nil
	}

	if ns.StackID == 0 {
		return 0, false, nil
	}

	return ns.StackID, true, nil
}

func buildBootdataURL(server string) (*url.URL, error) {
	parsed, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	trimmedPath := strings.TrimSuffix(parsed.Path, "/")
	parsed.Path = trimmedPath + "/bootdata"
	parsed.RawQuery = ""
	parsed.Fragment = ""

	return parsed, nil
}

func newBootdataHTTPClient(cfg GrafanaConfig) *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	if cfg.TLS != nil {
		transport.TLSClientConfig = cfg.TLS.ToStdTLSConfig()
	}

	return &http.Client{
		Timeout:   5 * time.Second,
		Transport: transport,
	}
}
