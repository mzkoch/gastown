// Package runtime provides helpers for runtime-specific integration.
package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/steveyegge/gastown/internal/claude"
	"github.com/steveyegge/gastown/internal/copilot"
	"github.com/steveyegge/gastown/internal/config"
	"github.com/steveyegge/gastown/internal/opencode"
	"github.com/steveyegge/gastown/internal/tmux"
)

// EnsureSettingsForRole installs runtime hook settings when supported.
func EnsureSettingsForRole(workDir, role string, rc *config.RuntimeConfig) error {
	if rc == nil {
		rc = config.DefaultRuntimeConfig()
	}

	if rc.Hooks == nil {
		return nil
	}

	switch rc.Hooks.Provider {
	case "claude":
		return claude.EnsureSettingsForRoleAt(workDir, role, rc.Hooks.Dir, rc.Hooks.SettingsFile)
	case "opencode":
		return opencode.EnsurePluginAt(workDir, rc.Hooks.Dir, rc.Hooks.SettingsFile)
	case "copilot":
		return copilot.EnsureHooksForRole(workDir, role, rc.Hooks.Dir, rc.Hooks.SettingsFile)
	default:
		return nil
	}
}

// SessionIDFromEnv returns the runtime session ID, if present.
// It checks GT_SESSION_ID_ENV first, then falls back to CLAUDE_SESSION_ID.
func SessionIDFromEnv() string {
	if envName := os.Getenv("GT_SESSION_ID_ENV"); envName != "" {
		if sessionID := os.Getenv(envName); sessionID != "" {
			return sessionID
		}
	}
	return os.Getenv("CLAUDE_SESSION_ID")
}

// SleepForReadyDelay sleeps for the runtime's configured readiness delay.
func SleepForReadyDelay(rc *config.RuntimeConfig) {
	if rc == nil || rc.Tmux == nil {
		return
	}
	if rc.Tmux.ReadyDelayMs <= 0 {
		return
	}
	time.Sleep(time.Duration(rc.Tmux.ReadyDelayMs) * time.Millisecond)
}

// WaitForCopilotReady waits for Copilot to reach a ready prompt before nudges.
// Non-copilot runtimes return immediately.
func WaitForCopilotReady(t *tmux.Tmux, sessionID string, rc *config.RuntimeConfig, timeout time.Duration) {
	if t == nil || sessionID == "" || !isCopilotRuntime(rc) {
		return
	}
	readyConfig := copilotReadyConfig(rc)
	if err := t.WaitForRuntimeReady(sessionID, readyConfig, timeout); err != nil {
		delay := readyConfig.Tmux.ReadyDelayMs
		if delay < 10000 {
			delay = 10000 // Copilot needs extra time before nudges land reliably.
		}
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
}

// StartupFallbackCommands returns commands that approximate Claude hooks when hooks are unavailable.
func StartupFallbackCommands(role string, rc *config.RuntimeConfig) []string {
	if rc == nil {
		rc = config.DefaultRuntimeConfig()
	}
	if rc.Hooks != nil {
		switch strings.ToLower(rc.Hooks.Provider) {
		case "claude", "opencode":
			return nil
		case "copilot":
			if !hooksAvailable(rc) {
				break
			}
			return nil
		}
	}

	role = strings.ToLower(role)
	command := "gt prime"
	if isAutonomousRole(role) {
		command += " && gt mail check --inject"
	}
	command += " && gt nudge deacon session-started"

	return []string{command}
}

func isCopilotRuntime(rc *config.RuntimeConfig) bool {
	if rc == nil {
		return false
	}
	if strings.EqualFold(rc.Provider, "copilot") {
		return true
	}
	if rc.Command == "" {
		return false
	}
	return strings.EqualFold(filepath.Base(rc.Command), "copilot")
}

func copilotReadyConfig(rc *config.RuntimeConfig) *config.RuntimeConfig {
	if rc == nil {
		return &config.RuntimeConfig{
			Provider: "copilot",
			Tmux: &config.RuntimeTmuxConfig{
				ReadyPromptPrefix: "❯",
				ReadyDelayMs:      3000,
			},
		}
	}

	ready := &config.RuntimeConfig{
		Provider: rc.Provider,
		Command:  rc.Command,
		Tmux:     &config.RuntimeTmuxConfig{},
	}
	if rc.Tmux != nil {
		*ready.Tmux = *rc.Tmux
	}
	if ready.Tmux.ReadyPromptPrefix == "" {
		ready.Tmux.ReadyPromptPrefix = "❯"
	}
	if ready.Tmux.ReadyDelayMs == 0 {
		ready.Tmux.ReadyDelayMs = 3000
	}
	return ready
}
func hooksAvailable(rc *config.RuntimeConfig) bool {
	if rc == nil || rc.Hooks == nil {
		return false
	}
	if rc.Hooks.Dir == "" || rc.Hooks.SettingsFile == "" {
		return false
	}
	hooksPath := filepath.Join(rc.Hooks.Dir, rc.Hooks.SettingsFile)
	if filepath.IsAbs(hooksPath) {
		_, err := os.Stat(hooksPath)
		return err == nil
	}
	if isCopilotRuntime(rc) {
		if cwd, err := os.Getwd(); err == nil {
			if _, err := os.Stat(filepath.Join(cwd, hooksPath)); err == nil {
				return true
			}
		}
		return false
	}
	if home := os.Getenv("HOME"); home != "" {
		if _, err := os.Stat(filepath.Join(home, hooksPath)); err == nil {
			return true
		}
	}
	return false
}

// RunStartupFallback sends the startup fallback commands via tmux.
func RunStartupFallback(t *tmux.Tmux, sessionID, role string, rc *config.RuntimeConfig) error {
	WaitForCopilotReady(t, sessionID, rc, 30*time.Second)
	commands := StartupFallbackCommands(role, rc)
	for _, cmd := range commands {
		if err := t.NudgeSession(sessionID, cmd); err != nil {
			return err
		}
	}
	return nil
}

// isAutonomousRole returns true if the given role should automatically
// inject mail check on startup. Autonomous roles (polecat, witness,
// refinery, deacon) operate without human prompting and need mail injection
// to receive work assignments.
//
// Non-autonomous roles (mayor, crew) are human-guided and should not
// have automatic mail injection to avoid confusion.
func isAutonomousRole(role string) bool {
	switch role {
	case "polecat", "witness", "refinery", "deacon":
		return true
	default:
		return false
	}
}
