package alerts

import (
	"bytes"
	"testing"
	"time"

	cmdconfig "github.com/grafana/grafanactl/cmd/grafanactl/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTimeArg(t *testing.T) {
	tests := []struct {
		name    string
		arg     string
		wantErr string
		check   func(t *testing.T, ms int64)
	}{
		{
			name: "now returns current time",
			arg:  "now",
			check: func(t *testing.T, ms int64) {
				t.Helper()
				diff := time.Now().UnixMilli() - ms
				assert.Less(t, diff, int64(1000), "should be within 1 second of now")
			},
		},
		{
			name: "empty string returns current time",
			arg:  "",
			check: func(t *testing.T, ms int64) {
				t.Helper()
				diff := time.Now().UnixMilli() - ms
				assert.Less(t, diff, int64(1000), "should be within 1 second of now")
			},
		},
		{
			name: "epoch milliseconds",
			arg:  "1700000000000",
			check: func(t *testing.T, ms int64) {
				t.Helper()
				assert.Equal(t, int64(1700000000000), ms)
			},
		},
		{
			name: "24h duration",
			arg:  "24h",
			check: func(t *testing.T, ms int64) {
				t.Helper()
				expected := time.Now().Add(-24 * time.Hour).UnixMilli()
				diff := expected - ms
				if diff < 0 {
					diff = -diff
				}
				assert.Less(t, diff, int64(1000), "should be within 1 second of 24h ago")
			},
		},
		{
			name: "7d duration",
			arg:  "7d",
			check: func(t *testing.T, ms int64) {
				t.Helper()
				expected := time.Now().Add(-7 * 24 * time.Hour).UnixMilli()
				diff := expected - ms
				if diff < 0 {
					diff = -diff
				}
				assert.Less(t, diff, int64(1000), "should be within 1 second of 7d ago")
			},
		},
		{
			name: "30d duration",
			arg:  "30d",
			check: func(t *testing.T, ms int64) {
				t.Helper()
				expected := time.Now().Add(-30 * 24 * time.Hour).UnixMilli()
				diff := expected - ms
				if diff < 0 {
					diff = -diff
				}
				assert.Less(t, diff, int64(1000), "should be within 1 second of 30d ago")
			},
		},
		{
			name: "1h30m duration",
			arg:  "1h30m",
			check: func(t *testing.T, ms int64) {
				t.Helper()
				expected := time.Now().Add(-90 * time.Minute).UnixMilli()
				diff := expected - ms
				if diff < 0 {
					diff = -diff
				}
				assert.Less(t, diff, int64(1000))
			},
		},
		{
			name:    "invalid duration",
			arg:     "abc",
			wantErr: "invalid time",
		},
		{
			name:    "invalid day format",
			arg:     "xd",
			wantErr: "invalid duration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms, err := parseTimeArg(tt.arg, true)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)

				return
			}

			require.NoError(t, err)
			tt.check(t, ms)
		})
	}
}

func TestHistoryOptsValidate(t *testing.T) {
	tests := []struct {
		name    string
		opts    historyOpts
		wantErr bool
	}{
		{
			name: "valid defaults",
			opts: func() historyOpts {
				o := historyOpts{}
				o.IO.RegisterCustomCodec("text", &historyTableCodec{})
				o.IO.DefaultFormat("text")
				o.IO.OutputFormat = "text"
				return o
			}(),
			wantErr: false,
		},
		{
			name: "valid json format",
			opts: func() historyOpts {
				o := historyOpts{}
				o.IO.RegisterCustomCodec("text", &historyTableCodec{})
				o.IO.OutputFormat = "json"
				return o
			}(),
			wantErr: false,
		},
		{
			name: "invalid format",
			opts: func() historyOpts {
				o := historyOpts{}
				o.IO.RegisterCustomCodec("text", &historyTableCodec{})
				o.IO.OutputFormat = "xml"
				return o
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHistoryCmdStructure(t *testing.T) {
	cmd := historyCmd(&cmdconfig.Options{})

	assert.Equal(t, "history", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	flags := cmd.Flags()

	fromFlag := flags.Lookup("from")
	require.NotNil(t, fromFlag)
	assert.Equal(t, "24h", fromFlag.DefValue)

	toFlag := flags.Lookup("to")
	require.NotNil(t, toFlag)
	assert.Equal(t, "now", toFlag.DefValue)

	limitFlag := flags.Lookup("limit")
	require.NotNil(t, limitFlag)
	assert.Equal(t, "1000", limitFlag.DefValue)

	outputFlag := flags.Lookup("output")
	require.NotNil(t, outputFlag)
	assert.Equal(t, "text", outputFlag.DefValue)
}

func TestHistoryTableCodecFormat(t *testing.T) {
	codec := &historyTableCodec{}
	assert.Equal(t, "text", string(codec.Format()))
}

func TestHistoryTableCodecDecode(t *testing.T) {
	codec := &historyTableCodec{}
	err := codec.Decode(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not support decoding")
}

func TestHistoryTableCodecEncode(t *testing.T) {
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
			wantErr: "expected []historyEntry",
		},
		{
			name:  "empty entries",
			input: []historyEntry{},
			wantOutputs: []string{
				"TIME",
				"ALERT_NAME",
				"PREVIOUS_STATE",
				"NEW_STATE",
				"DURATION",
			},
		},
		{
			name: "single entry without duration",
			input: []historyEntry{
				{
					Time:      "2025-01-15T10:00:00Z",
					AlertName: "CPU High",
					PrevState: "Normal",
					NewState:  "Alerting",
				},
			},
			wantOutputs: []string{
				"2025-01-15T10:00:00Z",
				"CPU High",
				"Normal",
				"Alerting",
			},
		},
		{
			name: "entry with duration",
			input: []historyEntry{
				{
					Time:      "2025-01-15T10:00:00Z",
					AlertName: "Disk Full",
					PrevState: "Alerting",
					NewState:  "Normal",
					Duration:  "5m30s",
				},
			},
			wantOutputs: []string{
				"Disk Full",
				"5m30s",
			},
		},
		{
			name: "multiple entries",
			input: []historyEntry{
				{
					Time:      "2025-01-15T10:00:00Z",
					AlertName: "Alert A",
					PrevState: "Normal",
					NewState:  "Alerting",
					Duration:  "1h0m",
				},
				{
					Time:      "2025-01-15T11:00:00Z",
					AlertName: "Alert B",
					PrevState: "Alerting",
					NewState:  "Normal",
				},
			},
			wantOutputs: []string{
				"Alert A", "Alert B",
				"Normal", "Alerting",
				"1h0m",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codec := &historyTableCodec{}
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

func TestAnnotationsToHistoryEntries(t *testing.T) {
	annotations := []alertAnnotation{
		{
			AlertName: "CPU High",
			PrevState: "Normal",
			NewState:  "Alerting",
			Time:      1705312800000, // 2024-01-15T10:00:00Z
			TimeEnd:   1705316400000, // 2024-01-15T11:00:00Z (1h later)
		},
		{
			AlertName: "Disk Full",
			PrevState: "Alerting",
			NewState:  "Normal",
			Time:      1705316400000,
			TimeEnd:   0, // no end time
		},
		{
			AlertName: "Memory Leak",
			PrevState: "Normal",
			NewState:  "Pending",
			Time:      1705316400000,
			TimeEnd:   1705316400000, // same as start (no duration)
		},
	}

	entries := annotationsToHistoryEntries(annotations)

	require.Len(t, entries, 3)

	assert.Equal(t, "CPU High", entries[0].AlertName)
	assert.Equal(t, "Normal", entries[0].PrevState)
	assert.Equal(t, "Alerting", entries[0].NewState)
	assert.Equal(t, "1h0m", entries[0].Duration)
	assert.Equal(t, "2024-01-15T10:00:00Z", entries[0].Time)

	assert.Equal(t, "Disk Full", entries[1].AlertName)
	assert.Empty(t, entries[1].Duration)

	assert.Equal(t, "Memory Leak", entries[2].AlertName)
	assert.Empty(t, entries[2].Duration)
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		dur  time.Duration
		want string
	}{
		{name: "seconds only", dur: 45 * time.Second, want: "45s"},
		{name: "minutes and seconds", dur: 5*time.Minute + 30*time.Second, want: "5m30s"},
		{name: "hours and minutes", dur: 2*time.Hour + 15*time.Minute, want: "2h15m"},
		{name: "days hours minutes", dur: 3*24*time.Hour + 4*time.Hour + 30*time.Minute, want: "3d4h30m"},
		{name: "exact minute", dur: time.Minute, want: "1m0s"},
		{name: "exact hour", dur: time.Hour, want: "1h0m"},
		{name: "exact day", dur: 24 * time.Hour, want: "1d0h0m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.dur)
			assert.Equal(t, tt.want, got)
		})
	}
}
