package alert

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAlert_Fingerprint(t *testing.T) {
	a := &Alert{
		Source: "waiter",
		Name:   "deploy_failed",
		Labels: map[string]string{
			"service":  "api",
			"strategy": "blue-green",
		},
	}
	fp := a.Fingerprint()
	assert.Len(t, fp, 64)

	// Same alert should produce same fingerprint
	a2 := &Alert{
		Source: "waiter",
		Name:   "deploy_failed",
		Labels: map[string]string{
			"strategy": "blue-green",
			"service":  "api",
		},
	}
	assert.Equal(t, fp, a2.Fingerprint())
}

func TestAlert_Fingerprint_DifferentLabels(t *testing.T) {
	a1 := &Alert{Source: "test", Name: "alert", Labels: map[string]string{"a": "1"}}
	a2 := &Alert{Source: "test", Name: "alert", Labels: map[string]string{"a": "2"}}
	assert.NotEqual(t, a1.Fingerprint(), a2.Fingerprint())
}

func TestAlert_Fingerprint_NoLabels(t *testing.T) {
	a := &Alert{Source: "test", Name: "alert"}
	fp := a.Fingerprint()
	assert.Len(t, fp, 64)
}

func TestAlert_GroupKey(t *testing.T) {
	a := &Alert{Source: "waiter", Name: "deploy_failed"}
	assert.Equal(t, "waiter|deploy_failed", a.GroupKey())
}

func TestParse_JSON(t *testing.T) {
	data := []byte(`{"source":"waiter","name":"deploy_failed","severity":"critical","labels":{"svc":"api"},"annotations":{"summary":"fail"},"status":"firing"}`)
	a, err := Parse(data)
	require.NoError(t, err)
	assert.Equal(t, "waiter", a.Source)
	assert.Equal(t, "deploy_failed", a.Name)
	assert.Equal(t, "critical", a.Severity)
	assert.Equal(t, "firing", a.Status)
	assert.Equal(t, "api", a.Labels["svc"])
	assert.Equal(t, "fail", a.Annotations["summary"])
}

func TestParse_YAML(t *testing.T) {
	data := []byte("source: waiter\nname: deploy_failed\nseverity: warning\nstatus: firing\n")
	a, err := Parse(data)
	require.NoError(t, err)
	assert.Equal(t, "waiter", a.Source)
	assert.Equal(t, "warning", a.Severity)
}

func TestParse_Defaults(t *testing.T) {
	data := []byte(`{"source":"test","name":"alert"}`)
	a, err := Parse(data)
	require.NoError(t, err)
	assert.Equal(t, "info", a.Severity)
	assert.Equal(t, "firing", a.Status)
}

func TestParse_MissingSource(t *testing.T) {
	data := []byte(`{"name":"alert"}`)
	_, err := Parse(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source")
}

func TestParse_MissingName(t *testing.T) {
	data := []byte(`{"source":"test"}`)
	_, err := Parse(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

func TestParse_InvalidInput(t *testing.T) {
	data := []byte(`not valid`)
	_, err := Parse(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "neither valid JSON nor YAML")
}

func TestParseMultiple_Array(t *testing.T) {
	data := []byte(`[{"source":"a","name":"b"},{"source":"c","name":"d"}]`)
	alerts, err := ParseMultiple(data)
	require.NoError(t, err)
	assert.Len(t, alerts, 2)
}

func TestParseMultiple_Single(t *testing.T) {
	data := []byte(`{"source":"a","name":"b"}`)
	alerts, err := ParseMultiple(data)
	require.NoError(t, err)
	assert.Len(t, alerts, 1)
}
