package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_LoadAlerts_Empty(t *testing.T) {
	mgr := NewWithPath(t.TempDir())
	alerts, err := mgr.LoadAlerts()
	require.NoError(t, err)
	assert.Empty(t, alerts)
}

func TestManager_SaveAndLoadAlerts(t *testing.T) {
	mgr := NewWithPath(t.TempDir())

	now := time.Now().Truncate(time.Second)
	alerts := map[string]*AlertState{
		"abc123": {
			Fingerprint: "abc123",
			Source:      "waiter",
			Name:        "deploy_failed",
			Severity:    "critical",
			Status:      "firing",
			FirstSeen:   now,
			LastSeen:    now,
			Count:       3,
			BackoffStep: 1,
		},
	}

	err := mgr.SaveAlerts(alerts)
	require.NoError(t, err)

	loaded, err := mgr.LoadAlerts()
	require.NoError(t, err)
	require.Contains(t, loaded, "abc123")
	assert.Equal(t, "waiter", loaded["abc123"].Source)
	assert.Equal(t, 3, loaded["abc123"].Count)
	assert.Equal(t, 1, loaded["abc123"].BackoffStep)
}

func TestManager_SaveAndLoadSilences(t *testing.T) {
	mgr := NewWithPath(t.TempDir())

	now := time.Now().Truncate(time.Second)
	silences := []Silence{
		{
			Fingerprint: "abc123",
			CreatedAt:   now,
			ExpiresAt:   now.Add(2 * time.Hour),
			Reason:      "maintenance",
		},
	}

	err := mgr.SaveSilences(silences)
	require.NoError(t, err)

	loaded, err := mgr.LoadSilences()
	require.NoError(t, err)
	require.Len(t, loaded, 1)
	assert.Equal(t, "abc123", loaded[0].Fingerprint)
	assert.Equal(t, "maintenance", loaded[0].Reason)
}

func TestManager_LoadSilences_Empty(t *testing.T) {
	mgr := NewWithPath(t.TempDir())
	silences, err := mgr.LoadSilences()
	require.NoError(t, err)
	assert.Nil(t, silences)
}

func TestManager_SaveAndLoadGroups(t *testing.T) {
	mgr := NewWithPath(t.TempDir())

	alertJSON, _ := json.Marshal(map[string]string{"source": "test", "name": "alert"})
	groups := []GroupBuffer{
		{
			GroupKey:    "test|alert",
			Alerts:     []json.RawMessage{alertJSON},
			FlushAfter: time.Now().Add(30 * time.Second).Truncate(time.Second),
		},
	}

	err := mgr.SaveGroups(groups)
	require.NoError(t, err)

	loaded, err := mgr.LoadGroups()
	require.NoError(t, err)
	require.Len(t, loaded, 1)
	assert.Equal(t, "test|alert", loaded[0].GroupKey)
	assert.Len(t, loaded[0].Alerts, 1)
}

func TestManager_SaveAndLoadRateLimits(t *testing.T) {
	mgr := NewWithPath(t.TempDir())

	now := time.Now().Truncate(time.Second)
	limits := []RateLimitState{
		{
			Plugin:     "lighthouse-webhook",
			Tokens:     95.5,
			LastRefill: now,
			MaxPerHour: 100,
		},
	}

	err := mgr.SaveRateLimits(limits)
	require.NoError(t, err)

	loaded, err := mgr.LoadRateLimits()
	require.NoError(t, err)
	require.Len(t, loaded, 1)
	assert.Equal(t, "lighthouse-webhook", loaded[0].Plugin)
	assert.Equal(t, 95.5, loaded[0].Tokens)
	assert.Equal(t, 100, loaded[0].MaxPerHour)
}

func TestManager_LoadAlerts_CorruptFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "alerts.json"), []byte("not json"), 0o644))

	mgr := NewWithPath(dir)
	_, err := mgr.LoadAlerts()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

func TestManager_SaveAlerts_CreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "subdir", "lighthouse")
	mgr := NewWithPath(dir)

	err := mgr.SaveAlerts(map[string]*AlertState{})
	require.NoError(t, err)

	_, err = os.Stat(dir)
	assert.NoError(t, err)
}
