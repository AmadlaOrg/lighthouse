package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/AmadlaOrg/lighthouse/alert"
	"github.com/AmadlaOrg/lighthouse/config"
	"github.com/AmadlaOrg/lighthouse/plugin"
	"github.com/AmadlaOrg/lighthouse/state"
)

// Result represents the outcome of processing an alert.
type Result struct {
	Fingerprint string `json:"fingerprint"`
	Action      string `json:"action"`
	Reason      string `json:"reason,omitempty"`
	Channel     string `json:"channel,omitempty"`
	Count       int    `json:"count,omitempty"`
}

// Engine defines the alert processing interface.
type Engine interface {
	Process(a *alert.Alert, now time.Time) (*Result, error)
	FlushGroups(now time.Time) ([]*Result, error)
}

type engine struct {
	cfg       *config.Config
	stateMgr  state.Manager
	pluginSvc plugin.Service
}

// New creates a new engine with the given configuration, state manager, and plugin service.
func New(cfg *config.Config, stateMgr state.Manager, pluginSvc plugin.Service) Engine {
	return &engine{
		cfg:       cfg,
		stateMgr:  stateMgr,
		pluginSvc: pluginSvc,
	}
}

// Process handles an incoming alert through the suppression pipeline.
func (e *engine) Process(a *alert.Alert, now time.Time) (*Result, error) {
	fingerprint := a.Fingerprint()

	alerts, err := e.stateMgr.LoadAlerts()
	if err != nil {
		return nil, fmt.Errorf("failed to load alerts: %w", err)
	}

	// Handle resolved status
	if a.Status == "resolved" {
		return e.handleResolved(fingerprint, a, alerts, now)
	}

	// Check silences
	silences, err := e.stateMgr.LoadSilences()
	if err != nil {
		return nil, fmt.Errorf("failed to load silences: %w", err)
	}
	if isSilenced(fingerprint, silences, now) {
		return &Result{Fingerprint: fingerprint, Action: "silenced"}, nil
	}

	existing := alerts[fingerprint]

	// Check flap detection
	if existing != nil && isFlapping(existing, e.cfg.FlapDetection.Window.Duration, e.cfg.FlapDetection.MaxTransitions) {
		existing.Flapping = true
		alerts[fingerprint] = existing
		_ = e.stateMgr.SaveAlerts(alerts)
		return &Result{Fingerprint: fingerprint, Action: "flapping"}, nil
	}

	// New alert
	if existing == nil {
		as := &state.AlertState{
			Fingerprint:  fingerprint,
			Source:       a.Source,
			Name:         a.Name,
			Severity:     a.Severity,
			Status:       "firing",
			FirstSeen:    now,
			LastSeen:     now,
			Count:        1,
			NextNotifyAt: now.Add(e.cfg.Backoff.Initial.Duration),
			BackoffStep:  0,
			Transitions:  []time.Time{now},
		}
		alerts[fingerprint] = as
		if err := e.stateMgr.SaveAlerts(alerts); err != nil {
			return nil, fmt.Errorf("failed to save alerts: %w", err)
		}

		// Check grouping
		if e.cfg.GroupWait.Duration > 0 {
			if err := e.addToGroup(a, as, now); err == nil {
				return &Result{Fingerprint: fingerprint, Action: "grouped", Count: 1}, nil
			}
		}

		// Deliver
		channel, err := e.deliver(a, as)
		if err != nil {
			return &Result{Fingerprint: fingerprint, Action: "delivery_failed", Reason: err.Error()}, nil
		}

		return &Result{Fingerprint: fingerprint, Action: "delivered", Channel: channel, Count: 1}, nil
	}

	// Existing alert - check dedup
	if isDeduplicated(existing, now, e.cfg.DedupWindow.Duration) {
		existing.Count++
		existing.LastSeen = now
		alerts[fingerprint] = existing
		_ = e.stateMgr.SaveAlerts(alerts)
		return &Result{Fingerprint: fingerprint, Action: "deduplicated", Count: existing.Count}, nil
	}

	// Check backoff
	if now.Before(existing.NextNotifyAt) {
		existing.Count++
		existing.LastSeen = now
		alerts[fingerprint] = existing
		_ = e.stateMgr.SaveAlerts(alerts)
		return &Result{Fingerprint: fingerprint, Action: "suppressed", Reason: "backoff", Count: existing.Count}, nil
	}

	// Check rate limits
	allLimited, err := e.allRateLimited(now)
	if err != nil {
		return nil, fmt.Errorf("failed to check rate limits: %w", err)
	}
	if allLimited {
		existing.Count++
		existing.LastSeen = now
		alerts[fingerprint] = existing
		_ = e.stateMgr.SaveAlerts(alerts)
		return &Result{Fingerprint: fingerprint, Action: "rate_limited", Count: existing.Count}, nil
	}

	// Check grouping
	if e.cfg.GroupWait.Duration > 0 {
		groups, _ := e.stateMgr.LoadGroups()
		for _, g := range groups {
			if g.GroupKey == a.GroupKey() && now.Before(g.FlushAfter) {
				if err := e.addToGroup(a, existing, now); err == nil {
					existing.Count++
					existing.LastSeen = now
					alerts[fingerprint] = existing
					_ = e.stateMgr.SaveAlerts(alerts)
					return &Result{Fingerprint: fingerprint, Action: "grouped", Count: existing.Count}, nil
				}
			}
		}
	}

	// Deliver
	existing.Count++
	existing.LastSeen = now
	existing.BackoffStep++
	existing.NextNotifyAt = computeNextNotify(e.cfg.Backoff, existing.BackoffStep, now)
	alerts[fingerprint] = existing
	if err := e.stateMgr.SaveAlerts(alerts); err != nil {
		return nil, fmt.Errorf("failed to save alerts: %w", err)
	}

	channel, err := e.deliver(a, existing)
	if err != nil {
		return &Result{Fingerprint: fingerprint, Action: "delivery_failed", Reason: err.Error(), Count: existing.Count}, nil
	}

	return &Result{Fingerprint: fingerprint, Action: "delivered", Channel: channel, Count: existing.Count}, nil
}

func (e *engine) handleResolved(fingerprint string, a *alert.Alert, alerts map[string]*state.AlertState, now time.Time) (*Result, error) {
	existing := alerts[fingerprint]
	if existing != nil {
		existing.Status = "resolved"
		existing.LastSeen = now
		existing.Transitions = append(existing.Transitions, now)
		alerts[fingerprint] = existing
		_ = e.stateMgr.SaveAlerts(alerts)
	}

	// Send resolve notification
	as := existing
	if as == nil {
		as = &state.AlertState{
			Fingerprint: fingerprint,
			Source:      a.Source,
			Name:        a.Name,
			Severity:    a.Severity,
			Status:      "resolved",
			FirstSeen:   now,
			LastSeen:    now,
			Count:       1,
		}
	}

	channel, err := e.deliver(a, as)
	if err != nil {
		return &Result{Fingerprint: fingerprint, Action: "delivery_failed", Reason: err.Error()}, nil
	}

	return &Result{Fingerprint: fingerprint, Action: "resolved", Channel: channel}, nil
}

// deliver sends the alert to available plugins and consumes a rate limit token.
func (e *engine) deliver(a *alert.Alert, as *state.AlertState) (string, error) {
	// Build enriched payload
	payload := map[string]any{
		"source":      a.Source,
		"name":        a.Name,
		"severity":    a.Severity,
		"labels":      a.Labels,
		"annotations": a.Annotations,
		"status":      a.Status,
		"fingerprint": as.Fingerprint,
		"count":       as.Count,
		"first_seen":  as.FirstSeen,
		"last_seen":   as.LastSeen,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Try each configured channel
	if len(e.cfg.Channels) == 0 {
		// No channels configured, try discovering plugins
		plugins, err := e.pluginSvc.Discover()
		if err != nil || len(plugins) == 0 {
			return "", fmt.Errorf("no notification channels configured and no plugins found")
		}
		for _, p := range plugins {
			var stdout, stderr bytes.Buffer
			code, err := e.pluginSvc.Exec(p, []string{"send"}, bytes.NewReader(data), &stdout, &stderr)
			if err == nil && code == 0 {
				_ = e.consumeRateToken(p, time.Now())
				return p, nil
			}
		}
		return "", fmt.Errorf("all plugins failed to deliver")
	}

	for _, ch := range e.cfg.Channels {
		var stdout, stderr bytes.Buffer
		code, err := e.pluginSvc.Exec(ch.Plugin, []string{"send"}, bytes.NewReader(data), &stdout, &stderr)
		if err == nil && code == 0 {
			_ = e.consumeRateToken(ch.Plugin, time.Now())
			return ch.Plugin, nil
		}
	}

	return "", fmt.Errorf("all configured channels failed to deliver")
}

func (e *engine) addToGroup(a *alert.Alert, as *state.AlertState, now time.Time) error {
	groups, err := e.stateMgr.LoadGroups()
	if err != nil {
		return err
	}

	payload := map[string]any{
		"source":      a.Source,
		"name":        a.Name,
		"severity":    a.Severity,
		"labels":      a.Labels,
		"annotations": a.Annotations,
		"status":      a.Status,
		"fingerprint": as.Fingerprint,
	}
	data, _ := json.Marshal(payload)

	groupKey := a.GroupKey()
	found := false
	for i, g := range groups {
		if g.GroupKey == groupKey {
			groups[i].Alerts = append(groups[i].Alerts, data)
			found = true
			break
		}
	}

	if !found {
		groups = append(groups, state.GroupBuffer{
			GroupKey:    groupKey,
			Alerts:     []json.RawMessage{data},
			FlushAfter: now.Add(e.cfg.GroupWait.Duration),
		})
	}

	return e.stateMgr.SaveGroups(groups)
}

// FlushGroups delivers any group buffers that are past their flush time.
func (e *engine) FlushGroups(now time.Time) ([]*Result, error) {
	groups, err := e.stateMgr.LoadGroups()
	if err != nil {
		return nil, fmt.Errorf("failed to load groups: %w", err)
	}

	var results []*Result
	var remaining []state.GroupBuffer

	for _, g := range groups {
		if now.After(g.FlushAfter) || now.Equal(g.FlushAfter) {
			// Deliver grouped alerts
			data, _ := json.Marshal(g.Alerts)
			delivered := false
			var channel string

			if len(e.cfg.Channels) > 0 {
				for _, ch := range e.cfg.Channels {
					var stdout, stderr bytes.Buffer
					code, err := e.pluginSvc.Exec(ch.Plugin, []string{"send"}, bytes.NewReader(data), &stdout, &stderr)
					if err == nil && code == 0 {
						channel = ch.Plugin
						delivered = true
						break
					}
				}
			} else {
				plugins, _ := e.pluginSvc.Discover()
				for _, p := range plugins {
					var stdout, stderr bytes.Buffer
					code, err := e.pluginSvc.Exec(p, []string{"send"}, bytes.NewReader(data), &stdout, &stderr)
					if err == nil && code == 0 {
						channel = p
						delivered = true
						break
					}
				}
			}

			action := "group_delivered"
			if !delivered {
				action = "group_delivery_failed"
			}

			results = append(results, &Result{
				Action:  action,
				Channel: channel,
				Count:   len(g.Alerts),
			})
		} else {
			remaining = append(remaining, g)
		}
	}

	if err := e.stateMgr.SaveGroups(remaining); err != nil {
		return results, fmt.Errorf("failed to save remaining groups: %w", err)
	}

	return results, nil
}

func (e *engine) allRateLimited(now time.Time) (bool, error) {
	if len(e.cfg.Channels) == 0 {
		return false, nil
	}

	limits, err := e.stateMgr.LoadRateLimits()
	if err != nil {
		return false, err
	}

	for _, ch := range e.cfg.Channels {
		rl := findRateLimit(limits, ch.Plugin)
		if rl == nil {
			return false, nil // no state yet = tokens available
		}
		refilled := refillTokens(rl, now)
		if refilled.Tokens >= 1 {
			return false, nil
		}
	}

	return true, nil
}

func (e *engine) consumeRateToken(pluginName string, now time.Time) error {
	limits, err := e.stateMgr.LoadRateLimits()
	if err != nil {
		return err
	}

	rl := findRateLimit(limits, pluginName)
	if rl == nil {
		// Find max_per_hour from config
		maxPerHour := 100 // default
		for _, ch := range e.cfg.Channels {
			if ch.Plugin == pluginName {
				maxPerHour = ch.MaxPerHour
				break
			}
		}
		limits = append(limits, state.RateLimitState{
			Plugin:     pluginName,
			Tokens:     float64(maxPerHour) - 1,
			LastRefill: now,
			MaxPerHour: maxPerHour,
		})
	} else {
		refilled := refillTokens(rl, now)
		refilled.Tokens--
		if refilled.Tokens < 0 {
			refilled.Tokens = 0
		}
		for i, l := range limits {
			if l.Plugin == pluginName {
				limits[i] = *refilled
				break
			}
		}
	}

	return e.stateMgr.SaveRateLimits(limits)
}
