package alert

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Alert represents a notification alert.
type Alert struct {
	Source      string            `json:"source" yaml:"source"`
	Name        string            `json:"name" yaml:"name"`
	Severity    string            `json:"severity" yaml:"severity"`
	Labels      map[string]string `json:"labels" yaml:"labels"`
	Annotations map[string]string `json:"annotations" yaml:"annotations"`
	Status      string            `json:"status" yaml:"status"`
}

// Fingerprint computes a unique SHA256 hash from source, name, and sorted labels.
func (a *Alert) Fingerprint() string {
	parts := []string{a.Source, a.Name}

	keys := make([]string, 0, len(a.Labels))
	for k := range a.Labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		parts = append(parts, k+"="+a.Labels[k])
	}

	h := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return fmt.Sprintf("%x", h)
}

// GroupKey returns a key for grouping alerts by source and name.
func (a *Alert) GroupKey() string {
	return a.Source + "|" + a.Name
}

// Parse parses alert data from JSON or YAML bytes.
func Parse(data []byte) (*Alert, error) {
	var a Alert
	if err := json.Unmarshal(data, &a); err != nil {
		if err := yaml.Unmarshal(data, &a); err != nil {
			return nil, fmt.Errorf("input is neither valid JSON nor YAML: %w", err)
		}
	}
	if a.Source == "" {
		return nil, fmt.Errorf("alert missing required field: source")
	}
	if a.Name == "" {
		return nil, fmt.Errorf("alert missing required field: name")
	}
	if a.Severity == "" {
		a.Severity = "info"
	}
	if a.Status == "" {
		a.Status = "firing"
	}
	return &a, nil
}

// ParseMultiple tries to parse data as a JSON array of alerts, falling back to single alert.
func ParseMultiple(data []byte) ([]*Alert, error) {
	var alerts []*Alert
	if err := json.Unmarshal(data, &alerts); err == nil && len(alerts) > 0 {
		return alerts, nil
	}

	a, err := Parse(data)
	if err != nil {
		return nil, err
	}
	return []*Alert{a}, nil
}
