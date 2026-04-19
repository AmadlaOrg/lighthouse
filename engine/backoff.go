package engine

import (
	"time"

	"github.com/AmadlaOrg/lighthouse/config"
)

// computeNextNotify calculates the next notification time based on the backoff config and step.
func computeNextNotify(cfg config.BackoffConfig, step int, now time.Time) time.Time {
	delay := cfg.Initial.Duration
	for i := 0; i < step; i++ {
		delay = time.Duration(float64(delay) * float64(cfg.Multiplier))
		if delay > cfg.Max.Duration {
			delay = cfg.Max.Duration
			break
		}
	}
	return now.Add(delay)
}
