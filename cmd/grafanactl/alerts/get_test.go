package alerts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetOptsValidate(t *testing.T) {
	tests := []struct {
		name    string
		opts    getOpts
		wantErr string
	}{
		{
			name: "valid json output format",
			opts: func() getOpts {
				o := getOpts{}
				o.IO.OutputFormat = "json"
				return o
			}(),
		},
		{
			name: "valid yaml output format",
			opts: func() getOpts {
				o := getOpts{}
				o.IO.OutputFormat = "yaml"
				return o
			}(),
		},
		{
			name: "invalid output format",
			opts: func() getOpts {
				o := getOpts{}
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

func TestGetCmdStructure(t *testing.T) {
	cmd := getCmd(nil)
	assert.Equal(t, "get UID", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	// Verify the output flag is registered
	f := cmd.Flags().Lookup("output")
	assert.NotNil(t, f)
	assert.Equal(t, "o", f.Shorthand)
}
