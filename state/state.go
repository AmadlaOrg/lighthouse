package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AlertState holds the state of a single alert by fingerprint.
type AlertState struct {
	Fingerprint  string      `json:"fingerprint"`
	Source       string      `json:"source"`
	Name         string      `json:"name"`
	Severity     string      `json:"severity"`
	Status       string      `json:"status"`
	FirstSeen    time.Time   `json:"first_seen"`
	LastSeen     time.Time   `json:"last_seen"`
	Count        int         `json:"count"`
	NextNotifyAt time.Time   `json:"next_notify_at"`
	BackoffStep  int         `json:"backoff_step"`
	Transitions  []time.Time `json:"transitions"`
	Flapping     bool        `json:"flapping"`
}

// Silence represents an active silence rule.
type Silence struct {
	Fingerprint string    `json:"fingerprint"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	Reason      string    `json:"reason"`
}

// GroupBuffer holds alerts waiting to be flushed as a group.
type GroupBuffer struct {
	GroupKey   string              `json:"group_key"`
	Alerts    []json.RawMessage   `json:"alerts"`
	FlushAfter time.Time          `json:"flush_after"`
}

// RateLimitState holds per-plugin rate limit state.
type RateLimitState struct {
	Plugin     string    `json:"plugin"`
	Tokens     float64   `json:"tokens"`
	LastRefill time.Time `json:"last_refill"`
	MaxPerHour int       `json:"max_per_hour"`
}

// Manager defines the interface for managing lighthouse state persistence.
type Manager interface {
	LoadAlerts() (map[string]*AlertState, error)
	SaveAlerts(alerts map[string]*AlertState) error
	LoadSilences() ([]Silence, error)
	SaveSilences(silences []Silence) error
	LoadGroups() ([]GroupBuffer, error)
	SaveGroups(groups []GroupBuffer) error
	LoadRateLimits() ([]RateLimitState, error)
	SaveRateLimits(limits []RateLimitState) error
}

type manager struct {
	stateDir string
	mu       sync.Mutex
}

// New creates a new state manager with the default state directory.
func New() Manager {
	return &manager{
		stateDir: defaultStateDir(),
	}
}

// NewWithPath creates a new state manager with a custom state directory.
func NewWithPath(dir string) Manager {
	return &manager{
		stateDir: dir,
	}
}

func defaultStateDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "lighthouse")
}

func (m *manager) ensureDir() error {
	return os.MkdirAll(m.stateDir, 0o755)
}

func (m *manager) loadJSON(filename string, v any) error {
	path := filepath.Join(m.stateDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // file doesn't exist = empty state
		}
		return fmt.Errorf("failed to read %s: %w", filename, err)
	}
	if len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to parse %s: %w", filename, err)
	}
	return nil
}

func (m *manager) saveJSON(filename string, v any) error {
	if err := m.ensureDir(); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal %s: %w", filename, err)
	}
	path := filepath.Join(m.stateDir, filename)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", filename, err)
	}
	return nil
}

func (m *manager) LoadAlerts() (map[string]*AlertState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	alerts := make(map[string]*AlertState)
	if err := m.loadJSON("alerts.json", &alerts); err != nil {
		return nil, err
	}
	return alerts, nil
}

func (m *manager) SaveAlerts(alerts map[string]*AlertState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveJSON("alerts.json", alerts)
}

func (m *manager) LoadSilences() ([]Silence, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var silences []Silence
	if err := m.loadJSON("silences.json", &silences); err != nil {
		return nil, err
	}
	return silences, nil
}

func (m *manager) SaveSilences(silences []Silence) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveJSON("silences.json", silences)
}

func (m *manager) LoadGroups() ([]GroupBuffer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var groups []GroupBuffer
	if err := m.loadJSON("groups.json", &groups); err != nil {
		return nil, err
	}
	return groups, nil
}

func (m *manager) SaveGroups(groups []GroupBuffer) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveJSON("groups.json", groups)
}

func (m *manager) LoadRateLimits() ([]RateLimitState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var limits []RateLimitState
	if err := m.loadJSON("rate_limits.json", &limits); err != nil {
		return nil, err
	}
	return limits, nil
}

func (m *manager) SaveRateLimits(limits []RateLimitState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveJSON("rate_limits.json", limits)
}
