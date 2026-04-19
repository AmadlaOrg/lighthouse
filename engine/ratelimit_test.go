package engine

import (
	"testing"
	"time"

	"github.com/AmadlaOrg/lighthouse/state"
	"github.com/stretchr/testify/assert"
)

func TestFindRateLimit_Found(t *testing.T) {
	limits := []state.RateLimitState{
		{Plugin: "lighthouse-webhook", Tokens: 50, MaxPerHour: 100},
		{Plugin: "lighthouse-slack", Tokens: 30, MaxPerHour: 60},
	}
	rl := findRateLimit(limits, "lighthouse-slack")
	assert.NotNil(t, rl)
	assert.Equal(t, "lighthouse-slack", rl.Plugin)
	assert.Equal(t, 30.0, rl.Tokens)
}

func TestFindRateLimit_NotFound(t *testing.T) {
	limits := []state.RateLimitState{
		{Plugin: "lighthouse-webhook", Tokens: 50, MaxPerHour: 100},
	}
	rl := findRateLimit(limits, "lighthouse-sms")
	assert.Nil(t, rl)
}

func TestRefillTokens(t *testing.T) {
	now := time.Now()
	rl := &state.RateLimitState{
		Plugin:     "lighthouse-webhook",
		Tokens:     50,
		LastRefill: now.Add(-30 * time.Minute),
		MaxPerHour: 100,
	}
	refilled := refillTokens(rl, now)
	assert.Equal(t, 100.0, refilled.Tokens) // 50 + 50 (half hour of 100/hr)
}

func TestRefillTokens_CappedAtMax(t *testing.T) {
	now := time.Now()
	rl := &state.RateLimitState{
		Plugin:     "lighthouse-webhook",
		Tokens:     90,
		LastRefill: now.Add(-1 * time.Hour),
		MaxPerHour: 100,
	}
	refilled := refillTokens(rl, now)
	assert.Equal(t, 100.0, refilled.Tokens) // capped at max
}

func TestAllow_HasTokens(t *testing.T) {
	now := time.Now()
	rl := &state.RateLimitState{
		Plugin:     "lighthouse-webhook",
		Tokens:     10,
		LastRefill: now,
		MaxPerHour: 100,
	}
	assert.True(t, Allow(rl, now))
}

func TestAllow_NoTokens(t *testing.T) {
	now := time.Now()
	rl := &state.RateLimitState{
		Plugin:     "lighthouse-webhook",
		Tokens:     0,
		LastRefill: now,
		MaxPerHour: 100,
	}
	assert.False(t, Allow(rl, now))
}
