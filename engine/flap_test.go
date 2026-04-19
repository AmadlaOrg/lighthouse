package engine

import (
	"testing"
	"time"

	"github.com/AmadlaOrg/lighthouse/state"
	"github.com/stretchr/testify/assert"
)

func TestIsFlapping_NotEnoughTransitions(t *testing.T) {
	existing := &state.AlertState{
		Transitions: []time.Time{time.Now(), time.Now()},
	}
	assert.False(t, isFlapping(existing, time.Hour, 5))
}

func TestIsFlapping_TooManyTransitions(t *testing.T) {
	now := time.Now()
	existing := &state.AlertState{
		Transitions: []time.Time{
			now.Add(-50 * time.Minute),
			now.Add(-40 * time.Minute),
			now.Add(-30 * time.Minute),
			now.Add(-20 * time.Minute),
			now.Add(-10 * time.Minute),
		},
	}
	assert.True(t, isFlapping(existing, time.Hour, 5))
}

func TestIsFlapping_TransitionsOutsideWindow(t *testing.T) {
	now := time.Now()
	existing := &state.AlertState{
		Transitions: []time.Time{
			now.Add(-5 * time.Hour),
			now.Add(-4 * time.Hour),
			now.Add(-3 * time.Hour),
			now.Add(-2 * time.Hour),
			now,
		},
	}
	assert.False(t, isFlapping(existing, time.Hour, 5))
}

func TestIsFlapping_NoTransitions(t *testing.T) {
	existing := &state.AlertState{}
	assert.False(t, isFlapping(existing, time.Hour, 5))
}
