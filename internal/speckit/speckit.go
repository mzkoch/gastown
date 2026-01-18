// Package speckit provides agent configuration generators for various LLM coding assistants.
// It supports Claude, Copilot, Cursor, generic MCP servers, and CLI fallback configurations.
package speckit

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed config/*.json config/*.md
var configFS embed.FS

// AgentType identifies the type of agent to generate configuration for.
type AgentType string

const (
	// AgentClaude generates .claude/settings.json
	AgentClaude AgentType = "claude"
	// AgentCopilot generates .github/copilot-instructions.md
	AgentCopilot AgentType = "copilot"
	// AgentCursor generates .cursor/mcp.json
	AgentCursor AgentType = "cursor"
	// AgentMCP generates .mcp/servers.json
	AgentMCP AgentType = "mcp"
	// AgentCLI generates CLI fallback commands
	AgentCLI AgentType = "cli"
)

// RoleType indicates whether a role is autonomous or interactive.
type RoleType string

const (
	// Autonomous roles (polecat, witness, refinery) need mail injection in SessionStart.
	Autonomous RoleType = "autonomous"
	// Interactive roles (mayor, crew) wait for user input.
	Interactive RoleType = "interactive"
)

// RoleTypeFor returns the RoleType for a given role name.
func RoleTypeFor(role string) RoleType {
	switch strings.ToLower(role) {
	case "polecat", "witness", "refinery", "deacon":
		return Autonomous
	default:
		return Interactive
	}
}

// Config holds the configuration for generating agent settings.
type Config struct {
	// WorkDir is the target directory for configuration files.
	WorkDir string
	// RoleType determines which template variant to use.
	RoleType RoleType
	// GtPath is the path to the gt binary (for fallback commands).
	GtPath string
}

// Generator is the interface for agent config generators.
type Generator interface {
	// Generate creates the agent configuration files.
	Generate(cfg Config) error
	// Path returns the relative path where config will be written.
	Path() string
}

// GetGenerator returns a generator for the specified agent type.
func GetGenerator(agentType AgentType) (Generator, error) {
	switch agentType {
	case AgentClaude:
		return &ClaudeGenerator{}, nil
	case AgentCopilot:
		return &CopilotGenerator{}, nil
	case AgentCursor:
		return &CursorGenerator{}, nil
	case AgentMCP:
		return &MCPGenerator{}, nil
	case AgentCLI:
		return &CLIGenerator{}, nil
	default:
		return nil, fmt.Errorf("unknown agent type: %s", agentType)
	}
}

// AllAgentTypes returns all supported agent types.
func AllAgentTypes() []AgentType {
	return []AgentType{AgentClaude, AgentCopilot, AgentCursor, AgentMCP, AgentCLI}
}

// EnsureConfig ensures the configuration exists for the given agent type.
// If the file already exists, it's left unchanged.
func EnsureConfig(agentType AgentType, cfg Config) error {
	gen, err := GetGenerator(agentType)
	if err != nil {
		return err
	}
	return gen.Generate(cfg)
}

// EnsureAllConfigs ensures configurations exist for all agent types.
func EnsureAllConfigs(cfg Config) error {
	for _, agentType := range AllAgentTypes() {
		if err := EnsureConfig(agentType, cfg); err != nil {
			return fmt.Errorf("generating %s config: %w", agentType, err)
		}
	}
	return nil
}

// ClaudeGenerator generates .claude/settings.json configuration.
type ClaudeGenerator struct{}

func (g *ClaudeGenerator) Path() string {
	return ".claude/settings.json"
}

func (g *ClaudeGenerator) Generate(cfg Config) error {
	settingsPath := filepath.Join(cfg.WorkDir, g.Path())

	// If settings already exist, don't overwrite
	if _, err := os.Stat(settingsPath); err == nil {
		return nil
	}

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
		return fmt.Errorf("creating settings directory: %w", err)
	}

	// Select template based on role type
	var templateName string
	switch cfg.RoleType {
	case Autonomous:
		templateName = "config/claude-settings-autonomous.json"
	default:
		templateName = "config/claude-settings-interactive.json"
	}

	// Read template
	content, err := configFS.ReadFile(templateName)
	if err != nil {
		return fmt.Errorf("reading template %s: %w", templateName, err)
	}

	// Write settings file
	if err := os.WriteFile(settingsPath, content, 0600); err != nil {
		return fmt.Errorf("writing settings: %w", err)
	}

	return nil
}

// CopilotGenerator generates .github/copilot-instructions.md configuration.
type CopilotGenerator struct{}

func (g *CopilotGenerator) Path() string {
	return ".github/copilot-instructions.md"
}

func (g *CopilotGenerator) Generate(cfg Config) error {
	configPath := filepath.Join(cfg.WorkDir, g.Path())

	// If config already exists, don't overwrite
	if _, err := os.Stat(configPath); err == nil {
		return nil
	}

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Select template based on role type
	var templateName string
	switch cfg.RoleType {
	case Autonomous:
		templateName = "config/copilot-instructions-autonomous.md"
	default:
		templateName = "config/copilot-instructions-interactive.md"
	}

	// Read template
	content, err := configFS.ReadFile(templateName)
	if err != nil {
		return fmt.Errorf("reading template %s: %w", templateName, err)
	}

	// Write config file
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// CursorGenerator generates .cursor/mcp.json configuration.
type CursorGenerator struct{}

func (g *CursorGenerator) Path() string {
	return ".cursor/mcp.json"
}

func (g *CursorGenerator) Generate(cfg Config) error {
	configPath := filepath.Join(cfg.WorkDir, g.Path())

	// If config already exists, don't overwrite
	if _, err := os.Stat(configPath); err == nil {
		return nil
	}

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Read template
	content, err := configFS.ReadFile("config/cursor-mcp.json")
	if err != nil {
		return fmt.Errorf("reading template: %w", err)
	}

	// Write config file
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// MCPGenerator generates .mcp/servers.json configuration.
type MCPGenerator struct{}

func (g *MCPGenerator) Path() string {
	return ".mcp/servers.json"
}

func (g *MCPGenerator) Generate(cfg Config) error {
	configPath := filepath.Join(cfg.WorkDir, g.Path())

	// If config already exists, don't overwrite
	if _, err := os.Stat(configPath); err == nil {
		return nil
	}

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Read template
	content, err := configFS.ReadFile("config/mcp-servers.json")
	if err != nil {
		return fmt.Errorf("reading template: %w", err)
	}

	// Write config file
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// CLIGenerator generates CLI fallback commands (no file output).
type CLIGenerator struct{}

func (g *CLIGenerator) Path() string {
	return "" // CLI generator doesn't write files
}

func (g *CLIGenerator) Generate(cfg Config) error {
	// CLI generator doesn't write files - it generates commands via GetCommands
	return nil
}

// GetCommands returns CLI fallback commands for agents without hook support.
func (g *CLIGenerator) GetCommands(cfg Config) []string {
	gtPath := cfg.GtPath
	if gtPath == "" {
		gtPath = "gt"
	}

	var commands []string

	// Prime command runs first
	commands = append(commands, fmt.Sprintf("%s prime", gtPath))

	// Autonomous roles need mail injection
	if cfg.RoleType == Autonomous {
		commands = append(commands, fmt.Sprintf("%s mail check --inject", gtPath))
	}

	// Notify deacon of session start
	commands = append(commands, fmt.Sprintf("%s nudge deacon session-started", gtPath))

	return commands
}

// GetCommandString returns CLI fallback commands as a single string.
func (g *CLIGenerator) GetCommandString(cfg Config) string {
	commands := g.GetCommands(cfg)
	return strings.Join(commands, " && ")
}

// CLIFallbackCommands returns CLI fallback commands for the given role.
// This is a convenience function for use by other packages.
func CLIFallbackCommands(role string, gtPath string) []string {
	gen := &CLIGenerator{}
	return gen.GetCommands(Config{
		RoleType: RoleTypeFor(role),
		GtPath:   gtPath,
	})
}

// CLIFallbackCommandString returns CLI fallback commands as a single string.
func CLIFallbackCommandString(role string, gtPath string) string {
	gen := &CLIGenerator{}
	return gen.GetCommandString(Config{
		RoleType: RoleTypeFor(role),
		GtPath:   gtPath,
	})
}

// MCPServersConfig represents the structure of .mcp/servers.json
type MCPServersConfig struct {
	MCPServers map[string]MCPServerEntry `json:"mcpServers"`
}

// MCPServerEntry represents a single MCP server configuration.
type MCPServerEntry struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// BuildMCPServersConfig creates an MCPServersConfig with the gastown server.
func BuildMCPServersConfig(gtPath string) *MCPServersConfig {
	if gtPath == "" {
		gtPath = "gt"
	}
	return &MCPServersConfig{
		MCPServers: map[string]MCPServerEntry{
			"gastown": {
				Command: gtPath,
				Args:    []string{"mcp", "serve"},
			},
		},
	}
}

// WriteMCPServersConfig writes the MCP servers configuration to the given path.
func WriteMCPServersConfig(path string, config *MCPServersConfig) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}
