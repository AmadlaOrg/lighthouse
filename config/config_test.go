package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	assert.Equal(t, 5*time.Minute, cfg.DedupWindow.Duration)
	assert.Equal(t, 30*time.Second, cfg.GroupWait.Duration)
	assert.Equal(t, 5*time.Minute, cfg.GroupInterval.Duration)
	assert.Equal(t, 4*time.Hour, cfg.RepeatInterval.Duration)
	assert.Equal(t, 5*time.Minute, cfg.Backoff.Initial.Duration)
	assert.Equal(t, 3, cfg.Backoff.Multiplier)
	assert.Equal(t, 24*time.Hour, cfg.Backoff.Max.Duration)
	assert.Equal(t, time.Hour, cfg.FlapDetection.Window.Duration)
	assert.Equal(t, 5, cfg.FlapDetection.MaxTransitions)
}

func TestLoadFromPath_NotExists(t *testing.T) {
	cfg, err := LoadFromPath("/nonexistent/config.yaml")
	require.NoError(t, err)
	assert.Equal(t, 5*time.Minute, cfg.DedupWindow.Duration)
}

func TestLoadFromPath_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
dedup_window: 10m
group_wait: 1m
backoff:
  initial: 10m
  multiplier: 2
  max: 12h
flap_detection:
  window: 30m
  max_transitions: 3
channels:
  - plugin: lighthouse-webhook
    max_per_hour: 50
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	cfg, err := LoadFromPath(path)
	require.NoError(t, err)
	assert.Equal(t, 10*time.Minute, cfg.DedupWindow.Duration)
	assert.Equal(t, time.Minute, cfg.GroupWait.Duration)
	assert.Equal(t, 10*time.Minute, cfg.Backoff.Initial.Duration)
	assert.Equal(t, 2, cfg.Backoff.Multiplier)
	assert.Equal(t, 12*time.Hour, cfg.Backoff.Max.Duration)
	assert.Equal(t, 30*time.Minute, cfg.FlapDetection.Window.Duration)
	assert.Equal(t, 3, cfg.FlapDetection.MaxTransitions)
	require.Len(t, cfg.Channels, 1)
	assert.Equal(t, "lighthouse-webhook", cfg.Channels[0].Plugin)
	assert.Equal(t, 50, cfg.Channels[0].MaxPerHour)
}

func TestLoadFromPath_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(":::invalid"), 0o644))

	_, err := LoadFromPath(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config")
}

func TestDuration_UnmarshalYAML(t *testing.T) {
	type wrapper struct {
		D Duration `yaml:"d"`
	}

	tests := []struct {
		input    string
		expected time.Duration
	}{
		{`d: 5m`, 5 * time.Minute},
		{`d: 1h`, time.Hour},
		{`d: 30s`, 30 * time.Second},
		{`d: 24h`, 24 * time.Hour},
	}

	for _, tt := range tests {
		var w wrapper
		err := yaml.Unmarshal([]byte(tt.input), &w)
		require.NoError(t, err, "input: %s", tt.input)
		assert.Equal(t, tt.expected, w.D.Duration, "input: %s", tt.input)
	}
}

func TestDuration_UnmarshalYAML_Invalid(t *testing.T) {
	type wrapper struct {
		D Duration `yaml:"d"`
	}
	var w wrapper
	err := yaml.Unmarshal([]byte(`d: notaduration`), &w)
	assert.Error(t, err)
}

func TestLoad_NoHome(t *testing.T) {
	orig := userHomeDir
	defer func() { userHomeDir = orig }()
	userHomeDir = func() (string, error) {
		return "", os.ErrNotExist
	}

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, 5*time.Minute, cfg.DedupWindow.Duration)
}
