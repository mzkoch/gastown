package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRoleTypeFor(t *testing.T) {
	tests := []struct {
		role     string
		expected RoleType
	}{
		{"polecat", Autonomous},
		{"witness", Autonomous},
		{"refinery", Autonomous},
		{"deacon", Autonomous},
		{"mayor", Interactive},
		{"crew", Interactive},
		{"", Interactive},
		{"unknown", Interactive},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			got := RoleTypeFor(tt.role)
			if got != tt.expected {
				t.Errorf("RoleTypeFor(%q) = %v, want %v", tt.role, got, tt.expected)
			}
		})
	}
}

func TestEnsureSettings_CreatesAutonomousSettings(t *testing.T) {
	tmpDir := t.TempDir()

	if err := EnsureSettings(tmpDir, Autonomous); err != nil {
		t.Fatalf("EnsureSettings() error = %v", err)
	}

	settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")
	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("failed to read settings: %v", err)
	}

	if !strings.Contains(string(content), "mail check --inject") {
		t.Error("autonomous settings should contain mail check")
	}
}

func TestEnsureSettings_InteractiveNoSessionStartMail(t *testing.T) {
	tmpDir := t.TempDir()

	if err := EnsureSettings(tmpDir, Interactive); err != nil {
		t.Fatalf("EnsureSettings() error = %v", err)
	}

	settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")
	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("failed to read settings: %v", err)
	}

	if strings.Contains(string(content), "SessionStart") {
		lines := strings.Split(string(content), "\n")
		inSessionStart := false
		for _, line := range lines {
			if strings.Contains(line, "SessionStart") {
				inSessionStart = true
			}
			if inSessionStart && strings.Contains(line, "mail check --inject") {
				t.Error("interactive SessionStart should not contain mail check")
				break
			}
			if inSessionStart && strings.Contains(line, "]") {
				break
			}
		}
	}
}

func TestEnsureSettings_DoesNotOverwriteExisting(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")
	original := []byte("original settings")

	if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
		t.Fatalf("failed to create settings dir: %v", err)
	}
	if err := os.WriteFile(settingsPath, original, 0600); err != nil {
		t.Fatalf("failed to write settings file: %v", err)
	}

	if err := EnsureSettings(tmpDir, Autonomous); err != nil {
		t.Fatalf("EnsureSettings() error = %v", err)
	}

	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("failed to read settings: %v", err)
	}
	if string(content) != string(original) {
		t.Error("EnsureSettings() should not overwrite existing file")
	}
}

func TestEnsureSettingsAt_EmptySettingsDir(t *testing.T) {
	tmpDir := t.TempDir()

	if err := EnsureSettingsAt(tmpDir, Autonomous, "", "settings.json"); err != nil {
		t.Fatalf("EnsureSettingsAt() error = %v", err)
	}

	settingsPath := filepath.Join(tmpDir, "settings.json")
	if _, err := os.Stat(settingsPath); err != nil {
		t.Fatalf("expected settings at workdir: %v", err)
	}
}

func TestEnsureSettingsAt_EmptySettingsFile(t *testing.T) {
	tmpDir := t.TempDir()

	if err := EnsureSettingsAt(tmpDir, Autonomous, ".claude", ""); err == nil {
		t.Error("EnsureSettingsAt() with empty settings file should error")
	}
}

func TestEnsureSettingsForRoleAt_CreatesCustomPath(t *testing.T) {
	tmpDir := t.TempDir()

	if err := EnsureSettingsForRoleAt(tmpDir, "polecat", "custom", "settings.json"); err != nil {
		t.Fatalf("EnsureSettingsForRoleAt() error = %v", err)
	}

	settingsPath := filepath.Join(tmpDir, "custom", "settings.json")
	if _, err := os.Stat(settingsPath); err != nil {
		t.Fatalf("expected settings at custom path: %v", err)
	}
}
