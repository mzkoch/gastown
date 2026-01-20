//go:build integration

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEnvironmentIsolation verifies that TestMain properly isolates
// the test environment by redirecting HOME and XDG directories.
func TestEnvironmentIsolation(t *testing.T) {
	// Verify HOME is redirected to a temp directory
	home := os.Getenv("HOME")
	if home == "" {
		t.Fatal("HOME is not set")
	}
	if !strings.Contains(home, "gt-integration-") {
		t.Errorf("HOME should be in test tmpdir, got: %s", home)
	}

	// Verify XDG_STATE_HOME is set
	stateHome := os.Getenv("XDG_STATE_HOME")
	if stateHome == "" {
		t.Fatal("XDG_STATE_HOME is not set")
	}
	if !strings.Contains(stateHome, "gt-integration-") {
		t.Errorf("XDG_STATE_HOME should be in test tmpdir, got: %s", stateHome)
	}

	// Verify XDG_CONFIG_HOME is set
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		t.Fatal("XDG_CONFIG_HOME is not set")
	}
	if !strings.Contains(configHome, "gt-integration-") {
		t.Errorf("XDG_CONFIG_HOME should be in test tmpdir, got: %s", configHome)
	}

	// Verify XDG_CACHE_HOME is set
	cacheHome := os.Getenv("XDG_CACHE_HOME")
	if cacheHome == "" {
		t.Fatal("XDG_CACHE_HOME is not set")
	}
	if !strings.Contains(cacheHome, "gt-integration-") {
		t.Errorf("XDG_CACHE_HOME should be in test tmpdir, got: %s", cacheHome)
	}

	// Verify directories exist
	for name, dir := range map[string]string{
		"HOME":              home,
		"XDG_STATE_HOME":    stateHome,
		"XDG_CONFIG_HOME":   configHome,
		"XDG_CACHE_HOME":    cacheHome,
	} {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("%s directory does not exist: %s", name, dir)
		}
	}

	// Verify all paths share the same base tmpdir
	basePath := filepath.Dir(filepath.Dir(home)) // go up two levels
	if !strings.HasPrefix(stateHome, basePath) {
		t.Errorf("XDG_STATE_HOME should be under %s", basePath)
	}
	if !strings.HasPrefix(configHome, basePath) {
		t.Errorf("XDG_CONFIG_HOME should be under %s", basePath)
	}
	if !strings.HasPrefix(cacheHome, basePath) {
		t.Errorf("XDG_CACHE_HOME should be under %s", basePath)
	}
}
