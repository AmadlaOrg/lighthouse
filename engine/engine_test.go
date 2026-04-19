package engine

import (
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/AmadlaOrg/lighthouse/alert"
	"github.com/AmadlaOrg/lighthouse/config"
	"github.com/AmadlaOrg/lighthouse/plugin"
	"github.com/AmadlaOrg/lighthouse/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPluginService struct{}

func (m *mockPluginService) Discover() ([]string, error) {
	return []string{"lighthouse-webhook"}, nil
}

func (m *mockPluginService) GetInfo(name string) (*plugin.Info, error) {
	return &plugin.Info{Name: name, Channel: "webhook"}, nil
}

func (m *mockPluginService) Exec(name string, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	result := map[string]any{"delivered": true, "channel": name}
	data, _ := json.Marshal(result)
	stdout.Write(data)
	return 0, nil
}

func newTestEngine(t *testing.T) (Engine, state.Manager) {
	t.Helper()
	cfg := config.Default()
	cfg.GroupWait.Duration = 0 // disable grouping for simpler tests
	stateMgr := state.NewWithPath(t.TempDir())
	pluginSvc := &mockPluginService{}
	return New(cfg, stateMgr, pluginSvc), stateMgr
}

func testAlert() *alert.Alert {
	return &alert.Alert{
		Source:      "waiter",
		Name:        "deploy_failed",
		Severity:    "critical",
		Labels:      map[string]string{"service": "api"},
		Annotations: map[string]string{"summary": "Deploy failed"},
		Status:      "firing",
	}
}

func TestEngine_Process_NewAlert(t *testing.T) {
	eng, _ := newTestEngine(t)
	a := testAlert()
	now := time.Now()

	result, err := eng.Process(a, now)
	require.NoError(t, err)
	assert.Equal(t, "delivered", result.Action)
	assert.Equal(t, 1, result.Count)
	assert.NotEmpty(t, result.Fingerprint)
}

func TestEngine_Process_Deduplicated(t *testing.T) {
	eng, _ := newTestEngine(t)
	a := testAlert()
	now := time.Now()

	_, err := eng.Process(a, now)
	require.NoError(t, err)

	// Second call within dedup window
	result, err := eng.Process(a, now.Add(2*time.Minute))
	require.NoError(t, err)
	assert.Equal(t, "deduplicated", result.Action)
	assert.Equal(t, 2, result.Count)
}

func TestEngine_Process_Silenced(t *testing.T) {
	eng, stateMgr := newTestEngine(t)
	a := testAlert()
	now := time.Now()
	fp := a.Fingerprint()

	// Add silence
	silences := []state.Silence{
		{Fingerprint: fp, CreatedAt: now, ExpiresAt: now.Add(time.Hour), Reason: "maintenance"},
	}
	require.NoError(t, stateMgr.SaveSilences(silences))

	result, err := eng.Process(a, now)
	require.NoError(t, err)
	assert.Equal(t, "silenced", result.Action)
}

func TestEngine_Process_Resolved(t *testing.T) {
	eng, _ := newTestEngine(t)
	a := testAlert()
	now := time.Now()

	// First fire it
	_, err := eng.Process(a, now)
	require.NoError(t, err)

	// Then resolve
	a.Status = "resolved"
	result, err := eng.Process(a, now.Add(time.Minute))
	require.NoError(t, err)
	assert.Equal(t, "resolved", result.Action)
}

func TestEngine_Process_Backoff(t *testing.T) {
	eng, _ := newTestEngine(t)
	a := testAlert()
	now := time.Now()

	// First delivery
	_, err := eng.Process(a, now)
	require.NoError(t, err)

	// After dedup window but past backoff (initial = 5m), at 5m30s
	result, err := eng.Process(a, now.Add(5*time.Minute+30*time.Second))
	require.NoError(t, err)
	// This is past both dedup and first backoff, should deliver again
	assert.Equal(t, "delivered", result.Action)
}

func TestEngine_FlushGroups_Empty(t *testing.T) {
	eng, _ := newTestEngine(t)
	results, err := eng.FlushGroups(time.Now())
	require.NoError(t, err)
	assert.Empty(t, results)
}
