package alerts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchOptsValidate(t *testing.T) {
	tests := []struct {
		name    string
		opts    searchOpts
		wantErr string
	}{
		{
			name:    "empty name is invalid",
			opts:    searchOpts{Name: ""},
			wantErr: "--name flag is required",
		},
		{
			name: "valid name",
			opts: func() searchOpts {
				o := searchOpts{Name: "cpu"}
				o.IO.OutputFormat = "json"
				return o
			}(),
		},
		{
			name: "name with spaces is valid",
			opts: func() searchOpts {
				o := searchOpts{Name: "disk usage alert"}
				o.IO.OutputFormat = "json"
				return o
			}(),
		},
		{
			name: "invalid output format with valid name",
			opts: func() searchOpts {
				o := searchOpts{Name: "test"}
				o.IO.OutputFormat = "xml"
				return o
			}(),
			wantErr: "unknown output format",
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

func TestSearchCmdStructure(t *testing.T) {
	cmd := searchCmd(nil)
	assert.Equal(t, "search", cmd.Use)
	assert.NotEmpty(t, cmd.Short)

	// Verify --name flag is registered
	f := cmd.Flags().Lookup("name")
	assert.NotNil(t, f)
	assert.NotEmpty(t, f.Usage)

	// Verify --output flag is registered
	o := cmd.Flags().Lookup("output")
	assert.NotNil(t, o)
}
