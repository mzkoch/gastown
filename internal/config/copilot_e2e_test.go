//go:build integration

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/steveyegge/gastown/internal/copilot"
)

// TestCopilotTrustE2E_ConfigCreation verifies that the Copilot config file
// is correctly created when it doesn't exist.
func TestCopilotTrustE2E_ConfigCreation(t *testing.T) {
	
	tmpDir := t.TempDir()

	xdgHome := filepath.Join(tmpDir, "xdg")
	t.Setenv("XDG_CONFIG_HOME", xdgHome)

	townRoot, rigPath := setupTestTown(t, tmpDir, "copilot")
	witnessDir := filepath.Join(rigPath, "witness")
	if err := os.MkdirAll(witnessDir, 0755); err != nil {
		t.Fatalf("Failed to create witness dir: %v", err)
	}

	// Call EnsureCopilotTrustedFolder
	if err := EnsureCopilotTrustedFolder(CopilotTrustConfig{
		Role:     "witness",
		TownRoot: townRoot,
		RigPath:  rigPath,
		WorkDir:  witnessDir,
	}); err != nil {
		t.Fatalf("EnsureCopilotTrustedFolder failed: %v", err)
	}

	// Verify config file was created
	copilotConfig := filepath.Join(xdgHome, ".copilot", "config.json")
	if _, err := os.Stat(copilotConfig); os.IsNotExist(err) {
		t.Fatal("Expected Copilot config.json to be created")
	}

	// Verify contents
	data, err := os.ReadFile(copilotConfig)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	trusted, ok := cfg["trusted_folders"]
	if !ok {
		t.Fatal("Expected trusted_folders in config")
	}

	folders, ok := trusted.([]interface{})
	if !ok {
		t.Fatalf("Expected trusted_folders to be array, got %T", trusted)
	}

	if len(folders) == 0 {
		t.Fatal("Expected at least one trusted folder")
	}

	foundWitness := false
	for _, f := range folders {
		if str, ok := f.(string); ok && str == witnessDir {
			foundWitness = true
			break
		}
	}

	if !foundWitness {
		t.Errorf("Expected witness dir %q in trusted_folders, got: %v", witnessDir, folders)
	}
}

// TestCopilotTrustE2E_NoDuplicates verifies that calling trust setup
// multiple times doesn't create duplicate entries.
func TestCopilotTrustE2E_NoDuplicates(t *testing.T) {
	
	tmpDir := t.TempDir()

	xdgHome := filepath.Join(tmpDir, "xdg")
	t.Setenv("XDG_CONFIG_HOME", xdgHome)

	townRoot, rigPath := setupTestTown(t, tmpDir, "copilot")
	witnessDir := filepath.Join(rigPath, "witness")
	if err := os.MkdirAll(witnessDir, 0755); err != nil {
		t.Fatalf("Failed to create witness dir: %v", err)
	}

	cfg := CopilotTrustConfig{
		Role:     "witness",
		TownRoot: townRoot,
		RigPath:  rigPath,
		WorkDir:  witnessDir,
	}

	// Call multiple times
	for i := 0; i < 3; i++ {
		if err := EnsureCopilotTrustedFolder(cfg); err != nil {
			t.Fatalf("Call %d failed: %v", i+1, err)
		}
	}

	// Verify only one entry exists
	copilotConfig := filepath.Join(xdgHome, ".copilot", "config.json")
	data, err := os.ReadFile(copilotConfig)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var configMap map[string]interface{}
	if err := json.Unmarshal(data, &configMap); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	folders, ok := configMap["trusted_folders"].([]interface{})
	if !ok {
		t.Fatal("Expected trusted_folders array")
	}

	count := 0
	for _, f := range folders {
		if str, ok := f.(string); ok && str == witnessDir {
			count++
		}
	}

	if count != 1 {
		t.Errorf("Expected exactly 1 entry for %q, found %d. Folders: %v", witnessDir, count, folders)
	}
}

// TestCopilotTrustE2E_AllRoles verifies that all agent startup paths
// properly trust their working directories.
func TestCopilotTrustE2E_AllRoles(t *testing.T) {
	

	tests := []struct {
		role       string
		setupDir   func(townRoot, rigPath string) (string, error)
		expectDir  func(townRoot, rigPath string) string
		skipReason string
	}{
		{
			role: "witness",
			setupDir: func(townRoot, rigPath string) (string, error) {
				witnessDir := filepath.Join(rigPath, "witness")
				return witnessDir, os.MkdirAll(witnessDir, 0755)
			},
			expectDir: func(townRoot, rigPath string) string {
				return filepath.Join(rigPath, "witness")
			},
		},
		{
			role: "refinery",
			setupDir: func(townRoot, rigPath string) (string, error) {
				refineryDir := filepath.Join(rigPath, "refinery", "rig")
				return refineryDir, os.MkdirAll(refineryDir, 0755)
			},
			expectDir: func(townRoot, rigPath string) string {
				return filepath.Join(rigPath, "refinery", "rig")
			},
		},
		{
			role: "polecat",
			setupDir: func(townRoot, rigPath string) (string, error) {
				polecatsDir := filepath.Join(rigPath, "polecats")
				polecatWorkDir := filepath.Join(polecatsDir, "capable", "testrig")
				return polecatWorkDir, os.MkdirAll(polecatWorkDir, 0755)
			},
			expectDir: func(townRoot, rigPath string) string {
				// For polecats, we expect the parent polecats/ dir to be trusted
				return filepath.Join(rigPath, "polecats")
			},
		},
		{
			role: "crew",
			setupDir: func(townRoot, rigPath string) (string, error) {
				crewDir := filepath.Join(rigPath, "crew", "worker1", "rig")
				return crewDir, os.MkdirAll(crewDir, 0755)
			},
			expectDir: func(townRoot, rigPath string) string {
				return filepath.Join(rigPath, "crew", "worker1", "rig")
			},
		},
		{
			role: "mayor",
			setupDir: func(townRoot, rigPath string) (string, error) {
				mayorDir := filepath.Join(townRoot, "mayor")
				return mayorDir, os.MkdirAll(mayorDir, 0755)
			},
			expectDir: func(townRoot, rigPath string) string {
				return filepath.Join(townRoot, "mayor")
			},
		},
		{
			role: "deacon",
			setupDir: func(townRoot, rigPath string) (string, error) {
				deaconDir := filepath.Join(townRoot, "deacon")
				return deaconDir, os.MkdirAll(deaconDir, 0755)
			},
			expectDir: func(townRoot, rigPath string) string {
				return filepath.Join(townRoot, "deacon")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.role, func(t *testing.T) {
			
			if tt.skipReason != "" {
				t.Skip(tt.skipReason)
			}

			tmpDir := t.TempDir()
			xdgHome := filepath.Join(tmpDir, "xdg")
			t.Setenv("XDG_CONFIG_HOME", xdgHome)

			townRoot, rigPath := setupTestTown(t, tmpDir, "copilot")

			workDir, err := tt.setupDir(townRoot, rigPath)
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			if err := EnsureCopilotTrustedFolder(CopilotTrustConfig{
				Role:     tt.role,
				TownRoot: townRoot,
				RigPath:  rigPath,
				WorkDir:  workDir,
			}); err != nil {
				t.Fatalf("EnsureCopilotTrustedFolder failed: %v", err)
			}

			// Verify expected directory is trusted
			expectedDir := tt.expectDir(townRoot, rigPath)
			copilotConfig := filepath.Join(xdgHome, ".copilot", "config.json")

			data, err := os.ReadFile(copilotConfig)
			if err != nil {
				t.Fatalf("Failed to read config: %v", err)
			}

			if !strings.Contains(string(data), expectedDir) {
				t.Errorf("Expected %q in trusted_folders. Config: %s", expectedDir, string(data))
			}

			t.Logf("✓ %s role correctly trusts %s", tt.role, expectedDir)
		})
	}
}

// TestCopilotTrustE2E_PolecatParentTrust specifically validates that
// polecats trust the polecats/ parent directory, not individual worktrees.
func TestCopilotTrustE2E_PolecatParentTrust(t *testing.T) {
	
	tmpDir := t.TempDir()

	xdgHome := filepath.Join(tmpDir, "xdg")
	t.Setenv("XDG_CONFIG_HOME", xdgHome)

	townRoot, rigPath := setupTestTown(t, tmpDir, "copilot")

	polecatsDir := filepath.Join(rigPath, "polecats")
	polecat1WorkDir := filepath.Join(polecatsDir, "capable", "testrig")
	polecat2WorkDir := filepath.Join(polecatsDir, "furiosa", "testrig")

	if err := os.MkdirAll(polecat1WorkDir, 0755); err != nil {
		t.Fatalf("Failed to create polecat1 dir: %v", err)
	}
	if err := os.MkdirAll(polecat2WorkDir, 0755); err != nil {
		t.Fatalf("Failed to create polecat2 dir: %v", err)
	}

	// Trust first polecat
	if err := EnsureCopilotTrustedFolder(CopilotTrustConfig{
		Role:     "polecat",
		TownRoot: townRoot,
		RigPath:  rigPath,
		WorkDir:  polecat1WorkDir,
	}); err != nil {
		t.Fatalf("First trust failed: %v", err)
	}

	// Trust second polecat
	if err := EnsureCopilotTrustedFolder(CopilotTrustConfig{
		Role:     "polecat",
		TownRoot: townRoot,
		RigPath:  rigPath,
		WorkDir:  polecat2WorkDir,
	}); err != nil {
		t.Fatalf("Second trust failed: %v", err)
	}

	// Verify polecats/ parent is trusted, not individual worktrees
	copilotConfig := filepath.Join(xdgHome, ".copilot", "config.json")
	data, err := os.ReadFile(copilotConfig)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	configStr := string(data)

	// Should contain polecats parent
	if !strings.Contains(configStr, polecatsDir) {
		t.Errorf("Expected polecats dir %q in config. Got: %s", polecatsDir, configStr)
	}

	// Should NOT contain specific worktrees
	if strings.Contains(configStr, polecat1WorkDir) {
		t.Errorf("Should not contain polecat1 worktree %q. Got: %s", polecat1WorkDir, configStr)
	}
	if strings.Contains(configStr, polecat2WorkDir) {
		t.Errorf("Should not contain polecat2 worktree %q. Got: %s", polecat2WorkDir, configStr)
	}

	// Parse and verify exactly one entry (the parent)
	var configMap map[string]interface{}
	if err := json.Unmarshal(data, &configMap); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	folders, ok := configMap["trusted_folders"].([]interface{})
	if !ok {
		t.Fatal("Expected trusted_folders array")
	}

	if len(folders) != 1 {
		t.Errorf("Expected exactly 1 trusted folder (polecats parent), got %d: %v", len(folders), folders)
	}
}

// TestCopilotTrustE2E_NonCopilotAgent verifies that when the agent is NOT
// Copilot, the config file is not created or modified.
func TestCopilotTrustE2E_NonCopilotAgent(t *testing.T) {
	

	agents := []string{"claude", "gemini", "codex", "cursor", "auggie", "amp"}

	for _, agent := range agents {
		agent := agent
		t.Run(agent, func(t *testing.T) {
			
			tmpDir := t.TempDir()

			xdgHome := filepath.Join(tmpDir, "xdg")
			t.Setenv("XDG_CONFIG_HOME", xdgHome)

			townRoot, rigPath := setupTestTown(t, tmpDir, agent)
			witnessDir := filepath.Join(rigPath, "witness")
			if err := os.MkdirAll(witnessDir, 0755); err != nil {
				t.Fatalf("Failed to create witness dir: %v", err)
			}

			if err := EnsureCopilotTrustedFolder(CopilotTrustConfig{
				Role:     "witness",
				TownRoot: townRoot,
				RigPath:  rigPath,
				WorkDir:  witnessDir,
			}); err != nil {
				t.Fatalf("EnsureCopilotTrustedFolder failed: %v", err)
			}

			// Verify config file was NOT created
			copilotConfig := filepath.Join(xdgHome, ".copilot", "config.json")
			if _, err := os.Stat(copilotConfig); !os.IsNotExist(err) {
				t.Errorf("Expected no Copilot config for agent %q, but file exists", agent)
			}

			t.Logf("✓ %s agent correctly skips Copilot config", agent)
		})
	}
}

// TestCopilotTrustE2E_AgentOverride verifies that AgentOverride is respected.
func TestCopilotTrustE2E_AgentOverride(t *testing.T) {
	

	t.Run("OverrideToCopilot", func(t *testing.T) {
		tmpDir := t.TempDir()
		xdgHome := filepath.Join(tmpDir, "xdg")
		t.Setenv("XDG_CONFIG_HOME", xdgHome)

		// Default agent is claude
		townRoot, rigPath := setupTestTown(t, tmpDir, "claude")
		witnessDir := filepath.Join(rigPath, "witness")
		if err := os.MkdirAll(witnessDir, 0755); err != nil {
			t.Fatalf("Failed to create witness dir: %v", err)
		}

		// But override to copilot
		if err := EnsureCopilotTrustedFolder(CopilotTrustConfig{
			Role:          "witness",
			TownRoot:      townRoot,
			RigPath:       rigPath,
			WorkDir:       witnessDir,
			AgentOverride: "copilot",
		}); err != nil {
			t.Fatalf("EnsureCopilotTrustedFolder failed: %v", err)
		}

		// Should create Copilot config
		copilotConfig := filepath.Join(xdgHome, ".copilot", "config.json")
		if _, err := os.Stat(copilotConfig); os.IsNotExist(err) {
			t.Error("Expected Copilot config to be created with override")
		}
	})

	t.Run("OverrideToNonCopilot", func(t *testing.T) {
		tmpDir := t.TempDir()
		xdgHome := filepath.Join(tmpDir, "xdg")
		t.Setenv("XDG_CONFIG_HOME", xdgHome)

		// Default agent is copilot
		townRoot, rigPath := setupTestTown(t, tmpDir, "copilot")
		witnessDir := filepath.Join(rigPath, "witness")
		if err := os.MkdirAll(witnessDir, 0755); err != nil {
			t.Fatalf("Failed to create witness dir: %v", err)
		}

		// But override to claude
		if err := EnsureCopilotTrustedFolder(CopilotTrustConfig{
			Role:          "witness",
			TownRoot:      townRoot,
			RigPath:       rigPath,
			WorkDir:       witnessDir,
			AgentOverride: "claude",
		}); err != nil {
			t.Fatalf("EnsureCopilotTrustedFolder failed: %v", err)
		}

		// Should NOT create Copilot config
		copilotConfig := filepath.Join(xdgHome, ".copilot", "config.json")
		if _, err := os.Stat(copilotConfig); !os.IsNotExist(err) {
			t.Error("Expected no Copilot config with claude override")
		}
	})
}

// TestCopilotTrustE2E_EmptyWorkDir verifies that empty WorkDir is handled gracefully.
func TestCopilotTrustE2E_EmptyWorkDir(t *testing.T) {
	
	tmpDir := t.TempDir()

	xdgHome := filepath.Join(tmpDir, "xdg")
	t.Setenv("XDG_CONFIG_HOME", xdgHome)

	townRoot, rigPath := setupTestTown(t, tmpDir, "copilot")

	// Call with empty WorkDir
	if err := EnsureCopilotTrustedFolder(CopilotTrustConfig{
		Role:     "witness",
		TownRoot: townRoot,
		RigPath:  rigPath,
		WorkDir:  "", // Empty!
	}); err != nil {
		t.Fatalf("EnsureCopilotTrustedFolder failed with empty WorkDir: %v", err)
	}

	// Should not create config file
	copilotConfig := filepath.Join(xdgHome, ".copilot", "config.json")
	if _, err := os.Stat(copilotConfig); !os.IsNotExist(err) {
		t.Error("Expected no Copilot config with empty WorkDir")
	}
}

// TestCopilotTrustE2E_DirectAPIUsage tests the lower-level copilot.EnsureTrustedFolder directly.
func TestCopilotTrustE2E_DirectAPIUsage(t *testing.T) {
	
	tmpDir := t.TempDir()

	xdgHome := filepath.Join(tmpDir, "xdg")
	t.Setenv("XDG_CONFIG_HOME", xdgHome)

	testPath := filepath.Join(tmpDir, "test-path")
	if err := os.MkdirAll(testPath, 0755); err != nil {
		t.Fatalf("Failed to create test path: %v", err)
	}

	// First call should update
	updated, err := copilot.EnsureTrustedFolder(testPath)
	if err != nil {
		t.Fatalf("First call failed: %v", err)
	}
	if !updated {
		t.Error("Expected first call to return updated=true")
	}

	// Second call should not update (already trusted)
	updated, err = copilot.EnsureTrustedFolder(testPath)
	if err != nil {
		t.Fatalf("Second call failed: %v", err)
	}
	if updated {
		t.Error("Expected second call to return updated=false (already trusted)")
	}

	// Verify config
	copilotConfig := filepath.Join(xdgHome, ".copilot", "config.json")
	data, err := os.ReadFile(copilotConfig)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	if !strings.Contains(string(data), testPath) {
		t.Errorf("Expected %q in config. Got: %s", testPath, string(data))
	}
}

// TestCopilotTrustE2E_MultipleDirectories verifies multiple directories can be trusted.
func TestCopilotTrustE2E_MultipleDirectories(t *testing.T) {
	
	tmpDir := t.TempDir()

	xdgHome := filepath.Join(tmpDir, "xdg")
	t.Setenv("XDG_CONFIG_HOME", xdgHome)

	townRoot, rigPath := setupTestTown(t, tmpDir, "copilot")

	// Trust multiple directories
	dirs := []string{
		filepath.Join(rigPath, "witness"),
		filepath.Join(rigPath, "refinery", "rig"),
		filepath.Join(townRoot, "mayor"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create %s: %v", dir, err)
		}

		if _, err := copilot.EnsureTrustedFolder(dir); err != nil {
			t.Fatalf("Failed to trust %s: %v", dir, err)
		}
	}

	// Verify all directories are in config
	copilotConfig := filepath.Join(xdgHome, ".copilot", "config.json")
	data, err := os.ReadFile(copilotConfig)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	configStr := string(data)
	for _, dir := range dirs {
		if !strings.Contains(configStr, dir) {
			t.Errorf("Expected %q in config. Got: %s", dir, configStr)
		}
	}

	// Parse and verify count
	var configMap map[string]interface{}
	if err := json.Unmarshal(data, &configMap); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	folders, ok := configMap["trusted_folders"].([]interface{})
	if !ok {
		t.Fatal("Expected trusted_folders array")
	}

	if len(folders) != len(dirs) {
		t.Errorf("Expected %d trusted folders, got %d: %v", len(dirs), len(folders), folders)
	}
}

// setupTestTown creates a minimal town structure for testing.
func setupTestTown(t *testing.T, tmpDir, defaultAgent string) (townRoot, rigPath string) {
	t.Helper()

	townRoot = filepath.Join(tmpDir, "town")
	rigName := "testrig"
	rigPath = filepath.Join(townRoot, rigName)

	dirs := []string{
		filepath.Join(townRoot, "mayor"),
		filepath.Join(townRoot, "settings"),
		filepath.Join(rigPath, "settings"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create %s: %v", dir, err)
		}
	}

	// Create town.json
	townConfig := map[string]interface{}{
		"type":       "town",
		"version":    2,
		"name":       "test-town",
		"created_at": time.Now().Format(time.RFC3339),
	}
	writeJSON(t, filepath.Join(townRoot, "mayor", "town.json"), townConfig)

	// Create town settings
	townSettings := map[string]interface{}{
		"type":          "town-settings",
		"version":       1,
		"default_agent": defaultAgent,
	}
	writeJSON(t, filepath.Join(townRoot, "settings", "config.json"), townSettings)

	// Create rig settings (empty)
	rigSettings := map[string]interface{}{
		"type":    "rig-settings",
		"version": 1,
	}
	writeJSON(t, filepath.Join(rigPath, "settings", "config.json"), rigSettings)

	return townRoot, rigPath
}

// writeJSON writes a JSON config file.
func writeJSON(t *testing.T, path string, data interface{}) {
	t.Helper()

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal JSON for %s: %v", path, err)
	}

	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		t.Fatalf("Failed to write %s: %v", path, err)
	}
}
