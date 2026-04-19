package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Info holds the metadata returned by a plugin's info command.
type Info struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Channel     string `json:"channel"`
	Description string `json:"description"`
}

// Service defines the plugin discovery and execution interface.
type Service interface {
	Discover() ([]string, error)
	GetInfo(pluginName string) (*Info, error)
	Exec(pluginName string, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error)
}

type service struct{}

var (
	execLookPath = exec.LookPath
	execCommand  = exec.Command
	osGetenv     = os.Getenv
)

// New creates a new plugin service.
func New() Service {
	return &service{}
}

// Discover scans PATH for lighthouse-* binaries and returns their names.
func (s *service) Discover() ([]string, error) {
	pathEnv := osGetenv("PATH")
	if pathEnv == "" {
		return nil, nil
	}

	seen := make(map[string]bool)
	var plugins []string

	for _, dir := range filepath.SplitList(pathEnv) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if strings.HasPrefix(name, "lighthouse-") && !seen[name] {
				fullPath := filepath.Join(dir, name)
				info, err := os.Stat(fullPath)
				if err != nil {
					continue
				}
				if info.Mode()&0111 != 0 {
					seen[name] = true
					plugins = append(plugins, name)
				}
			}
		}
	}

	return plugins, nil
}

type heryEnvelope struct {
	Type string          `json:"_type"`
	Body json.RawMessage `json:"_body"`
}

// GetInfo calls a plugin's info subcommand and parses the JSON response.
// It supports both HERY-wrapped and flat JSON responses.
func (s *service) GetInfo(pluginName string) (*Info, error) {
	path, err := execLookPath(pluginName)
	if err != nil {
		return nil, fmt.Errorf("plugin %s not found in PATH: %w", pluginName, err)
	}

	cmd := execCommand(path, "info", "-o", "json")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get info from %s: %w", pluginName, err)
	}

	var envelope heryEnvelope
	if err := json.Unmarshal(out, &envelope); err == nil && envelope.Type != "" && envelope.Body != nil {
		var info Info
		if err := json.Unmarshal(envelope.Body, &info); err != nil {
			return nil, fmt.Errorf("failed to parse HERY _body from %s: %w", pluginName, err)
		}
		return &info, nil
	}

	var info Info
	if err := json.Unmarshal(out, &info); err != nil {
		return nil, fmt.Errorf("failed to parse info from %s: %w", pluginName, err)
	}

	return &info, nil
}

// Exec runs an arbitrary subcommand on a plugin with the given args,
// pipes stdin/stdout/stderr, and returns the exit code.
func (s *service) Exec(pluginName string, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	path, err := execLookPath(pluginName)
	if err != nil {
		return -1, fmt.Errorf("plugin %s not found in PATH: %w", pluginName, err)
	}

	cmd := execCommand(path, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return -1, fmt.Errorf("plugin %s exec failed: %w", pluginName, err)
	}

	return 0, nil
}
