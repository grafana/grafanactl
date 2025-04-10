package grafana

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/grafana/grafanactl/internal/config"
	"github.com/grafana/grafanactl/internal/httputils"
)

func AuthenticateAndProxyHandler(cfg *config.GrafanaConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html")

		if cfg.Server == "" {
			httputils.Error(r, w, "Error: No Grafana URL configured", errors.New("no Grafana URL configured"), http.StatusBadRequest)
			return
		}

		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, cfg.Server+r.URL.Path, nil)
		if err != nil {
			httputils.Error(r, w, http.StatusText(http.StatusInternalServerError), err, http.StatusInternalServerError)
			return
		}

		AuthenticateRequest(cfg, req)
		req.Header.Set("User-Agent", httputils.UserAgent)

		client, err := httputils.NewHTTPClient()
		if err != nil {
			httputils.Error(r, w, http.StatusText(http.StatusInternalServerError), err, http.StatusInternalServerError)
			return
		}

		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			httputils.Write(r, w, body)
			return
		}

		// TODO
		msg := ""
		if cfg.APIToken == "" {
			msg += "<p><b>Warning:</b> No service account token specified.</p>"
		}

		if resp.StatusCode == http.StatusFound {
			w.WriteHeader(http.StatusUnauthorized)
			httputils.Write(r, w, []byte(msg+"<p>Authentication error</p>"))
		} else {
			body, _ := io.ReadAll(resp.Body)
			w.WriteHeader(resp.StatusCode)
			httputils.Write(r, w, []byte(fmt.Sprintf("%s%s", msg, string(body))))
		}
	}
}

func AuthenticateRequest(config *config.GrafanaConfig, request *http.Request) {
	if config.User != "" {
		request.SetBasicAuth(config.User, config.Password)
	} else if config.APIToken != "" {
		request.Header.Set("Authorization", "Bearer "+config.APIToken)
	}
}
