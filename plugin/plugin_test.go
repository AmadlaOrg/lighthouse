package plugin

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_Discover(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lighthouse-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	for _, name := range []string{"lighthouse-webhook", "lighthouse-slack", "other-tool"} {
		f, err := os.Create(filepath.Join(tmpDir, name))
		require.NoError(t, err)
		require.NoError(t, f.Chmod(0755))
		f.Close()
	}

	origGetenv := osGetenv
	defer func() { osGetenv = origGetenv }()
	osGetenv = func(key string) string {
		if key == "PATH" {
			return tmpDir
		}
		return ""
	}

	svc := &service{}
	plugins, err := svc.Discover()
	require.NoError(t, err)

	assert.Contains(t, plugins, "lighthouse-webhook")
	assert.Contains(t, plugins, "lighthouse-slack")
	assert.NotContains(t, plugins, "other-tool")
}

func TestService_DiscoverEmpty(t *testing.T) {
	origGetenv := osGetenv
	defer func() { osGetenv = origGetenv }()
	osGetenv = func(key string) string { return "" }

	svc := &service{}
	plugins, err := svc.Discover()
	require.NoError(t, err)
	assert.Nil(t, plugins)
}

func TestService_GetInfo(t *testing.T) {
	expectedInfo := Info{
		Name:        "lighthouse-webhook",
		Version:     "1.0.0",
		Channel:     "webhook",
		Description: "Sends notifications via HTTP webhook",
	}
	infoJSON, _ := json.Marshal(expectedInfo)

	origLookPath := execLookPath
	origCommand := execCommand
	defer func() {
		execLookPath = origLookPath
		execCommand = origCommand
	}()

	execLookPath = func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	}
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("echo", string(infoJSON))
	}

	svc := &service{}
	info, err := svc.GetInfo("lighthouse-webhook")
	require.NoError(t, err)
	assert.Equal(t, "lighthouse-webhook", info.Name)
	assert.Equal(t, "webhook", info.Channel)
}

func TestService_GetInfoNotFound(t *testing.T) {
	origLookPath := execLookPath
	defer func() { execLookPath = origLookPath }()
	execLookPath = func(name string) (string, error) {
		return "", exec.ErrNotFound
	}

	svc := &service{}
	_, err := svc.GetInfo("lighthouse-nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in PATH")
}

func TestService_Exec(t *testing.T) {
	origLookPath := execLookPath
	origCommand := execCommand
	defer func() {
		execLookPath = origLookPath
		execCommand = origCommand
	}()

	execLookPath = func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	}
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("echo", "hello")
	}

	svc := &service{}
	var stdout, stderr bytes.Buffer
	code, err := svc.Exec("lighthouse-webhook", []string{"send"}, nil, &stdout, &stderr)
	require.NoError(t, err)
	assert.Equal(t, 0, code)
	assert.Contains(t, stdout.String(), "hello")
}

func TestService_ExecNotFound(t *testing.T) {
	origLookPath := execLookPath
	defer func() { execLookPath = origLookPath }()
	execLookPath = func(name string) (string, error) {
		return "", exec.ErrNotFound
	}

	svc := &service{}
	var stdout, stderr bytes.Buffer
	code, err := svc.Exec("lighthouse-nonexistent", []string{"send"}, nil, &stdout, &stderr)
	assert.Error(t, err)
	assert.Equal(t, -1, code)
}

func TestService_ExecFailure(t *testing.T) {
	origLookPath := execLookPath
	origCommand := execCommand
	defer func() {
		execLookPath = origLookPath
		execCommand = origCommand
	}()

	execLookPath = func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	}
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	}

	svc := &service{}
	var stdout, stderr bytes.Buffer
	code, err := svc.Exec("lighthouse-webhook", []string{"send"}, nil, &stdout, &stderr)
	require.NoError(t, err)
	assert.Equal(t, 1, code)
}
