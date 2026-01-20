//go:build integration

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestMain provides environment isolation for integration tests.
// It ensures that tests do not interfere with the user's real Gas Town
// installation by redirecting XDG directories to a temporary location.
func TestMain(m *testing.M) {
	// Create temporary directories for test isolation
	tmpBase, err := os.MkdirTemp("", "gt-integration-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create test tmpdir: %v\n", err)
		os.Exit(1)
	}

	// Set up isolated XDG directories
	testHome := filepath.Join(tmpBase, "home")
	testState := filepath.Join(tmpBase, "state")
	testConfig := filepath.Join(tmpBase, "config")
	testCache := filepath.Join(tmpBase, "cache")

	for _, dir := range []string{testHome, testState, testConfig, testCache} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "create test dir %s: %v\n", dir, err)
			os.Exit(1)
		}
	}

	// Save original environment
	originalHome := os.Getenv("HOME")
	originalStateHome := os.Getenv("XDG_STATE_HOME")
	originalConfigHome := os.Getenv("XDG_CONFIG_HOME")
	originalCacheHome := os.Getenv("XDG_CACHE_HOME")

	// Set test environment
	_ = os.Setenv("HOME", testHome)
	_ = os.Setenv("XDG_STATE_HOME", testState)
	_ = os.Setenv("XDG_CONFIG_HOME", testConfig)
	_ = os.Setenv("XDG_CACHE_HOME", testCache)

	// Run tests
	code := m.Run()

	// Restore original environment
	_ = os.Setenv("HOME", originalHome)
	_ = os.Setenv("XDG_STATE_HOME", originalStateHome)
	_ = os.Setenv("XDG_CONFIG_HOME", originalConfigHome)
	_ = os.Setenv("XDG_CACHE_HOME", originalCacheHome)

	// Clean up
	_ = os.RemoveAll(tmpBase)

	os.Exit(code)
}
