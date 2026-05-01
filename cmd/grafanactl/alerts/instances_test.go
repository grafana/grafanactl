package alerts

import (
	"bytes"
	"testing"

	cmdconfig "github.com/grafana/grafanactl/cmd/grafanactl/config"
	"github.com/stretchr/testify/assert"
)

func TestInstancesOptsValidate(t *testing.T) {
	tests := []struct {
		name    string
		state   string
		wantErr string
	}{
		{
			name:  "no state filter is valid",
			state: "",
		},
		{
			name:  "firing state is valid",
			state: "firing",
		},
		{
			name:  "pending state is valid",
			state: "pending",
		},
		{
			name:    "invalid state is rejected",
			state:   "invalid",
			wantErr: "invalid state filter",
		},
		{
			name:    "unknown state is rejected",
			state:   "resolved",
			wantErr: "invalid state filter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &instancesOpts{
				State: tt.state,
			}
			opts.IO.RegisterCustomCodec("text", &instanceTableCodec{})
			opts.IO.DefaultFormat("text")
			opts.IO.OutputFormat = "text"

			err := opts.Validate()

			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestInstancesCmdStructure(t *testing.T) {
	configOpts := &cmdconfig.Options{}
	cmd := instancesCmd(configOpts)

	assert.Equal(t, "instances", cmd.Use)
	assert.Equal(t, "List currently firing alert instances", cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotEmpty(t, cmd.Example)

	stateFlag := cmd.Flags().Lookup("state")
	assert.NotNil(t, stateFlag, "should have --state flag")
	assert.Equal(t, "", stateFlag.DefValue)

	outputFlag := cmd.Flags().Lookup("output")
	assert.NotNil(t, outputFlag, "should have --output flag")
	assert.Equal(t, "o", outputFlag.Shorthand)
	assert.Equal(t, "text", outputFlag.DefValue)
}

func TestInstanceTableCodecFormat(t *testing.T) {
	codec := &instanceTableCodec{}
	assert.Equal(t, "text", string(codec.Format()))
}

func TestInstanceTableCodecDecode(t *testing.T) {
	codec := &instanceTableCodec{}
	err := codec.Decode(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not support decoding")
}

func TestInstanceTableCodecEncode(t *testing.T) {
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
			wantErr: "expected []instanceEntry",
		},
		{
			name:  "empty entries list",
			input: []instanceEntry{},
			wantOutputs: []string{
				"RULE",
				"STATE",
				"ACTIVE_SINCE",
				"LABELS",
				"VALUE",
			},
			notOutputs: []string{
				"firing",
				"pending",
			},
		},
		{
			name: "single firing instance",
			input: []instanceEntry{
				{
					RuleName: "CPU High",
					State:    "firing",
					ActiveAt: "2025-01-15 10:30:00",
					Labels:   "host=server1,severity=critical",
					Value:    "95.2",
				},
			},
			wantOutputs: []string{
				"RULE",
				"CPU High",
				"firing",
				"2025-01-15 10:30:00",
				"host=server1,severity=critical",
				"95.2",
			},
		},
		{
			name: "multiple instances",
			input: []instanceEntry{
				{
					RuleName: "CPU High",
					State:    "firing",
					ActiveAt: "2025-01-15 10:30:00",
					Labels:   "host=server1",
					Value:    "95.2",
				},
				{
					RuleName: "Memory Low",
					State:    "pending",
					ActiveAt: "2025-01-15 11:00:00",
					Labels:   "host=server2",
					Value:    "128MB",
				},
			},
			wantOutputs: []string{
				"CPU High", "firing", "95.2",
				"Memory Low", "pending", "128MB",
			},
		},
		{
			name: "instance with empty fields",
			input: []instanceEntry{
				{
					RuleName: "Test Alert",
					State:    "firing",
					ActiveAt: "",
					Labels:   "",
					Value:    "",
				},
			},
			wantOutputs: []string{
				"Test Alert",
				"firing",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codec := &instanceTableCodec{}
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

func TestFormatLabels(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		want   string
	}{
		{
			name:   "nil labels",
			labels: nil,
			want:   "",
		},
		{
			name:   "empty labels",
			labels: map[string]string{},
			want:   "",
		},
		{
			name:   "single label",
			labels: map[string]string{"host": "server1"},
			want:   "host=server1",
		},
		{
			name:   "multiple labels sorted",
			labels: map[string]string{"severity": "critical", "host": "server1", "app": "api"},
			want:   "app=api,host=server1,severity=critical",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatLabels(tt.labels)
			assert.Equal(t, tt.want, got)
		})
	}
}
