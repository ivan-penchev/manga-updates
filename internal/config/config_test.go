package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_WithConfigFileAndExpansion(t *testing.T) {
	// Create a temporary config file
	configFileContent := `
notifier:
  smtp2go:
    api_key: "${TEST_SMTP_KEY}"
  recipient_email: "test@example.com"
series_data_folder: "from_file"
`
	tmpFile, err := os.CreateTemp("", "config_*.yaml")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Remove(tmpFile.Name())
	})

	_, err = tmpFile.WriteString(configFileContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// Unset interfering env vars
	interferingVars := []string{
		"NOTIFICATION_EMAIL_RECIPIENT",
		"SERIES_DATAFOLDER",
		"SMTP2GO_API_KEY",
	}
	for _, v := range interferingVars {
		originalVal, exists := os.LookupEnv(v)
		if exists {
			require.NoError(t, os.Unsetenv(v))
			t.Cleanup(func() {
				_ = os.Setenv(v, originalVal)
			})
		}
	}

	// Set environment variables
	t.Setenv("CONFIG_FILE", tmpFile.Name())
	t.Setenv("TEST_SMTP_KEY", "secret-key-123")
	t.Setenv("REMOTE_CHROME_URL", "ws://env-override:3000") // Set via ENV, not file

	// Run Load with nil
	cfg, err := Load("")
	require.NoError(t, err)

	// Verify
	assert.Equal(t, "secret-key-123", cfg.Notifier.SMTP2GO.APIKey, "Env var expansion in file should work")
	assert.Equal(t, "test@example.com", cfg.Notifier.RecipientEmail, "Value from file should be loaded")
	assert.Equal(t, "ws://env-override:3000", cfg.RemoteChromeURL, "Value from ENV should be loaded")
	assert.Equal(t, "from_file", cfg.SeriesDataFolder, "Value from file should override default")
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	// Create a temporary config file
	configFileContent := `
remote_chrome_url: "ws://from-file:3000"
`
	tmpFile, err := os.CreateTemp("", "config_*.yaml")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Remove(tmpFile.Name())
	})

	_, err = tmpFile.WriteString(configFileContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// Set environment variables
	t.Setenv("CONFIG_FILE", tmpFile.Name())
	t.Setenv("REMOTE_CHROME_URL", "ws://from-env:3000")

	// Run Load
	cfg, err := Load("")
	require.NoError(t, err)

	// Verify Env overrides File
	assert.Equal(t, "ws://from-env:3000", cfg.RemoteChromeURL, "ENV should override File")
}
