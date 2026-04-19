package engine

import (
	"time"

	"github.com/AmadlaOrg/lighthouse/state"
)

// isFlapping returns true if the alert has had too many status transitions within the window.
func isFlapping(existing *state.AlertState, window time.Duration, maxTransitions int) bool {
	if len(existing.Transitions) < maxTransitions {
		return false
	}

	now := existing.Transitions[len(existing.Transitions)-1]
	cutoff := now.Add(-window)

	count := 0
	for _, t := range existing.Transitions {
		if t.After(cutoff) || t.Equal(cutoff) {
			count++
		}
	}

	return count >= maxTransitions
}
