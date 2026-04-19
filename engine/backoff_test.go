package engine

import (
	"testing"
	"time"

	"github.com/AmadlaOrg/lighthouse/config"
	"github.com/stretchr/testify/assert"
)

func TestComputeNextNotify_Step0(t *testing.T) {
	cfg := config.BackoffConfig{
		Initial:    config.Duration{Duration: 5 * time.Minute},
		Multiplier: 3,
		Max:        config.Duration{Duration: 24 * time.Hour},
	}
	now := time.Now()
	next := computeNextNotify(cfg, 0, now)
	assert.Equal(t, now.Add(5*time.Minute), next)
}

func TestComputeNextNotify_Step1(t *testing.T) {
	cfg := config.BackoffConfig{
		Initial:    config.Duration{Duration: 5 * time.Minute},
		Multiplier: 3,
		Max:        config.Duration{Duration: 24 * time.Hour},
	}
	now := time.Now()
	next := computeNextNotify(cfg, 1, now)
	assert.Equal(t, now.Add(15*time.Minute), next)
}

func TestComputeNextNotify_Step2(t *testing.T) {
	cfg := config.BackoffConfig{
		Initial:    config.Duration{Duration: 5 * time.Minute},
		Multiplier: 3,
		Max:        config.Duration{Duration: 24 * time.Hour},
	}
	now := time.Now()
	next := computeNextNotify(cfg, 2, now)
	assert.Equal(t, now.Add(45*time.Minute), next)
}

func TestComputeNextNotify_CappedAtMax(t *testing.T) {
	cfg := config.BackoffConfig{
		Initial:    config.Duration{Duration: 5 * time.Minute},
		Multiplier: 3,
		Max:        config.Duration{Duration: 24 * time.Hour},
	}
	now := time.Now()
	next := computeNextNotify(cfg, 10, now)
	assert.Equal(t, now.Add(24*time.Hour), next)
}
