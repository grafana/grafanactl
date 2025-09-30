package config

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchBootdataStack_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bootdata" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"settings": map[string]any{
				"namespace": "stacks-12345",
			},
		})
	}))
	defer server.Close()

	cfg := GrafanaConfig{Server: server.URL}

	stackID, ok, err := discoverStackId(context.Background(), cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !ok {
		t.Fatalf("expected ok to be true")
	}
	if stackID != 12345 {
		t.Fatalf("unexpected stack id: %d", stackID)
	}
}

func TestFetchBootdataStack_NonStackNamespace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"settings": map[string]any{
				"namespace": "grafana",
			},
		})
	}))
	defer server.Close()

	cfg := GrafanaConfig{Server: server.URL}

	_, ok, err := discoverStackId(context.Background(), cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ok {
		t.Fatalf("expected ok to be false")
	}
}

func TestFetchBootdataStack_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := GrafanaConfig{Server: server.URL}

	_, _, err := discoverStackId(context.Background(), cfg)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestFetchBootdataStack_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{"))
	}))
	defer server.Close()

	cfg := GrafanaConfig{Server: server.URL}

	_, _, err := discoverStackId(context.Background(), cfg)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestFetchBootdataStack_TLSSkipVerify(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"settings": map[string]any{
				"namespace": "stacks-678",
			},
		})
	}))
	defer server.Close()

	cfg := GrafanaConfig{
		Server: server.URL,
		TLS: &TLS{
			Insecure: true,
		},
	}

	stackID, ok, err := discoverStackId(context.Background(), cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !ok {
		t.Fatalf("expected ok to be true")
	}
	if stackID != 678 {
		t.Fatalf("unexpected stack id: %d", stackID)
	}
}
