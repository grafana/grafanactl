package config

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiscoverStackID_Success(t *testing.T) {
	req := require.New(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req.Equal("/bootdata", r.URL.Path)
		req.NoError(json.NewEncoder(w).Encode(map[string]any{
			"settings": map[string]any{
				"namespace": "stacks-12345",
			},
		}))
	}))
	defer server.Close()

	cfg := GrafanaConfig{Server: server.URL}

	stackID, err := discoverStackId(context.Background(), cfg)
	req.NoError(err)
	req.Equal(int64(12345), stackID)
}

func TestDiscoverStackID_NonStackNamespace(t *testing.T) {
	req := require.New(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req.NoError(json.NewEncoder(w).Encode(map[string]any{
			"settings": map[string]any{
				"namespace": "grafana",
			},
		}))
	}))
	defer server.Close()

	cfg := GrafanaConfig{Server: server.URL}

	_, err := discoverStackId(context.Background(), cfg)
	req.Error(err)
}

func TestDiscoverStackID_HTTPError(t *testing.T) {
	req := require.New(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := GrafanaConfig{Server: server.URL}

	_, err := discoverStackId(context.Background(), cfg)
	req.Error(err)
}

func TestDiscoverStackID_InvalidJSON(t *testing.T) {
	req := require.New(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, writeErr := w.Write([]byte("{"))
		req.NoError(writeErr)
	}))
	defer server.Close()

	cfg := GrafanaConfig{Server: server.URL}

	_, err := discoverStackId(context.Background(), cfg)
	req.Error(err)
}

func TestDiscoverStackID_TLSSkipVerify(t *testing.T) {
	req := require.New(t)

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req.NoError(json.NewEncoder(w).Encode(map[string]any{
			"settings": map[string]any{
				"namespace": "stacks-678",
			},
		}))
	}))
	defer server.Close()

	cfg := GrafanaConfig{
		Server: server.URL,
		TLS: &TLS{
			Insecure: true,
		},
	}

	stackID, err := discoverStackId(context.Background(), cfg)
	req.NoError(err)
	req.Equal(int64(678), stackID)
}
