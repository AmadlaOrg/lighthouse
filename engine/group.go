package engine

import (
	"time"

	"github.com/AmadlaOrg/lighthouse/state"
)

// shouldGroup returns true if a group buffer exists for the given key and hasn't been flushed.
func shouldGroup(groups []state.GroupBuffer, groupKey string, now time.Time) bool {
	for _, g := range groups {
		if g.GroupKey == groupKey && now.Before(g.FlushAfter) {
			return true
		}
	}
	return false
}
