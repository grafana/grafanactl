package alerts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExportOptsValidate(t *testing.T) {
	tests := []struct {
		name    string
		opts    exportOpts
		wantErr string
	}{
		{
			name: "valid json format",
			opts: exportOpts{Format: "json"},
		},
		{
			name: "valid yaml format",
			opts: exportOpts{Format: "yaml"},
		},
		{
			name: "valid hcl format",
			opts: exportOpts{Format: "hcl"},
		},
		{
			name:    "invalid format",
			opts:    exportOpts{Format: "xml"},
			wantErr: "unsupported export format",
		},
		{
			name:    "empty format",
			opts:    exportOpts{Format: ""},
			wantErr: "unsupported export format",
		},
		{
			name:    "group without folder-uid",
			opts:    exportOpts{Format: "json", Group: "my-group"},
			wantErr: "--group requires --folder-uid",
		},
		{
			name:    "rule-uid with folder-uid",
			opts:    exportOpts{Format: "json", RuleUID: "rule-1", FolderUID: "folder-1"},
			wantErr: "--rule-uid cannot be combined with --folder-uid or --group",
		},
		{
			name:    "rule-uid with group",
			opts:    exportOpts{Format: "yaml", RuleUID: "rule-1", Group: "group-1", FolderUID: "folder-1"},
			wantErr: "--rule-uid cannot be combined with --folder-uid or --group",
		},
		{
			name:    "rule-uid with both folder-uid and group",
			opts:    exportOpts{Format: "json", RuleUID: "rule-1", FolderUID: "folder-1", Group: "group-1"},
			wantErr: "--rule-uid cannot be combined with --folder-uid or --group",
		},
		{
			name: "valid rule-uid alone",
			opts: exportOpts{Format: "json", RuleUID: "rule-1"},
		},
		{
			name: "valid folder-uid alone",
			opts: exportOpts{Format: "yaml", FolderUID: "folder-1"},
		},
		{
			name: "valid folder-uid with group",
			opts: exportOpts{Format: "hcl", FolderUID: "folder-1", Group: "group-1"},
		},
		{
			name: "no filter flags",
			opts: exportOpts{Format: "json"},
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

func TestExportCmdStructure(t *testing.T) {
	cmd := exportCmd(nil)
	assert.Equal(t, "export", cmd.Use)
	assert.NotEmpty(t, cmd.Short)

	// Verify flags are registered with correct defaults
	formatFlag := cmd.Flags().Lookup("format")
	assert.NotNil(t, formatFlag)
	assert.Equal(t, "yaml", formatFlag.DefValue)

	ruleUIDFlag := cmd.Flags().Lookup("rule-uid")
	assert.NotNil(t, ruleUIDFlag)
	assert.Equal(t, "", ruleUIDFlag.DefValue)

	folderUIDFlag := cmd.Flags().Lookup("folder-uid")
	assert.NotNil(t, folderUIDFlag)
	assert.Equal(t, "", folderUIDFlag.DefValue)

	groupFlag := cmd.Flags().Lookup("group")
	assert.NotNil(t, groupFlag)
	assert.Equal(t, "", groupFlag.DefValue)
}
