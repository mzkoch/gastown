package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCopilotIntegration_ConfigFileCreation tests that config file is created when it doesn't exist.
func TestCopilotIntegration_ConfigFileCreation(t *testing.T) {
	xdgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgHome)

	townRoot := t.TempDir()
	rigPath := filepath.Join(townRoot, "testrig")
	workDir := filepath.Join(rigPath, "witness", "rig")

	townSettings := NewTownSettings()
	townSettings.DefaultAgent = "copilot"
	if err := SaveTownSettings(TownSettingsPath(townRoot), townSettings); err != nil {
		t.Fatalf("SaveTownSettings: %v", err)
	}
	if err := SaveRigSettings(RigSettingsPath(rigPath), NewRigSettings()); err != nil {
		t.Fatalf("SaveRigSettings: %v", err)
	}

	configPath := filepath.Join(xdgHome, ".copilot", "config.json")
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("expected config file to not exist initially")
	}

	if err := EnsureCopilotTrustedFolder(CopilotTrustConfig{
		TownRoot: townRoot,
		RigPath:  rigPath,
		WorkDir:  workDir,
	}); err != nil {
		t.Fatalf("EnsureCopilotTrustedFolder: %v", err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("expected config file to be created")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile config.json: %v", err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("Unmarshal config: %v", err)
	}

	trusted, ok := cfg["trusted_folders"].([]interface{})
	if !ok || len(trusted) == 0 {
		t.Fatalf("expected trusted_folders to be populated, got: %v", cfg)
	}
}

// TestCopilotIntegration_TrustedFolderAddition tests that folders are added to trusted list.
func TestCopilotIntegration_TrustedFolderAddition(t *testing.T) {
	xdgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgHome)

	townRoot := t.TempDir()
	rigPath := filepath.Join(townRoot, "testrig")

	townSettings := NewTownSettings()
	townSettings.DefaultAgent = "copilot"
	if err := SaveTownSettings(TownSettingsPath(townRoot), townSettings); err != nil {
		t.Fatalf("SaveTownSettings: %v", err)
	}
	if err := SaveRigSettings(RigSettingsPath(rigPath), NewRigSettings()); err != nil {
		t.Fatalf("SaveRigSettings: %v", err)
	}

	workDir1 := filepath.Join(rigPath, "witness", "rig")
	workDir2 := filepath.Join(rigPath, "refinery", "rig")

	if err := EnsureCopilotTrustedFolder(CopilotTrustConfig{
		TownRoot: townRoot,
		RigPath:  rigPath,
		WorkDir:  workDir1,
	}); err != nil {
		t.Fatalf("First EnsureCopilotTrustedFolder: %v", err)
	}

	if err := EnsureCopilotTrustedFolder(CopilotTrustConfig{
		TownRoot: townRoot,
		RigPath:  rigPath,
		WorkDir:  workDir2,
	}); err != nil {
		t.Fatalf("Second EnsureCopilotTrustedFolder: %v", err)
	}

	configPath := filepath.Join(xdgHome, ".copilot", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile config.json: %v", err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("Unmarshal config: %v", err)
	}

	trusted, ok := cfg["trusted_folders"].([]interface{})
	if !ok || len(trusted) < 2 {
		t.Fatalf("expected at least 2 trusted folders, got: %v", trusted)
	}
}

// TestCopilotIntegration_Idempotency tests that calling EnsureCopilotTrustedFolder twice
// with the same path doesn't duplicate entries.
func TestCopilotIntegration_Idempotency(t *testing.T) {
	tests := []struct {
		name string
		role string
	}{
		{"witness", "witness"},
		{"boot", "boot"},
		{"mayor", "mayor"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xdgHome := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", xdgHome)

			townRoot := t.TempDir()
			rigPath := filepath.Join(townRoot, "testrig")
			workDir := filepath.Join(rigPath, tt.role, "rig")

			townSettings := NewTownSettings()
			townSettings.DefaultAgent = "copilot"
			townSettings.RoleAgents = map[string]string{tt.role: "copilot"}
			if err := SaveTownSettings(TownSettingsPath(townRoot), townSettings); err != nil {
				t.Fatalf("SaveTownSettings: %v", err)
			}
			if err := SaveRigSettings(RigSettingsPath(rigPath), NewRigSettings()); err != nil {
				t.Fatalf("SaveRigSettings: %v", err)
			}

			// Call twice with the same path
			if err := EnsureCopilotTrustedFolder(CopilotTrustConfig{
				Role:     tt.role,
				TownRoot: townRoot,
				RigPath:  rigPath,
				WorkDir:  workDir,
			}); err != nil {
				t.Fatalf("First call: %v", err)
			}

			if err := EnsureCopilotTrustedFolder(CopilotTrustConfig{
				Role:     tt.role,
				TownRoot: townRoot,
				RigPath:  rigPath,
				WorkDir:  workDir,
			}); err != nil {
				t.Fatalf("Second call: %v", err)
			}

			configPath := filepath.Join(xdgHome, ".copilot", "config.json")
			data, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("ReadFile config.json: %v", err)
			}

			var cfg map[string]interface{}
			if err := json.Unmarshal(data, &cfg); err != nil {
				t.Fatalf("Unmarshal config: %v", err)
			}

			trusted, ok := cfg["trusted_folders"].([]interface{})
			if !ok {
				t.Fatalf("expected trusted_folders to exist")
			}

			// Convert to absolute paths for comparison
			absWorkDir, _ := filepath.Abs(workDir)
			count := 0
			for _, t := range trusted {
				if s, ok := t.(string); ok {
					absPath, _ := filepath.Abs(s)
					if strings.EqualFold(absPath, absWorkDir) {
						count++
					}
				}
			}

			if count != 1 {
				t.Fatalf("expected workDir to appear exactly once, appeared %d times. Trusted folders: %v", count, trusted)
			}
		})
	}
}

// TestCopilotIntegration_MultipleAgentStartupPatterns tests different agent startup patterns.
func TestCopilotIntegration_MultipleAgentStartupPatterns(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		workDir  string
		agentCfg string
	}{
		{
			name:    "boot role",
			role:    "boot",
			workDir: "boot/rig",
		},
		{
			name:    "mayor role",
			role:    "mayor",
			workDir: "mayor/rig",
		},
		{
			name:    "witness role",
			role:    "witness",
			workDir: "witness/rig",
		},
		{
			name:    "refinery role",
			role:    "refinery",
			workDir: "refinery/rig",
		},
		{
			name:    "deacon role",
			role:    "deacon",
			workDir: "deacon/rig",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xdgHome := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", xdgHome)

			townRoot := t.TempDir()
			rigPath := filepath.Join(townRoot, "testrig")
			workDir := filepath.Join(rigPath, tt.workDir)

			townSettings := NewTownSettings()
			townSettings.DefaultAgent = "copilot"
			townSettings.RoleAgents = map[string]string{tt.role: "copilot"}
			if err := SaveTownSettings(TownSettingsPath(townRoot), townSettings); err != nil {
				t.Fatalf("SaveTownSettings: %v", err)
			}
			if err := SaveRigSettings(RigSettingsPath(rigPath), NewRigSettings()); err != nil {
				t.Fatalf("SaveRigSettings: %v", err)
			}

			if err := EnsureCopilotTrustedFolder(CopilotTrustConfig{
				Role:     tt.role,
				TownRoot: townRoot,
				RigPath:  rigPath,
				WorkDir:  workDir,
			}); err != nil {
				t.Fatalf("EnsureCopilotTrustedFolder: %v", err)
			}

			configPath := filepath.Join(xdgHome, ".copilot", "config.json")
			data, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("ReadFile config.json: %v", err)
			}

			var cfg map[string]interface{}
			if err := json.Unmarshal(data, &cfg); err != nil {
				t.Fatalf("Unmarshal config: %v", err)
			}

			trusted, ok := cfg["trusted_folders"].([]interface{})
			if !ok || len(trusted) == 0 {
				t.Fatalf("expected trusted_folders to be populated, got: %v", cfg)
			}
		})
	}
}

// TestCopilotIntegration_PolecatSpecialCase tests that polecats trust the parent
// directory instead of individual worktrees.
func TestCopilotIntegration_PolecatSpecialCase(t *testing.T) {
	xdgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgHome)

	townRoot := t.TempDir()
	rigPath := filepath.Join(townRoot, "testrig")
	polecatsDir := filepath.Join(rigPath, "polecats")
	polecatWorktree1 := filepath.Join(polecatsDir, "capable", "testrig")
	polecatWorktree2 := filepath.Join(polecatsDir, "swift", "testrig")

	townSettings := NewTownSettings()
	townSettings.DefaultAgent = "copilot"
	if err := SaveTownSettings(TownSettingsPath(townRoot), townSettings); err != nil {
		t.Fatalf("SaveTownSettings: %v", err)
	}
	if err := SaveRigSettings(RigSettingsPath(rigPath), NewRigSettings()); err != nil {
		t.Fatalf("SaveRigSettings: %v", err)
	}

	// Call with first polecat worktree
	if err := EnsureCopilotTrustedFolder(CopilotTrustConfig{
		Role:     "polecat",
		TownRoot: townRoot,
		RigPath:  rigPath,
		WorkDir:  polecatWorktree1,
	}); err != nil {
		t.Fatalf("First polecat call: %v", err)
	}

	// Call with second polecat worktree
	if err := EnsureCopilotTrustedFolder(CopilotTrustConfig{
		Role:     "polecat",
		TownRoot: townRoot,
		RigPath:  rigPath,
		WorkDir:  polecatWorktree2,
	}); err != nil {
		t.Fatalf("Second polecat call: %v", err)
	}

	configPath := filepath.Join(xdgHome, ".copilot", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile config.json: %v", err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("Unmarshal config: %v", err)
	}

	trusted, ok := cfg["trusted_folders"].([]interface{})
	if !ok {
		t.Fatalf("expected trusted_folders to exist")
	}

	// Verify polecats/ directory is trusted
	absPolecatsDir, _ := filepath.Abs(polecatsDir)
	var found bool
	for _, entry := range trusted {
		if s, ok := entry.(string); ok {
			absPath, _ := filepath.Abs(s)
			if strings.EqualFold(absPath, absPolecatsDir) {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatalf("expected polecats directory %q to be trusted, got: %v", absPolecatsDir, trusted)
	}

	// Verify individual worktrees are NOT trusted
	absWorktree1, _ := filepath.Abs(polecatWorktree1)
	absWorktree2, _ := filepath.Abs(polecatWorktree2)
	for _, entry := range trusted {
		if s, ok := entry.(string); ok {
			absPath, _ := filepath.Abs(s)
			if strings.EqualFold(absPath, absWorktree1) || strings.EqualFold(absPath, absWorktree2) {
				t.Fatalf("individual polecat worktrees should not be trusted, got: %s", s)
			}
		}
	}

	// Should only have one entry (the polecats/ directory)
	if len(trusted) != 1 {
		t.Fatalf("expected exactly 1 trusted folder (polecats/), got %d: %v", len(trusted), trusted)
	}
}

// TestCopilotIntegration_PolecatParentDirAddedOnce verifies that even when calling
// EnsureCopilotTrustedFolder multiple times with different polecat worktrees,
// the parent directory is only added once.
func TestCopilotIntegration_PolecatParentDirAddedOnce(t *testing.T) {
	xdgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgHome)

	townRoot := t.TempDir()
	rigPath := filepath.Join(townRoot, "testrig")
	polecatsDir := filepath.Join(rigPath, "polecats")

	townSettings := NewTownSettings()
	townSettings.DefaultAgent = "copilot"
	if err := SaveTownSettings(TownSettingsPath(townRoot), townSettings); err != nil {
		t.Fatalf("SaveTownSettings: %v", err)
	}
	if err := SaveRigSettings(RigSettingsPath(rigPath), NewRigSettings()); err != nil {
		t.Fatalf("SaveRigSettings: %v", err)
	}

	// Call multiple times with different polecat worktrees
	for i := 0; i < 3; i++ {
		worktree := filepath.Join(polecatsDir, "polecat"+string(rune(48+i)), "testrig")
		if err := EnsureCopilotTrustedFolder(CopilotTrustConfig{
			Role:     "polecat",
			TownRoot: townRoot,
			RigPath:  rigPath,
			WorkDir:  worktree,
		}); err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
	}

	configPath := filepath.Join(xdgHome, ".copilot", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile config.json: %v", err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("Unmarshal config: %v", err)
	}

	trusted, ok := cfg["trusted_folders"].([]interface{})
	if !ok {
		t.Fatalf("expected trusted_folders to exist")
	}

	// Should have exactly one entry (the polecats/ parent directory)
	if len(trusted) != 1 {
		t.Fatalf("expected exactly 1 trusted folder, got %d: %v", len(trusted), trusted)
	}
}

// TestCopilotIntegration_ConfigWithExistingData preserves existing config data
// when adding a new trusted folder.
func TestCopilotIntegration_ConfigWithExistingData(t *testing.T) {
	xdgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgHome)

	townRoot := t.TempDir()
	rigPath := filepath.Join(townRoot, "testrig")
	workDir := filepath.Join(rigPath, "witness", "rig")

	townSettings := NewTownSettings()
	townSettings.DefaultAgent = "copilot"
	if err := SaveTownSettings(TownSettingsPath(townRoot), townSettings); err != nil {
		t.Fatalf("SaveTownSettings: %v", err)
	}
	if err := SaveRigSettings(RigSettingsPath(rigPath), NewRigSettings()); err != nil {
		t.Fatalf("SaveRigSettings: %v", err)
	}

	// Create initial config with some data
	configDir := filepath.Join(xdgHome, ".copilot")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	initialCfg := map[string]interface{}{
		"some_setting": "some_value",
		"nested": map[string]interface{}{
			"key": "value",
		},
	}
	initialData, _ := json.MarshalIndent(initialCfg, "", "  ")
	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, initialData, 0600); err != nil {
		t.Fatalf("WriteFile initial config: %v", err)
	}

	// Call EnsureCopilotTrustedFolder
	if err := EnsureCopilotTrustedFolder(CopilotTrustConfig{
		TownRoot: townRoot,
		RigPath:  rigPath,
		WorkDir:  workDir,
	}); err != nil {
		t.Fatalf("EnsureCopilotTrustedFolder: %v", err)
	}

	// Verify existing data is preserved
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile config.json: %v", err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("Unmarshal config: %v", err)
	}

	if v, ok := cfg["some_setting"]; !ok || v != "some_value" {
		t.Fatalf("expected some_setting to be preserved, got: %v", v)
	}

	nested, ok := cfg["nested"].(map[string]interface{})
	if !ok || nested["key"] != "value" {
		t.Fatalf("expected nested data to be preserved, got: %v", cfg["nested"])
	}

	if _, ok := cfg["trusted_folders"]; !ok {
		t.Fatalf("expected trusted_folders to be added")
	}
}

// TestCopilotIntegration_SkipsNonCopilotAgents verifies that non-copilot agents
// don't update the copilot config.
func TestCopilotIntegration_SkipsNonCopilotAgents(t *testing.T) {
	tests := []struct {
		name         string
		defaultAgent string
	}{
		{"claude", "claude"},
		{"gemini", "gemini"},
		{"cursor", "cursor"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xdgHome := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", xdgHome)

			townRoot := t.TempDir()
			rigPath := filepath.Join(townRoot, "testrig")
			workDir := filepath.Join(rigPath, "witness", "rig")

			townSettings := NewTownSettings()
			townSettings.DefaultAgent = tt.defaultAgent
			if err := SaveTownSettings(TownSettingsPath(townRoot), townSettings); err != nil {
				t.Fatalf("SaveTownSettings: %v", err)
			}
			if err := SaveRigSettings(RigSettingsPath(rigPath), NewRigSettings()); err != nil {
				t.Fatalf("SaveRigSettings: %v", err)
			}

			if err := EnsureCopilotTrustedFolder(CopilotTrustConfig{
				TownRoot: townRoot,
				RigPath:  rigPath,
				WorkDir:  workDir,
			}); err != nil {
				t.Fatalf("EnsureCopilotTrustedFolder: %v", err)
			}

			// Config file should not be created for non-copilot agents
			configPath := filepath.Join(xdgHome, ".copilot", "config.json")
			if _, err := os.Stat(configPath); !os.IsNotExist(err) {
				t.Fatalf("expected no config.json for %s agent, but file exists", tt.name)
			}
		})
	}
}

// TestCopilotIntegration_EmptyWorkDirSkipped verifies that an empty WorkDir
// doesn't create a config file.
func TestCopilotIntegration_EmptyWorkDirSkipped(t *testing.T) {
	xdgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgHome)

	if err := EnsureCopilotTrustedFolder(CopilotTrustConfig{
		TownRoot: "/some/town",
		RigPath:  "/some/rig",
		WorkDir:  "", // Empty work dir
	}); err != nil {
		t.Fatalf("EnsureCopilotTrustedFolder: %v", err)
	}

	// Config file should not be created
	configPath := filepath.Join(xdgHome, ".copilot", "config.json")
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("expected no config.json with empty WorkDir")
	}
}

// TestCopilotIntegration_RoleAgentOverride tests role-specific agent configuration.
func TestCopilotIntegration_RoleAgentOverride(t *testing.T) {
	xdgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgHome)

	townRoot := t.TempDir()
	rigPath := filepath.Join(townRoot, "testrig")
	workDir := filepath.Join(rigPath, "witness", "rig")

	townSettings := NewTownSettings()
	townSettings.DefaultAgent = "claude"
	townSettings.RoleAgents = map[string]string{
		"witness": "copilot",
	}
	if err := SaveTownSettings(TownSettingsPath(townRoot), townSettings); err != nil {
		t.Fatalf("SaveTownSettings: %v", err)
	}
	if err := SaveRigSettings(RigSettingsPath(rigPath), NewRigSettings()); err != nil {
		t.Fatalf("SaveRigSettings: %v", err)
	}

	if err := EnsureCopilotTrustedFolder(CopilotTrustConfig{
		Role:     "witness",
		TownRoot: townRoot,
		RigPath:  rigPath,
		WorkDir:  workDir,
	}); err != nil {
		t.Fatalf("EnsureCopilotTrustedFolder: %v", err)
	}

	// Config file should be created because witness role has copilot override
	configPath := filepath.Join(xdgHome, ".copilot", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected config.json to be created, got err: %v", err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("Unmarshal config: %v", err)
	}

	trusted, ok := cfg["trusted_folders"].([]interface{})
	if !ok || len(trusted) == 0 {
		t.Fatalf("expected trusted_folders to be populated")
	}
}

// TestCopilotIntegration_AgentOverridePathBased tests agent override via AgentOverride field.
func TestCopilotIntegration_AgentOverridePathBased(t *testing.T) {
	xdgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgHome)

	townRoot := t.TempDir()
	rigPath := filepath.Join(townRoot, "testrig")
	workDir := filepath.Join(rigPath, "witness", "rig")

	townSettings := NewTownSettings()
	townSettings.DefaultAgent = "claude"
	if err := SaveTownSettings(TownSettingsPath(townRoot), townSettings); err != nil {
		t.Fatalf("SaveTownSettings: %v", err)
	}
	if err := SaveRigSettings(RigSettingsPath(rigPath), NewRigSettings()); err != nil {
		t.Fatalf("SaveRigSettings: %v", err)
	}

	// Use AgentOverride to explicitly select copilot
	if err := EnsureCopilotTrustedFolder(CopilotTrustConfig{
		TownRoot:      townRoot,
		RigPath:       rigPath,
		WorkDir:       workDir,
		AgentOverride: "copilot",
	}); err != nil {
		t.Fatalf("EnsureCopilotTrustedFolder: %v", err)
	}

	// Config file should be created
	configPath := filepath.Join(xdgHome, ".copilot", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected config.json to be created, got err: %v", err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("Unmarshal config: %v", err)
	}

	trusted, ok := cfg["trusted_folders"].([]interface{})
	if !ok || len(trusted) == 0 {
		t.Fatalf("expected trusted_folders to be populated")
	}
}

// TestCopilotIntegration_PathNormalization tests that paths are properly normalized
// and duplicates with different formats are recognized.
func TestCopilotIntegration_PathNormalization(t *testing.T) {
	xdgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgHome)

	townRoot := t.TempDir()
	rigPath := filepath.Join(townRoot, "testrig")
	workDir := filepath.Join(rigPath, "witness", "rig")

	townSettings := NewTownSettings()
	townSettings.DefaultAgent = "copilot"
	if err := SaveTownSettings(TownSettingsPath(townRoot), townSettings); err != nil {
		t.Fatalf("SaveTownSettings: %v", err)
	}
	if err := SaveRigSettings(RigSettingsPath(rigPath), NewRigSettings()); err != nil {
		t.Fatalf("SaveRigSettings: %v", err)
	}

	// First call with standard path
	if err := EnsureCopilotTrustedFolder(CopilotTrustConfig{
		TownRoot: townRoot,
		RigPath:  rigPath,
		WorkDir:  workDir,
	}); err != nil {
		t.Fatalf("First call: %v", err)
	}

	// Second call with path that has trailing slash or extra dots
	workDirVariant := workDir + string(filepath.Separator)
	if err := EnsureCopilotTrustedFolder(CopilotTrustConfig{
		TownRoot: townRoot,
		RigPath:  rigPath,
		WorkDir:  workDirVariant,
	}); err != nil {
		t.Fatalf("Second call with variant: %v", err)
	}

	configPath := filepath.Join(xdgHome, ".copilot", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile config.json: %v", err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("Unmarshal config: %v", err)
	}

	trusted, ok := cfg["trusted_folders"].([]interface{})
	if !ok {
		t.Fatalf("expected trusted_folders to exist")
	}

	// Should have exactly one entry (path normalization should prevent duplicates)
	if len(trusted) != 1 {
		t.Fatalf("expected exactly 1 trusted folder due to path normalization, got %d: %v", len(trusted), trusted)
	}
}
