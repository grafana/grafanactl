package alerts

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoiseReportOptsValidate(t *testing.T) {
	tests := []struct {
		name    string
		opts    noiseReportOpts
		wantErr string
	}{
		{
			name: "valid defaults",
			opts: func() noiseReportOpts {
				o := noiseReportOpts{Threshold: 5}
				o.IO.RegisterCustomCodec("text", &noiseReportTableCodec{})
				o.IO.DefaultFormat("text")
				o.IO.OutputFormat = "text"
				return o
			}(),
		},
		{
			name:    "threshold zero is invalid",
			opts:    noiseReportOpts{Threshold: 0},
			wantErr: "--threshold must be at least 1",
		},
		{
			name:    "negative threshold is invalid",
			opts:    noiseReportOpts{Threshold: -1},
			wantErr: "--threshold must be at least 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNoiseReportCmdStructure(t *testing.T) {
	cmd := noiseReportCmd(nil)

	assert.Equal(t, "noise-report", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	periodFlag := cmd.Flags().Lookup("period")
	require.NotNil(t, periodFlag)
	assert.Equal(t, "7d", periodFlag.DefValue)

	thresholdFlag := cmd.Flags().Lookup("threshold")
	require.NotNil(t, thresholdFlag)
	assert.Equal(t, "5", thresholdFlag.DefValue)

	outputFlag := cmd.Flags().Lookup("output")
	require.NotNil(t, outputFlag)
}

func TestAnalyzeNoise(t *testing.T) {
	tests := []struct {
		name      string
		entries   []stateHistoryEntry
		threshold int
		want      []NoiseEntry
	}{
		{
			name:      "empty entries returns empty result",
			entries:   nil,
			threshold: 5,
			want:      []NoiseEntry{},
		},
		{
			name: "single alert with fire and resolve is meaningful",
			entries: []stateHistoryEntry{
				{RuleTitle: "CPU High", Current: "Alerting", Timestamp: time.UnixMilli(1000)},
				{RuleTitle: "CPU High", Current: "Normal", Timestamp: time.UnixMilli(61000)},
			},
			threshold: 5,
			want: []NoiseEntry{
				{AlertName: "CPU High", FireCount: 1, ResolveCount: 1, AvgDuration: "1m0s", Classification: "meaningful"},
			},
		},
		{
			name: "alert exceeding threshold is noisy",
			entries: func() []stateHistoryEntry {
				var entries []stateHistoryEntry
				for i := range 10 {
					entries = append(entries, stateHistoryEntry{
						RuleTitle: "Flappy Alert",
						Current:   "Alerting",
						Timestamp: time.UnixMilli(int64(i) * 60000),
					})
					entries = append(entries, stateHistoryEntry{
						RuleTitle: "Flappy Alert",
						Current:   "Normal",
						Timestamp: time.UnixMilli(int64(i)*60000 + 30000),
					})
				}
				return entries
			}(),
			threshold: 5,
			want: []NoiseEntry{
				{AlertName: "Flappy Alert", FireCount: 10, ResolveCount: 10, AvgDuration: "30s", Classification: "noisy"},
			},
		},
		{
			name: "at threshold is still meaningful",
			entries: func() []stateHistoryEntry {
				var entries []stateHistoryEntry
				for i := range 5 {
					entries = append(entries, stateHistoryEntry{
						RuleTitle: "Edge Case",
						Current:   "Alerting",
						Timestamp: time.UnixMilli(int64(i) * 10000),
					})
					entries = append(entries, stateHistoryEntry{
						RuleTitle: "Edge Case",
						Current:   "Normal",
						Timestamp: time.UnixMilli(int64(i)*10000 + 1000),
					})
				}
				return entries
			}(),
			threshold: 5,
			want: []NoiseEntry{
				{AlertName: "Edge Case", FireCount: 5, ResolveCount: 5, AvgDuration: "1s", Classification: "meaningful"},
			},
		},
		{
			name: "multiple alerts sorted by fire count descending",
			entries: []stateHistoryEntry{
				{RuleTitle: "Low Fire", Current: "Alerting", Timestamp: time.UnixMilli(1000)},
				{RuleTitle: "Low Fire", Current: "Normal", Timestamp: time.UnixMilli(2000)},
				{RuleTitle: "High Fire", Current: "Alerting", Timestamp: time.UnixMilli(1000)},
				{RuleTitle: "High Fire", Current: "Normal", Timestamp: time.UnixMilli(2000)},
				{RuleTitle: "High Fire", Current: "Alerting", Timestamp: time.UnixMilli(3000)},
				{RuleTitle: "High Fire", Current: "Normal", Timestamp: time.UnixMilli(4000)},
				{RuleTitle: "High Fire", Current: "Alerting", Timestamp: time.UnixMilli(5000)},
				{RuleTitle: "High Fire", Current: "Normal", Timestamp: time.UnixMilli(6000)},
				{RuleTitle: "Mid Fire", Current: "Alerting", Timestamp: time.UnixMilli(1000)},
				{RuleTitle: "Mid Fire", Current: "Normal", Timestamp: time.UnixMilli(2000)},
				{RuleTitle: "Mid Fire", Current: "Alerting", Timestamp: time.UnixMilli(3000)},
				{RuleTitle: "Mid Fire", Current: "Normal", Timestamp: time.UnixMilli(4000)},
			},
			threshold: 5,
			want: []NoiseEntry{
				{AlertName: "High Fire", FireCount: 3, ResolveCount: 3, AvgDuration: "1s", Classification: "meaningful"},
				{AlertName: "Mid Fire", FireCount: 2, ResolveCount: 2, AvgDuration: "1s", Classification: "meaningful"},
				{AlertName: "Low Fire", FireCount: 1, ResolveCount: 1, AvgDuration: "1s", Classification: "meaningful"},
			},
		},
		{
			name: "mixed states: Alerting, Firing, OK, Normal",
			entries: []stateHistoryEntry{
				{RuleTitle: "Mixed", Current: "Alerting", Timestamp: time.UnixMilli(1000)},
				{RuleTitle: "Mixed", Current: "OK", Timestamp: time.UnixMilli(61000)},
				{RuleTitle: "Mixed", Current: "Firing", Timestamp: time.UnixMilli(70000)},
				{RuleTitle: "Mixed", Current: "Normal", Timestamp: time.UnixMilli(130000)},
			},
			threshold: 5,
			want: []NoiseEntry{
				{AlertName: "Mixed", FireCount: 2, ResolveCount: 2, AvgDuration: "1m0s", Classification: "meaningful"},
			},
		},
		{
			name: "fire without resolve has no duration",
			entries: []stateHistoryEntry{
				{RuleTitle: "NoDuration", Current: "Alerting", Timestamp: time.UnixMilli(1000)},
				{RuleTitle: "NoDuration", Current: "Alerting", Timestamp: time.UnixMilli(2000)},
			},
			threshold: 5,
			want: []NoiseEntry{
				{AlertName: "NoDuration", FireCount: 2, ResolveCount: 0, AvgDuration: "", Classification: "meaningful"},
			},
		},
		{
			name: "uid populated from ruleUID",
			entries: []stateHistoryEntry{
				{RuleTitle: "WithUID", Current: "Alerting", RuleUID: "rule-123", Timestamp: time.UnixMilli(1000)},
				{RuleTitle: "WithUID", Current: "Normal", Timestamp: time.UnixMilli(2000)},
			},
			threshold: 5,
			want: []NoiseEntry{
				{AlertName: "WithUID", UID: "rule-123", FireCount: 1, ResolveCount: 1, AvgDuration: "1s", Classification: "meaningful"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := analyzeNoise(tt.entries, tt.threshold)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParsePeriod(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "7 days", input: "7d", want: "168h0m0s"},
		{name: "1 day", input: "1d", want: "24h0m0s"},
		{name: "30 days", input: "30d", want: "720h0m0s"},
		{name: "24 hours", input: "24h", want: "24h0m0s"},
		{name: "1 hour", input: "1h", want: "1h0m0s"},
		{name: "30 minutes", input: "30m", want: "30m0s"},
		{name: "invalid day format", input: "xd", wantErr: true},
		{name: "invalid format", input: "abc", wantErr: true},
		{name: "empty string", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePeriod(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.String())
		})
	}
}

func TestNoiseReportTableCodecFormat(t *testing.T) {
	codec := &noiseReportTableCodec{}
	assert.Equal(t, "text", string(codec.Format()))
}

func TestNoiseReportTableCodecDecode(t *testing.T) {
	codec := &noiseReportTableCodec{}
	err := codec.Decode(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support decoding")
}

func TestNoiseReportTableCodecEncode(t *testing.T) {
	tests := []struct {
		name        string
		input       any
		wantErr     string
		wantOutputs []string
		notOutputs  []string
	}{
		{
			name:    "wrong input type",
			input:   "not entries",
			wantErr: "expected []NoiseEntry",
		},
		{
			name:  "empty entries list",
			input: []NoiseEntry{},
			wantOutputs: []string{
				"ALERT_NAME",
				"FIRES",
				"RESOLVES",
				"AVG_DURATION",
				"CLASSIFICATION",
			},
			notOutputs: []string{
				"noisy",
				"meaningful",
			},
		},
		{
			name: "single entry",
			input: []NoiseEntry{
				{AlertName: "CPU High", UID: "uid-1", FireCount: 10, ResolveCount: 8, AvgDuration: "5m0s", Classification: "noisy"},
			},
			wantOutputs: []string{
				"ALERT_NAME",
				"CPU High",
				"uid-1",
				"10",
				"8",
				"5m0s",
				"noisy",
			},
		},
		{
			name: "multiple entries",
			input: []NoiseEntry{
				{AlertName: "Alert A", FireCount: 20, ResolveCount: 15, AvgDuration: "2m0s", Classification: "noisy"},
				{AlertName: "Alert B", FireCount: 2, ResolveCount: 1, AvgDuration: "30s", Classification: "meaningful"},
			},
			wantOutputs: []string{
				"Alert A", "20", "noisy",
				"Alert B", "2", "meaningful",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codec := &noiseReportTableCodec{}
			var buf bytes.Buffer
			err := codec.Encode(&buf, tt.input)

			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}

			assert.NoError(t, err)
			output := buf.String()

			for _, want := range tt.wantOutputs {
				assert.Contains(t, output, want)
			}

			for _, notWant := range tt.notOutputs {
				assert.NotContains(t, output, notWant)
			}
		})
	}
}
