package config_test

import (
	"testing"

	"github.com/grafana/grafanactl/internal/config"
	"github.com/stretchr/testify/require"
)

func Test_SetValue(t *testing.T) {
	testCases := []struct {
		name           string
		input          config.Config
		path           string
		value          string
		expectedOutput config.Config
	}{
		{
			name:           "string",
			input:          config.Config{},
			path:           "current-context",
			value:          "ctx-name",
			expectedOutput: config.Config{CurrentContext: "ctx-name"},
		},
		{
			name:  "string in new context",
			input: config.Config{},
			path:  "contexts.new.grafana.server",
			value: "url",
			expectedOutput: config.Config{
				Contexts: map[string]*config.Context{
					"new": {
						Grafana: &config.GrafanaConfig{Server: "url"},
					},
				},
			},
		},
		{
			name: "string in existing context",
			input: config.Config{
				Contexts: map[string]*config.Context{
					"existing": {
						Grafana: &config.GrafanaConfig{Server: "url"},
					},
				},
			},
			path:  "contexts.existing.grafana.server",
			value: "new-url",
			expectedOutput: config.Config{
				Contexts: map[string]*config.Context{
					"existing": {
						Grafana: &config.GrafanaConfig{Server: "new-url"},
					},
				},
			},
		},
		{
			name:  "boolean in new context",
			input: config.Config{},
			path:  "contexts.new.grafana.insecure-skip-tls-verify",
			value: "true",
			expectedOutput: config.Config{
				Contexts: map[string]*config.Context{
					"new": {
						Grafana: &config.GrafanaConfig{InsecureSkipTLSVerify: true},
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			req := require.New(t)

			err := config.SetValue(&testCase.input, testCase.path, testCase.value)
			req.NoError(err)

			req.Equal(testCase.expectedOutput, testCase.input)
		})
	}
}

func Test_SetValue_withUnknownField(t *testing.T) {
	req := require.New(t)
	input := config.Config{}

	err := config.SetValue(&input, "unknown", "value")
	req.Error(err)
}

func Test_UnsetValue(t *testing.T) {
	testCases := []struct {
		name           string
		input          config.Config
		path           string
		expectedOutput config.Config
	}{
		{
			name:           "string",
			input:          config.Config{CurrentContext: "ctx-name"},
			path:           "current-context",
			expectedOutput: config.Config{},
		},
		{
			name: "map entry",
			input: config.Config{
				Contexts: map[string]*config.Context{
					"dev": {
						Grafana: &config.GrafanaConfig{Server: "dev-url"},
					},
					"prod": {
						Grafana: &config.GrafanaConfig{Server: "prod-url"},
					},
				},
			},
			path: "contexts.prod",
			expectedOutput: config.Config{
				Contexts: map[string]*config.Context{
					"dev": {
						Grafana: &config.GrafanaConfig{Server: "dev-url"},
					},
				},
			},
		},
		{
			name: "string in context",
			input: config.Config{
				Contexts: map[string]*config.Context{
					"existing": {
						Grafana: &config.GrafanaConfig{Server: "url", User: "user"},
					},
				},
			},
			path: "contexts.existing.grafana.user",
			expectedOutput: config.Config{
				Contexts: map[string]*config.Context{
					"existing": {
						Grafana: &config.GrafanaConfig{Server: "url"},
					},
				},
			},
		},
		{
			name: "boolean in new context",
			input: config.Config{
				Contexts: map[string]*config.Context{
					"existing": {
						Grafana: &config.GrafanaConfig{InsecureSkipTLSVerify: true},
					},
				},
			},
			path: "contexts.existing.grafana.insecure-skip-tls-verify",
			expectedOutput: config.Config{
				Contexts: map[string]*config.Context{
					"existing": {
						Grafana: &config.GrafanaConfig{InsecureSkipTLSVerify: false},
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			req := require.New(t)

			err := config.UnsetValue(&testCase.input, testCase.path)
			req.NoError(err)

			req.Equal(testCase.expectedOutput, testCase.input)
		})
	}
}

func Test_UnsetValue_withUnknownField(t *testing.T) {
	req := require.New(t)
	input := config.Config{}

	err := config.UnsetValue(&input, "unknown")
	req.Error(err)
}
