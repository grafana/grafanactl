package alerts

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
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

// stateHistoryFrame represents the Grafana data frame JSON response from /api/v1/rules/history.
type stateHistoryFrame struct {
	Schema struct {
		Fields []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"fields"`
	} `json:"schema"`
	Data struct {
		Values []json.RawMessage `json:"values"`
	} `json:"data"`
}

// stateHistoryEntry represents a parsed LokiEntry from the state history API line field.
type stateHistoryEntry struct {
	SchemaVersion int               `json:"schemaVersion"`
	Previous      string            `json:"previous"`
	Current       string            `json:"current"`
	Error         string            `json:"error,omitempty"`
	Values        map[string]any    `json:"values,omitempty"`
	Condition     string            `json:"condition,omitempty"`
	DashboardUID  string            `json:"dashboardUID,omitempty"`
	PanelID       int64             `json:"panelID,omitempty"`
	Fingerprint   string            `json:"fingerprint,omitempty"`
	RuleTitle     string            `json:"ruleTitle"`
	RuleID        int64             `json:"ruleID,omitempty"`
	RuleUID       string            `json:"ruleUID"`
	Labels        map[string]string `json:"labels,omitempty"`
	Timestamp     time.Time         `json:"-"` // populated from the frame's time field, not from JSON
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

func fetchStateHistory(ctx context.Context, gCtx *config.Context, from, to int64) ([]stateHistoryEntry, error) {
	requestURL, err := url.Parse(gCtx.Grafana.Server)
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %w", err)
	}

	requestURL.Path += "/api/v1/rules/history"

	q := requestURL.Query()
	q.Set("from", strconv.FormatInt(from, 10))
	q.Set("to", strconv.FormatInt(to, 10))
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
		return nil, fmt.Errorf("request to /api/v1/rules/history failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request to /api/v1/rules/history failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read state history response: %w", err)
	}

	return parseStateHistoryFrame(body)
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

func parseStateHistoryFrame(data []byte) ([]stateHistoryEntry, error) {
	var frame stateHistoryFrame
	if err := json.Unmarshal(data, &frame); err != nil {
		return nil, fmt.Errorf("failed to decode state history response: %w", err)
	}

	timeIdx := -1
	lineIdx := -1

	for i, field := range frame.Schema.Fields {
		switch field.Name {
		case "time":
			timeIdx = i
		case "line":
			lineIdx = i
		}
	}

	if timeIdx == -1 || lineIdx == -1 {
		return nil, nil
	}

	if len(frame.Data.Values) <= timeIdx || len(frame.Data.Values) <= lineIdx {
		return nil, nil
	}

	var timestamps []int64
	if err := json.Unmarshal(frame.Data.Values[timeIdx], &timestamps); err != nil {
		return nil, fmt.Errorf("failed to parse state history timestamps: %w", err)
	}

	var lines []json.RawMessage
	if err := json.Unmarshal(frame.Data.Values[lineIdx], &lines); err != nil {
		return nil, fmt.Errorf("failed to parse state history lines: %w", err)
	}

	entries := make([]stateHistoryEntry, 0, len(lines))

	for i, line := range lines {
		var entry stateHistoryEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		if i < len(timestamps) {
			entry.Timestamp = time.UnixMilli(timestamps[i])
		}

		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})

	return entries, nil
}
