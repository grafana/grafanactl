package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/adrg/xdg"
	"github.com/grafana/grafanactl/internal/config"
	"github.com/grafana/grafanactl/internal/testutils"
	"github.com/stretchr/testify/require"
)

func TestLoad_explicitFile(t *testing.T) {
	req := require.New(t)

	cfg, err := config.Load(t.Context(), config.ExplicitConfigFile("./testdata/config.yaml"))
	req.NoError(err)

	req.Equal("local", cfg.CurrentContext)
	req.Len(cfg.Contexts, 1)
	req.Equal("local", cfg.Contexts["local"].Name)
	req.Equal("http://localhost:3000/", cfg.Contexts["local"].Grafana.Server)
}

func TestLoad_explicitFile_notFound(t *testing.T) {
	req := require.New(t)

	_, err := config.Load(t.Context(), config.ExplicitConfigFile("./testdata/does-not-exist.yaml"))
	req.Error(err)
	req.ErrorIs(err, os.ErrNotExist)
}

func TestLoad_standardLocation_noExistingConfig(t *testing.T) {
	req := require.New(t)

	fakeConfigDir := t.TempDir()

	t.Setenv("XDG_CONFIG_HOME", fakeConfigDir)

	// make sure the xdg library uses the new-fake env var we just set
	xdg.Reload()

	cfg, err := config.Load(t.Context(), config.StandardLocation())
	req.NoError(err)

	// An empty configuration is returned
	req.Equal("default", cfg.CurrentContext)
	req.Len(cfg.Contexts, 1)
}

func TestLoad_standardLocation_withExistingConfig(t *testing.T) {
	req := require.New(t)

	fakeConfigDir := t.TempDir()

	t.Setenv("XDG_CONFIG_HOME", fakeConfigDir)

	// create a barebones config file at the standard location
	err := os.MkdirAll(filepath.Join(fakeConfigDir, config.StandardConfigFolder), 0777)
	req.NoError(err)

	err = os.WriteFile(
		filepath.Join(fakeConfigDir, config.StandardConfigFolder, config.StandardConfigFileName),
		[]byte(`current-context: local`),
		0600,
	)
	req.NoError(err)

	// make sure the xdg library uses the new-fake env var we just set
	xdg.Reload()

	cfg, err := config.Load(t.Context(), config.StandardLocation())
	req.NoError(err)

	req.Equal("local", cfg.CurrentContext)
	req.Empty(cfg.Contexts)
}

func TestLoad_standardLocation_withEnvVar(t *testing.T) {
	req := require.New(t)

	// Set the environment variable to point to a test config
	t.Setenv(config.ConfigFileEnvVar, "./testdata/config.yaml")

	cfg, err := config.Load(t.Context(), config.StandardLocation())
	req.NoError(err)

	req.Equal("local", cfg.CurrentContext)
	req.Len(cfg.Contexts, 1)
	req.Equal("local", cfg.Contexts["local"].Name)
	req.Equal("http://localhost:3000/", cfg.Contexts["local"].Grafana.Server)
}

func TestLoad_standardLocation_envVarTakesPrecedence(t *testing.T) {
	req := require.New(t)

	fakeConfigDir := t.TempDir()

	t.Setenv("XDG_CONFIG_HOME", fakeConfigDir)

	// create a config file at the standard location with different content
	err := os.MkdirAll(filepath.Join(fakeConfigDir, config.StandardConfigFolder), 0777)
	req.NoError(err)

	err = os.WriteFile(
		filepath.Join(fakeConfigDir, config.StandardConfigFolder, config.StandardConfigFileName),
		[]byte(`current-context: standard-location`),
		0600,
	)
	req.NoError(err)

	// Set the environment variable to point to a different config
	t.Setenv(config.ConfigFileEnvVar, "./testdata/config.yaml")

	// make sure the xdg library uses the new-fake env var we just set
	xdg.Reload()

	cfg, err := config.Load(t.Context(), config.StandardLocation())
	req.NoError(err)

	// Should load from env var, not standard location
	req.Equal("local", cfg.CurrentContext)
	req.Len(cfg.Contexts, 1)
	req.Equal("http://localhost:3000/", cfg.Contexts["local"].Grafana.Server)
}

func TestLoad_withOverride(t *testing.T) {
	req := require.New(t)

	cfg, err := config.Load(t.Context(), config.ExplicitConfigFile("./testdata/config.yaml"), func(cfg *config.Config) error {
		cfg.CurrentContext = "overridden"
		return nil
	})
	req.NoError(err)

	req.Equal("overridden", cfg.CurrentContext)
	req.Len(cfg.Contexts, 1)
	req.Equal("http://localhost:3000/", cfg.Contexts["local"].Grafana.Server)
}

func TestLoad_withInvalidYaml(t *testing.T) {
	req := require.New(t)

	cfg := `current-context: local
this-field-is-invalid: []`

	configFile := testutils.CreateTempFile(t, cfg)

	_, err := config.Load(t.Context(), config.ExplicitConfigFile(configFile))
	req.Error(err)
	req.ErrorAs(err, &config.UnmarshalError{})
	req.ErrorContains(err, "unknown field \"this-field-is-invalid\"")
}

func TestWrite(t *testing.T) {
	req := require.New(t)

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	cfg := config.Config{
		CurrentContext: "local",
	}

	err := config.Write(t.Context(), config.ExplicitConfigFile(configFile), cfg)
	req.NoError(err)

	req.FileExists(configFile)
}
