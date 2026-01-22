package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureCopilotTrustedFolder_NoWorkDir(t *testing.T) {
	if err := EnsureCopilotTrustedFolder(CopilotTrustConfig{}); err != nil {
		t.Fatalf("EnsureCopilotTrustedFolder returned error: %v", err)
	}
}

func TestEnsureCopilotTrustedFolder_UpdatesWhenCopilot(t *testing.T) {
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

	xdgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgHome)

	if err := EnsureCopilotTrustedFolder(CopilotTrustConfig{
		TownRoot:  townRoot,
		RigPath:   rigPath,
		WorkDir:   workDir,
		ConfigDir: filepath.Join(xdgHome, ".copilot"),
	}); err != nil {
		t.Fatalf("EnsureCopilotTrustedFolder: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(xdgHome, ".copilot", "config.json"))
	if err != nil {
		t.Fatalf("ReadFile config.json: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected config.json to be written")
	}
}

func TestEnsureCopilotTrustedFolder_SkipsNonCopilot(t *testing.T) {
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

	xdgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgHome)

	if err := EnsureCopilotTrustedFolder(CopilotTrustConfig{
		TownRoot:  townRoot,
		RigPath:   rigPath,
		WorkDir:   workDir,
		ConfigDir: filepath.Join(xdgHome, ".copilot"),
	}); err != nil {
		t.Fatalf("EnsureCopilotTrustedFolder: %v", err)
	}

	if _, err := os.Stat(filepath.Join(xdgHome, ".copilot", "config.json")); !os.IsNotExist(err) {
		t.Fatalf("expected no config.json, got err=%v", err)
	}
}

func TestEnsureCopilotTrustedFolder_PolecatTrustsParent(t *testing.T) {
	townRoot := t.TempDir()
	rigPath := filepath.Join(townRoot, "testrig")
	polecatWorkDir := filepath.Join(rigPath, "polecats", "capable", "testrig")

	townSettings := NewTownSettings()
	townSettings.DefaultAgent = "copilot"
	if err := SaveTownSettings(TownSettingsPath(townRoot), townSettings); err != nil {
		t.Fatalf("SaveTownSettings: %v", err)
	}
	if err := SaveRigSettings(RigSettingsPath(rigPath), NewRigSettings()); err != nil {
		t.Fatalf("SaveRigSettings: %v", err)
	}

	xdgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgHome)

	// Call with a polecat working directory
	if err := EnsureCopilotTrustedFolder(CopilotTrustConfig{
		Role:      "polecat",
		TownRoot:  townRoot,
		RigPath:   rigPath,
		WorkDir:   polecatWorkDir,
		ConfigDir: filepath.Join(xdgHome, ".copilot"),
	}); err != nil {
		t.Fatalf("EnsureCopilotTrustedFolder: %v", err)
	}

	// Verify that polecats/ parent dir was trusted, not the specific worktree
	data, err := os.ReadFile(filepath.Join(xdgHome, ".copilot", "config.json"))
	if err != nil {
		t.Fatalf("ReadFile config.json: %v", err)
	}
	
	configStr := string(data)
	polecatsDir := filepath.Join(rigPath, "polecats")
	if !strings.Contains(configStr, polecatsDir) {
		t.Fatalf("expected polecats directory %q to be trusted, got config:\n%s", polecatsDir, configStr)
	}
	if strings.Contains(configStr, polecatWorkDir) {
		t.Fatalf("expected polecats parent to be trusted, not specific worktree %q, got config:\n%s", polecatWorkDir, configStr)
	}
}
