package alerts

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/grafana/grafanactl/internal/config"
	"github.com/grafana/grafanactl/internal/httputils"
)

// rulesResponse is the response from /api/prometheus/grafana/api/v1/rules.
type rulesResponse struct {
	Data struct {
		RuleGroups []ruleGroup      `json:"groups"`
		Totals     map[string]int64 `json:"totals,omitempty"`
	} `json:"data"`
}

// alertsResponse is the response from /api/prometheus/grafana/api/v1/alerts.
type alertsResponse struct {
	Data struct {
		Alerts []alertInstance `json:"alerts"`
	} `json:"data"`
}

type ruleGroup struct {
	Name           string         `json:"name"`
	FolderUID      string         `json:"folderUid"`
	Rules          []alertingRule `json:"rules"`
	Interval       float64        `json:"interval"`
	LastEvaluation time.Time      `json:"lastEvaluation"`
}

type alertingRule struct {
	State          string            `json:"state"`
	Name           string            `json:"name"`
	Health         string            `json:"health"`
	LastEvaluation time.Time         `json:"lastEvaluation"`
	Alerts         []alertInstance   `json:"alerts"`
	UID            string            `json:"uid"`
	FolderUID      string            `json:"folderUid"`
	Labels         map[string]string `json:"labels"`
	Totals         map[string]int64  `json:"totals,omitempty"`
	Type           string            `json:"type"`
}

type alertInstance struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	State       string            `json:"state"`
	ActiveAt    *time.Time        `json:"activeAt"`
	Value       string            `json:"value"`
}

// alertAnnotation represents an alert state change annotation from /api/annotations.
type alertAnnotation struct {
	ID           int64  `json:"id"`
	AlertID      int64  `json:"alertId"`
	AlertName    string `json:"alertName"`
	NewState     string `json:"newState"`
	PrevState    string `json:"prevState"`
	Time         int64  `json:"time"`
	TimeEnd      int64  `json:"timeEnd"`
	DashboardID  int64  `json:"dashboardId"`
	DashboardUID string `json:"dashboardUID"`
	PanelID      int64  `json:"panelId"`
	Text         string `json:"text"`
}

func fetchRulesFromPrometheusAPI(ctx context.Context, gCtx *config.Context) (*rulesResponse, error) {
	resp, err := makeAlertingRequest(ctx, gCtx, "/api/prometheus/grafana/api/v1/rules")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result rulesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode rules response: %w", err)
	}

	return &result, nil
}

func fetchFiringAlerts(ctx context.Context, gCtx *config.Context) ([]alertInstance, error) {
	resp, err := makeAlertingRequest(ctx, gCtx, "/api/prometheus/grafana/api/v1/alerts")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result alertsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode alerts response: %w", err)
	}

	return result.Data.Alerts, nil
}

func fetchAlertAnnotations(ctx context.Context, gCtx *config.Context, from, to int64) ([]alertAnnotation, error) {
	requestURL, err := url.Parse(gCtx.Grafana.Server)
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %w", err)
	}

	requestURL.Path += "/api/annotations"

	q := requestURL.Query()
	q.Set("type", "alert")
	q.Set("from", strconv.FormatInt(from, 10))
	q.Set("to", strconv.FormatInt(to, 10))
	q.Set("limit", "5000")
	requestURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return nil, err
	}

	setAuthHeaders(req, gCtx)

	httpClient := &http.Client{
		Timeout:   alertingRequestTimeout,
		Transport: httputils.NewTransport(gCtx),
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to /api/annotations failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request to /api/annotations failed with status %d: %s", resp.StatusCode, string(body))
	}

	var annotations []alertAnnotation
	if err := json.NewDecoder(resp.Body).Decode(&annotations); err != nil {
		return nil, fmt.Errorf("failed to decode annotations response: %w", err)
	}

	return annotations, nil
}

// alertingRequestTimeout is the timeout for alerting API requests.
// The Prometheus-compatible rules API can return very large responses
// on instances with many alert rules, requiring a longer timeout than
// the default 10s used by httputils.NewHTTPClient.
const alertingRequestTimeout = 60 * time.Second

func makeAlertingRequest(ctx context.Context, gCtx *config.Context, path string) (*http.Response, error) {
	requestURL, err := url.Parse(gCtx.Grafana.Server)
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %w", err)
	}

	requestURL.Path += path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return nil, err
	}

	setAuthHeaders(req, gCtx)

	httpClient := &http.Client{
		Timeout:   alertingRequestTimeout,
		Transport: httputils.NewTransport(gCtx),
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", path, err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		return nil, fmt.Errorf("request to %s failed with status %d: %s", path, resp.StatusCode, string(body))
	}

	return resp, nil
}

func setAuthHeaders(req *http.Request, gCtx *config.Context) {
	if gCtx.Grafana.APIToken != "" {
		req.Header.Set("Authorization", "Bearer "+gCtx.Grafana.APIToken)
	} else if gCtx.Grafana.User != "" && gCtx.Grafana.Password != "" {
		req.SetBasicAuth(gCtx.Grafana.User, gCtx.Grafana.Password)
	}

	if gCtx.Grafana.OrgID != 0 {
		req.Header.Set("X-Grafana-Org-Id", strconv.FormatInt(gCtx.Grafana.OrgID, 10))
	}
}
