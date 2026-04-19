package engine

import (
	"time"

	"github.com/AmadlaOrg/lighthouse/state"
)

// isDeduplicated returns true if the alert was seen within the dedup window.
func isDeduplicated(existing *state.AlertState, now time.Time, window time.Duration) bool {
	return now.Sub(existing.LastSeen) < window
}
