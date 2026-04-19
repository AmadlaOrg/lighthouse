package engine

import (
	"time"

	"github.com/AmadlaOrg/lighthouse/state"
)

// findRateLimit finds the rate limit state for a given plugin.
func findRateLimit(limits []state.RateLimitState, plugin string) *state.RateLimitState {
	for i, l := range limits {
		if l.Plugin == plugin {
			return &limits[i]
		}
	}
	return nil
}

// refillTokens refills tokens based on elapsed time since last refill.
func refillTokens(rl *state.RateLimitState, now time.Time) *state.RateLimitState {
	elapsed := now.Sub(rl.LastRefill)
	tokensToAdd := elapsed.Hours() * float64(rl.MaxPerHour)
	rl.Tokens += tokensToAdd
	max := float64(rl.MaxPerHour)
	if rl.Tokens > max {
		rl.Tokens = max
	}
	rl.LastRefill = now
	return rl
}

// Allow returns true if the token bucket has at least one token available.
func Allow(rl *state.RateLimitState, now time.Time) bool {
	refilled := refillTokens(rl, now)
	return refilled.Tokens >= 1
}
