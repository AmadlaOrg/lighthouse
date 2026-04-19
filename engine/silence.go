package engine

import (
	"time"

	"github.com/AmadlaOrg/lighthouse/state"
)

// isSilenced returns true if the fingerprint matches an active silence.
func isSilenced(fingerprint string, silences []state.Silence, now time.Time) bool {
	for _, s := range silences {
		if s.Fingerprint == fingerprint && now.Before(s.ExpiresAt) {
			return true
		}
	}
	return false
}
