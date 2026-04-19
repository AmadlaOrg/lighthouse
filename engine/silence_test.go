package engine

import (
	"testing"
	"time"

	"github.com/AmadlaOrg/lighthouse/state"
	"github.com/stretchr/testify/assert"
)

func TestIsSilenced_ActiveSilence(t *testing.T) {
	now := time.Now()
	silences := []state.Silence{
		{Fingerprint: "abc123", ExpiresAt: now.Add(time.Hour)},
	}
	assert.True(t, isSilenced("abc123", silences, now))
}

func TestIsSilenced_ExpiredSilence(t *testing.T) {
	now := time.Now()
	silences := []state.Silence{
		{Fingerprint: "abc123", ExpiresAt: now.Add(-time.Hour)},
	}
	assert.False(t, isSilenced("abc123", silences, now))
}

func TestIsSilenced_DifferentFingerprint(t *testing.T) {
	now := time.Now()
	silences := []state.Silence{
		{Fingerprint: "abc123", ExpiresAt: now.Add(time.Hour)},
	}
	assert.False(t, isSilenced("xyz789", silences, now))
}

func TestIsSilenced_NoSilences(t *testing.T) {
	now := time.Now()
	assert.False(t, isSilenced("abc123", nil, now))
}
