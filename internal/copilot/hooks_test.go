package copilot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureHooksForRole_MergesMissingHooks(t *testing.T) {
	tmpDir := t.TempDir()
	hooksPath := filepath.Join(tmpDir, "hooks.json")

	existing := map[string]any{
		"version": 1,
		"hooks": map[string]any{
			"sessionStart": []any{
				map[string]any{
					"type": "command",
					"bash": "gt prime --hook",
				},
			},
		},
	}
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		t.Fatalf("marshal existing hooks: %v", err)
	}
	if err := os.WriteFile(hooksPath, data, 0600); err != nil {
		t.Fatalf("write existing hooks: %v", err)
	}

	if err := EnsureHooksForRole(tmpDir, "polecat", ".", "hooks.json"); err != nil {
		t.Fatalf("EnsureHooksForRole() error = %v", err)
	}

	updated, err := os.ReadFile(hooksPath)
	if err != nil {
		t.Fatalf("read hooks: %v", err)
	}

	var merged hooksConfig
	if err := json.Unmarshal(updated, &merged); err != nil {
		t.Fatalf("parse hooks: %v", err)
	}

	if merged.Version != 1 {
		t.Errorf("expected version 1, got %d", merged.Version)
	}
	if len(merged.Hooks["sessionStart"]) == 0 {
		t.Fatal("expected sessionStart hooks to be present")
	}
	if len(merged.Hooks["preCompact"]) == 0 {
		t.Error("expected preCompact hook to be merged")
	}
}

func TestEnsureHooksForRole_NoChangesWhenComplete(t *testing.T) {
	tmpDir := t.TempDir()
	hooksPath := filepath.Join(tmpDir, "hooks.json")

	required, err := requiredHooksForRole("crew")
	if err != nil {
		t.Fatalf("requiredHooksForRole() error = %v", err)
	}
	required.Version = 1

	data, err := json.MarshalIndent(required, "", "  ")
	if err != nil {
		t.Fatalf("marshal hooks: %v", err)
	}
	if err := os.WriteFile(hooksPath, data, 0600); err != nil {
		t.Fatalf("write hooks: %v", err)
	}

	infoBefore, err := os.Stat(hooksPath)
	if err != nil {
		t.Fatalf("stat hooks before: %v", err)
	}

	if err := EnsureHooksForRole(tmpDir, "crew", ".", "hooks.json"); err != nil {
		t.Fatalf("EnsureHooksForRole() error = %v", err)
	}

	infoAfter, err := os.Stat(hooksPath)
	if err != nil {
		t.Fatalf("stat hooks after: %v", err)
	}

	if !infoAfter.ModTime().Equal(infoBefore.ModTime()) {
		t.Error("expected hooks.json to be unchanged when already complete")
	}
}
