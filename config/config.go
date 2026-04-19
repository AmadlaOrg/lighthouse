package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration wraps time.Duration for YAML unmarshaling.
type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	d.Duration = parsed
	return nil
}

func (d Duration) MarshalYAML() (any, error) {
	return d.Duration.String(), nil
}

// BackoffConfig holds exponential backoff settings.
type BackoffConfig struct {
	Initial    Duration `yaml:"initial"`
	Multiplier int      `yaml:"multiplier"`
	Max        Duration `yaml:"max"`
}

// FlapConfig holds flap detection settings.
type FlapConfig struct {
	Window         Duration `yaml:"window"`
	MaxTransitions int      `yaml:"max_transitions"`
}

// ChannelConfig holds per-channel rate limit settings.
type ChannelConfig struct {
	Plugin     string `yaml:"plugin"`
	MaxPerHour int    `yaml:"max_per_hour"`
}

// Config holds the full lighthouse configuration.
type Config struct {
	DedupWindow    Duration        `yaml:"dedup_window"`
	GroupWait      Duration        `yaml:"group_wait"`
	GroupInterval  Duration        `yaml:"group_interval"`
	RepeatInterval Duration        `yaml:"repeat_interval"`
	Backoff        BackoffConfig   `yaml:"backoff"`
	FlapDetection  FlapConfig      `yaml:"flap_detection"`
	Channels       []ChannelConfig `yaml:"channels"`
}

var userHomeDir = os.UserHomeDir

// Default returns a Config with sensible defaults.
func Default() *Config {
	return &Config{
		DedupWindow:    Duration{5 * time.Minute},
		GroupWait:      Duration{30 * time.Second},
		GroupInterval:  Duration{5 * time.Minute},
		RepeatInterval: Duration{4 * time.Hour},
		Backoff: BackoffConfig{
			Initial:    Duration{5 * time.Minute},
			Multiplier: 3,
			Max:        Duration{24 * time.Hour},
		},
		FlapDetection: FlapConfig{
			Window:         Duration{1 * time.Hour},
			MaxTransitions: 5,
		},
		Channels: []ChannelConfig{},
	}
}

// Load reads the configuration file from ~/.config/lighthouse/config.yaml.
// If the file does not exist, it returns the default configuration.
func Load() (*Config, error) {
	home, err := userHomeDir()
	if err != nil {
		return Default(), nil
	}
	return LoadFromPath(filepath.Join(home, ".config", "lighthouse", "config.yaml"))
}

// LoadFromPath reads configuration from the given file path.
// If the file does not exist, it returns the default configuration.
func LoadFromPath(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg, nil
}
