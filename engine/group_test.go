package engine

import (
	"testing"
	"time"

	"github.com/AmadlaOrg/lighthouse/state"
	"github.com/stretchr/testify/assert"
)

func TestShouldGroup_ActiveGroup(t *testing.T) {
	now := time.Now()
	groups := []state.GroupBuffer{
		{GroupKey: "test|alert", FlushAfter: now.Add(30 * time.Second)},
	}
	assert.True(t, shouldGroup(groups, "test|alert", now))
}

func TestShouldGroup_ExpiredGroup(t *testing.T) {
	now := time.Now()
	groups := []state.GroupBuffer{
		{GroupKey: "test|alert", FlushAfter: now.Add(-30 * time.Second)},
	}
	assert.False(t, shouldGroup(groups, "test|alert", now))
}

func TestShouldGroup_NoMatch(t *testing.T) {
	now := time.Now()
	groups := []state.GroupBuffer{
		{GroupKey: "other|alert", FlushAfter: now.Add(30 * time.Second)},
	}
	assert.False(t, shouldGroup(groups, "test|alert", now))
}

func TestShouldGroup_Empty(t *testing.T) {
	now := time.Now()
	assert.False(t, shouldGroup(nil, "test|alert", now))
}
