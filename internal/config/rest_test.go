package config

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	authlib "github.com/grafana/authlib/types"
)

func TestNewNamespacedRESTConfig_UsesBootdataStack(t *testing.T) {
	bootdataServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/grafana/bootdata" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"settings": map[string]any{
				"namespace": "stacks-98765",
			},
		})
	}))
	defer bootdataServer.Close()

	ctx := Context{
		Grafana: &GrafanaConfig{
			Server:  bootdataServer.URL + "/grafana",
			StackID: 12345,
		},
	}

	restCfg := NewNamespacedRESTConfig(ctx)

	if got, want := restCfg.Namespace, authlib.CloudNamespaceFormatter(98765); got != want {
		t.Fatalf("expected namespace %s, got %s", want, got)
	}

	if ctx.Grafana.StackID != 12345 {
		t.Fatalf("expected original stack ID to remain unchanged, got %d", ctx.Grafana.StackID)
	}
}

func TestNewNamespacedRESTConfig_FallsBackOnBootdataError(t *testing.T) {
	bootdataServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer bootdataServer.Close()

	ctx := Context{
		Grafana: &GrafanaConfig{
			Server:  bootdataServer.URL,
			StackID: 555,
		},
	}

	restCfg := NewNamespacedRESTConfig(ctx)

	if got, want := restCfg.Namespace, authlib.CloudNamespaceFormatter(555); got != want {
		t.Fatalf("expected namespace %s, got %s", want, got)
	}
}

func TestNewNamespacedRESTConfig_FallsBackWhenBootdataNotStack(t *testing.T) {
	bootdataServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"settings": map[string]any{
				"namespace": "grafana",
			},
		})
	}))
	defer bootdataServer.Close()

	ctx := Context{
		Grafana: &GrafanaConfig{
			Server:  bootdataServer.URL,
			StackID: 42,
		},
	}

	restCfg := NewNamespacedRESTConfig(ctx)

	if got, want := restCfg.Namespace, authlib.CloudNamespaceFormatter(42); got != want {
		t.Fatalf("expected namespace %s, got %s", want, got)
	}
}
