package engine

import (
	"testing"
	"time"

	"github.com/AmadlaOrg/lighthouse/state"
	"github.com/stretchr/testify/assert"
)

func TestIsDeduplicated_WithinWindow(t *testing.T) {
	now := time.Now()
	existing := &state.AlertState{LastSeen: now.Add(-2 * time.Minute)}
	assert.True(t, isDeduplicated(existing, now, 5*time.Minute))
}

func TestIsDeduplicated_OutsideWindow(t *testing.T) {
	now := time.Now()
	existing := &state.AlertState{LastSeen: now.Add(-10 * time.Minute)}
	assert.False(t, isDeduplicated(existing, now, 5*time.Minute))
}

func TestIsDeduplicated_ExactBoundary(t *testing.T) {
	now := time.Now()
	existing := &state.AlertState{LastSeen: now.Add(-5 * time.Minute)}
	assert.False(t, isDeduplicated(existing, now, 5*time.Minute))
}
