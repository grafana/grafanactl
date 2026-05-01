package alerts

import (
	"bytes"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func strPtr(s string) *string {
	return &s
}

func TestDerefStr(t *testing.T) {
	tests := []struct {
		name string
		in   *string
		want string
	}{
		{
			name: "nil pointer returns empty string",
			in:   nil,
			want: "",
		},
		{
			name: "non-nil pointer returns value",
			in:   strPtr("hello"),
			want: "hello",
		},
		{
			name: "empty string pointer returns empty string",
			in:   strPtr(""),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := derefStr(tt.in)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAlertTableCodecFormat(t *testing.T) {
	codec := &alertTableCodec{}
	assert.Equal(t, "text", string(codec.Format()))
}

func TestAlertTableCodecDecode(t *testing.T) {
	codec := &alertTableCodec{}
	err := codec.Decode(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not support decoding")
}

func TestAlertTableCodecEncode(t *testing.T) {
	tests := []struct {
		name        string
		input       any
		wantErr     string
		wantOutputs []string
		notOutputs  []string
	}{
		{
			name:    "wrong input type",
			input:   "not rules",
			wantErr: "expected models.ProvisionedAlertRules",
		},
		{
			name:  "empty rules list",
			input: models.ProvisionedAlertRules{},
			wantOutputs: []string{
				"TITLE",
				"UID",
				"FOLDER",
				"GROUP",
				"STATUS",
			},
			notOutputs: []string{
				"active",
				"paused",
			},
		},
		{
			name: "single active rule",
			input: models.ProvisionedAlertRules{
				&models.ProvisionedAlertRule{
					Title:     strPtr("CPU High"),
					UID:       "alert-1",
					FolderUID: strPtr("folder-a"),
					RuleGroup: strPtr("group-1"),
					IsPaused:  false,
				},
			},
			wantOutputs: []string{
				"TITLE",
				"CPU High",
				"alert-1",
				"folder-a",
				"group-1",
				"active",
			},
		},
		{
			name: "single paused rule",
			input: models.ProvisionedAlertRules{
				&models.ProvisionedAlertRule{
					Title:     strPtr("Disk Full"),
					UID:       "alert-2",
					FolderUID: strPtr("folder-b"),
					RuleGroup: strPtr("group-2"),
					IsPaused:  true,
				},
			},
			wantOutputs: []string{
				"Disk Full",
				"alert-2",
				"paused",
			},
		},
		{
			name: "multiple rules mixed status",
			input: models.ProvisionedAlertRules{
				&models.ProvisionedAlertRule{
					Title:     strPtr("Alert A"),
					UID:       "uid-a",
					FolderUID: strPtr("folder-1"),
					RuleGroup: strPtr("group-1"),
					IsPaused:  false,
				},
				&models.ProvisionedAlertRule{
					Title:     strPtr("Alert B"),
					UID:       "uid-b",
					FolderUID: strPtr("folder-2"),
					RuleGroup: strPtr("group-2"),
					IsPaused:  true,
				},
				&models.ProvisionedAlertRule{
					Title:     strPtr("Alert C"),
					UID:       "uid-c",
					FolderUID: strPtr("folder-1"),
					RuleGroup: strPtr("group-1"),
					IsPaused:  false,
				},
			},
			wantOutputs: []string{
				"Alert A", "uid-a", "active",
				"Alert B", "uid-b", "paused",
				"Alert C", "uid-c", "active",
			},
		},
		{
			name: "rule with nil fields",
			input: models.ProvisionedAlertRules{
				&models.ProvisionedAlertRule{
					Title:     nil,
					UID:       "uid-nil",
					FolderUID: nil,
					RuleGroup: nil,
					IsPaused:  false,
				},
			},
			wantOutputs: []string{
				"uid-nil",
				"active",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codec := &alertTableCodec{}
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

func TestAlertListTableCodecFormat(t *testing.T) {
	codec := &alertListTableCodec{}
	assert.Equal(t, "text", string(codec.Format()))
}

func TestAlertListTableCodecDecode(t *testing.T) {
	codec := &alertListTableCodec{}
	err := codec.Decode(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not support decoding")
}

func TestAlertListTableCodecEncode(t *testing.T) {
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
			wantErr: "expected []AlertListEntry",
		},
		{
			name:  "empty entries",
			input: []AlertListEntry{},
			wantOutputs: []string{
				"TITLE",
				"UID",
				"FOLDER",
				"GROUP",
				"STATUS",
				"STATE",
			},
			notOutputs: []string{
				"firing",
				"inactive",
			},
		},
		{
			name: "single firing rule",
			input: []AlertListEntry{
				{
					Title:     "CPU High",
					UID:       "alert-1",
					FolderUID: "folder-a",
					RuleGroup: "group-1",
					Status:    "active",
					State:     "firing",
				},
			},
			wantOutputs: []string{
				"TITLE", "STATE",
				"CPU High",
				"alert-1",
				"folder-a",
				"group-1",
				"active",
				"firing",
			},
		},
		{
			name: "multiple rules with different states",
			input: []AlertListEntry{
				{
					Title:     "Alert A",
					UID:       "uid-a",
					FolderUID: "folder-1",
					RuleGroup: "group-1",
					Status:    "active",
					State:     "firing",
				},
				{
					Title:     "Alert B",
					UID:       "uid-b",
					FolderUID: "folder-2",
					RuleGroup: "group-2",
					Status:    "paused",
					State:     "inactive",
				},
				{
					Title:     "Alert C",
					UID:       "uid-c",
					FolderUID: "folder-1",
					RuleGroup: "group-1",
					Status:    "active",
					State:     "unknown",
				},
			},
			wantOutputs: []string{
				"Alert A", "uid-a", "active", "firing",
				"Alert B", "uid-b", "paused", "inactive",
				"Alert C", "uid-c", "active", "unknown",
			},
		},
		{
			name: "rule with pending state",
			input: []AlertListEntry{
				{
					Title:     "Memory Warning",
					UID:       "uid-mem",
					FolderUID: "folder-x",
					RuleGroup: "group-x",
					Status:    "active",
					State:     "pending",
				},
			},
			wantOutputs: []string{
				"Memory Warning",
				"uid-mem",
				"pending",
			},
		},
		{
			name: "rule with error state",
			input: []AlertListEntry{
				{
					Title:     "Broken Query",
					UID:       "uid-err",
					FolderUID: "folder-y",
					RuleGroup: "group-y",
					Status:    "active",
					State:     "error",
				},
			},
			wantOutputs: []string{
				"Broken Query",
				"uid-err",
				"error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codec := &alertListTableCodec{}
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

func TestMergeProvisioningWithRuntime(t *testing.T) {
	tests := []struct {
		name                string
		rules               models.ProvisionedAlertRules
		runtimeStateByTitle map[string]string
		want                []AlertListEntry
	}{
		{
			name:                "empty rules",
			rules:               models.ProvisionedAlertRules{},
			runtimeStateByTitle: map[string]string{},
			want:                []AlertListEntry{},
		},
		{
			name: "rules with matching runtime state",
			rules: models.ProvisionedAlertRules{
				&models.ProvisionedAlertRule{
					Title:     strPtr("CPU High"),
					UID:       "uid-1",
					FolderUID: strPtr("folder-a"),
					RuleGroup: strPtr("group-1"),
					IsPaused:  false,
				},
				&models.ProvisionedAlertRule{
					Title:     strPtr("Disk Full"),
					UID:       "uid-2",
					FolderUID: strPtr("folder-b"),
					RuleGroup: strPtr("group-2"),
					IsPaused:  true,
				},
			},
			runtimeStateByTitle: map[string]string{
				"CPU High":  "firing",
				"Disk Full": "inactive",
			},
			want: []AlertListEntry{
				{
					Title:     "CPU High",
					UID:       "uid-1",
					FolderUID: "folder-a",
					RuleGroup: "group-1",
					Status:    "active",
					State:     "firing",
				},
				{
					Title:     "Disk Full",
					UID:       "uid-2",
					FolderUID: "folder-b",
					RuleGroup: "group-2",
					Status:    "paused",
					State:     "inactive",
				},
			},
		},
		{
			name: "rules without runtime state fallback to unknown",
			rules: models.ProvisionedAlertRules{
				&models.ProvisionedAlertRule{
					Title:     strPtr("No Runtime"),
					UID:       "uid-3",
					FolderUID: strPtr("folder-c"),
					RuleGroup: strPtr("group-3"),
					IsPaused:  false,
				},
			},
			runtimeStateByTitle: map[string]string{},
			want: []AlertListEntry{
				{
					Title:     "No Runtime",
					UID:       "uid-3",
					FolderUID: "folder-c",
					RuleGroup: "group-3",
					Status:    "active",
					State:     "unknown",
				},
			},
		},
		{
			name: "rule with nil title",
			rules: models.ProvisionedAlertRules{
				&models.ProvisionedAlertRule{
					Title:     nil,
					UID:       "uid-nil",
					FolderUID: nil,
					RuleGroup: nil,
					IsPaused:  false,
				},
			},
			runtimeStateByTitle: map[string]string{},
			want: []AlertListEntry{
				{
					Title:     "",
					UID:       "uid-nil",
					FolderUID: "",
					RuleGroup: "",
					Status:    "active",
					State:     "unknown",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeProvisioningWithRuntime(tt.rules, tt.runtimeStateByTitle)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFilterByState(t *testing.T) {
	entries := []AlertListEntry{
		{Title: "Alert A", State: "firing"},
		{Title: "Alert B", State: "inactive"},
		{Title: "Alert C", State: "firing"},
		{Title: "Alert D", State: "pending"},
		{Title: "Alert E", State: "unknown"},
		{Title: "Alert F", State: "error"},
	}

	tests := []struct {
		name       string
		state      string
		wantTitles []string
	}{
		{
			name:       "filter firing",
			state:      "firing",
			wantTitles: []string{"Alert A", "Alert C"},
		},
		{
			name:       "filter inactive",
			state:      "inactive",
			wantTitles: []string{"Alert B"},
		},
		{
			name:       "filter pending",
			state:      "pending",
			wantTitles: []string{"Alert D"},
		},
		{
			name:       "filter error",
			state:      "error",
			wantTitles: []string{"Alert F"},
		},
		{
			name:       "filter unknown",
			state:      "unknown",
			wantTitles: []string{"Alert E"},
		},
		{
			name:       "filter case insensitive",
			state:      "FIRING",
			wantTitles: []string{"Alert A", "Alert C"},
		},
		{
			name:       "no matches",
			state:      "nonexistent",
			wantTitles: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterByState(entries, tt.state)

			gotTitles := make([]string, len(got))
			for i, e := range got {
				gotTitles[i] = e.Title
			}

			assert.Equal(t, tt.wantTitles, gotTitles)
		})
	}
}

func TestListOptsValidate(t *testing.T) {
	tests := []struct {
		name    string
		state   string
		wantErr string
	}{
		{
			name:    "empty state is valid",
			state:   "",
			wantErr: "",
		},
		{
			name:    "firing is valid",
			state:   "firing",
			wantErr: "",
		},
		{
			name:    "pending is valid",
			state:   "pending",
			wantErr: "",
		},
		{
			name:    "inactive is valid",
			state:   "inactive",
			wantErr: "",
		},
		{
			name:    "error is valid",
			state:   "error",
			wantErr: "",
		},
		{
			name:    "invalid state",
			state:   "bogus",
			wantErr: "invalid --state value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &listOpts{}
			flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
			opts.setup(flags)
			opts.State = tt.state

			err := opts.Validate()

			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
