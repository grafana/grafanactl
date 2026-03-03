package loki

import (
	"time"
)

// QueryRequest represents a Loki query request.
type QueryRequest struct {
	Query string
	Start time.Time
	End   time.Time
	Step  time.Duration
	Limit int
}

// IsRange returns true if this is a range query.
func (r QueryRequest) IsRange() bool {
	return !r.Start.IsZero() && !r.End.IsZero()
}

// QueryResponse represents the response from a Loki query.
type QueryResponse struct {
	Status    string          `json:"status"`
	Data      QueryResultData `json:"data"`
	ErrorType string          `json:"errorType,omitempty"`
	Error     string          `json:"error,omitempty"`
}

// QueryResultData holds the query result data.
type QueryResultData struct {
	ResultType string        `json:"resultType"`
	Result     []StreamEntry `json:"result"`
	Stats      *QueryStats   `json:"stats,omitempty"`
}

// StreamEntry represents a single log stream from the query result.
type StreamEntry struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"` // [[timestamp, line], ...]
}

// QueryStats contains statistics about the query execution.
type QueryStats struct {
	Summary QuerySummary `json:"summary"`
}

// QuerySummary contains summary statistics.
type QuerySummary struct {
	BytesProcessedPerSecond int64   `json:"bytesProcessedPerSecond,omitempty"`
	LinesProcessedPerSecond int64   `json:"linesProcessedPerSecond,omitempty"`
	TotalBytesProcessed     int64   `json:"totalBytesProcessed,omitempty"`
	TotalLinesProcessed     int64   `json:"totalLinesProcessed,omitempty"`
	ExecTime                float64 `json:"execTime,omitempty"`
}

// LabelsResponse represents the response from the Loki labels API.
type LabelsResponse struct {
	Status string   `json:"status"`
	Data   []string `json:"data"`
}

// SeriesResponse represents the response from the Loki series API.
type SeriesResponse struct {
	Status string              `json:"status"`
	Data   []map[string]string `json:"data"`
}

// GrafanaQueryResponse represents the response from Grafana's datasource query API.
type GrafanaQueryResponse struct {
	Results map[string]GrafanaResult `json:"results"`
}

// GrafanaResult represents a single result from a Grafana query.
type GrafanaResult struct {
	Frames      []DataFrame `json:"frames,omitempty"`
	Error       string      `json:"error,omitempty"`
	ErrorSource string      `json:"errorSource,omitempty"`
	Status      int         `json:"status,omitempty"`
}

// DataFrame represents a Grafana data frame.
type DataFrame struct {
	Schema DataFrameSchema `json:"schema"`
	Data   DataFrameData   `json:"data"`
}

// DataFrameSchema describes the structure of a data frame.
type DataFrameSchema struct {
	Name   string  `json:"name,omitempty"`
	Fields []Field `json:"fields,omitempty"`
}

// Field describes a field in a data frame.
type Field struct {
	Name   string            `json:"name,omitempty"`
	Type   string            `json:"type,omitempty"`
	Labels map[string]string `json:"labels,omitempty"`
}

// DataFrameData contains the actual data values.
type DataFrameData struct {
	Values [][]any `json:"values,omitempty"`
}
