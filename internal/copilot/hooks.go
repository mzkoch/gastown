package copilot

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/steveyegge/gastown/internal/claude"
	"github.com/steveyegge/gastown/internal/util"
)

//go:embed config/*.json
var configFS embed.FS

type hooksConfig struct {
	Version int                         `json:"version"`
	Hooks   map[string][]map[string]any `json:"hooks,omitempty"`
}

// EnsureHooksForRole ensures a hooks.json file exists for Copilot CLI.
// It merges required hooks into any existing file without overwriting user entries.
func EnsureHooksForRole(workDir, role, hooksDir, hooksFile string) error {
	if hooksFile == "" {
		return errors.New("hooks file name is required")
	}
	if hooksDir == "" {
		hooksDir = "."
	}

	hooksPath := filepath.Join(workDir, hooksDir, hooksFile)
	existing, err := readHooksConfig(hooksPath)
	if err != nil {
		return err
	}
	required, err := requiredHooksForRole(role)
	if err != nil {
		return err
	}

	required.Version = 1

	updated := mergeHooks(existing, required)
	if !updated {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(hooksPath), 0755); err != nil {
		return fmt.Errorf("creating hooks directory: %w", err)
	}
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding hooks: %w", err)
	}
	if err := util.AtomicWriteFile(hooksPath, data, 0600); err != nil {
		return fmt.Errorf("writing hooks: %w", err)
	}
	return nil
}

func requiredHooksForRole(role string) (*hooksConfig, error) {
	roleType := claude.RoleTypeFor(role)
	templatePath := "config/hooks-interactive.json"
	if roleType == claude.Autonomous {
		templatePath = "config/hooks-autonomous.json"
	}
	content, err := configFS.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("reading hooks template: %w", err)
	}
	var cfg hooksConfig
	if err := json.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("parsing hooks template: %w", err)
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	if cfg.Hooks == nil {
		cfg.Hooks = make(map[string][]map[string]any)
	}
	return &cfg, nil
}

func readHooksConfig(path string) (*hooksConfig, error) {
	cfg := &hooksConfig{Version: 1, Hooks: make(map[string][]map[string]any)}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading hooks: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing hooks: %w", err)
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	if cfg.Hooks == nil {
		cfg.Hooks = make(map[string][]map[string]any)
	}
	return cfg, nil
}

func mergeHooks(existing, required *hooksConfig) bool {
	if existing == nil || required == nil {
		return false
	}

	updated := false
	if existing.Version == 0 && required.Version != 0 {
		existing.Version = required.Version
		updated = true
	}
	if required.Version > existing.Version {
		existing.Version = required.Version
		updated = true
	}
	if existing.Hooks == nil {
		existing.Hooks = make(map[string][]map[string]any)
	}

	for hookName, entries := range required.Hooks {
		existingEntries := existing.Hooks[hookName]
		for _, entry := range entries {
			if !containsHookEntry(existingEntries, entry) {
				existingEntries = append(existingEntries, entry)
				updated = true
			}
		}
		if len(existingEntries) > 0 {
			existing.Hooks[hookName] = existingEntries
		}
	}
	return updated
}

func containsHookEntry(existing []map[string]any, entry map[string]any) bool {
	target, err := json.Marshal(entry)
	if err != nil {
		return false
	}
	for _, candidate := range existing {
		data, err := json.Marshal(candidate)
		if err != nil {
			continue
		}
		if string(data) == string(target) {
			return true
		}
	}
	return false
}
