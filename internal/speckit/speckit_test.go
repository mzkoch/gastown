package speckit

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

func TestGetGenerator(t *testing.T) {
	tests := []struct {
		agentType AgentType
		wantErr   bool
	}{
		{AgentClaude, false},
		{AgentCopilot, false},
		{AgentCursor, false},
		{AgentMCP, false},
		{AgentCLI, false},
		{"unknown", true},
	}

	for _, tt := range tests {
		t.Run(string(tt.agentType), func(t *testing.T) {
			gen, err := GetGenerator(tt.agentType)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetGenerator(%q) error = %v, wantErr %v", tt.agentType, err, tt.wantErr)
				return
			}
			if !tt.wantErr && gen == nil {
				t.Errorf("GetGenerator(%q) returned nil generator", tt.agentType)
			}
		})
	}
}

func TestAllAgentTypes(t *testing.T) {
	types := AllAgentTypes()
	if len(types) != 5 {
		t.Errorf("AllAgentTypes() returned %d types, want 5", len(types))
	}

	expected := map[AgentType]bool{
		AgentClaude:  false,
		AgentCopilot: false,
		AgentCursor:  false,
		AgentMCP:     false,
		AgentCLI:     false,
	}

	for _, at := range types {
		if _, ok := expected[at]; !ok {
			t.Errorf("AllAgentTypes() returned unexpected type: %q", at)
		}
		expected[at] = true
	}

	for at, found := range expected {
		if !found {
			t.Errorf("AllAgentTypes() missing type: %q", at)
		}
	}
}

func TestClaudeGenerator(t *testing.T) {
	tmpDir := t.TempDir()

	gen := &ClaudeGenerator{}
	if gen.Path() != ".claude/settings.json" {
		t.Errorf("ClaudeGenerator.Path() = %q, want %q", gen.Path(), ".claude/settings.json")
	}

	// Test autonomous role
	cfg := Config{
		WorkDir:  tmpDir,
		RoleType: Autonomous,
	}
	if err := gen.Generate(cfg); err != nil {
		t.Fatalf("ClaudeGenerator.Generate() error = %v", err)
	}

	settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")
	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("Failed to read settings: %v", err)
	}

	// Verify mail check is in autonomous settings
	if !strings.Contains(string(content), "mail check --inject") {
		t.Error("Autonomous settings should contain mail check")
	}

	// Test that existing file is not overwritten
	originalContent := content
	if err := gen.Generate(cfg); err != nil {
		t.Fatalf("Second Generate() error = %v", err)
	}
	newContent, _ := os.ReadFile(settingsPath)
	if string(newContent) != string(originalContent) {
		t.Error("Generate() should not overwrite existing file")
	}
}

func TestClaudeGeneratorInteractive(t *testing.T) {
	tmpDir := t.TempDir()

	gen := &ClaudeGenerator{}
	cfg := Config{
		WorkDir:  tmpDir,
		RoleType: Interactive,
	}
	if err := gen.Generate(cfg); err != nil {
		t.Fatalf("ClaudeGenerator.Generate() error = %v", err)
	}

	settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")
	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("Failed to read settings: %v", err)
	}

	// Verify SessionStart doesn't have mail check in interactive mode
	if strings.Contains(string(content), "SessionStart") {
		// Interactive mode should NOT have mail check in SessionStart
		lines := strings.Split(string(content), "\n")
		inSessionStart := false
		for _, line := range lines {
			if strings.Contains(line, "SessionStart") {
				inSessionStart = true
			}
			if inSessionStart && strings.Contains(line, "mail check --inject") {
				t.Error("Interactive SessionStart should not contain mail check")
				break
			}
			if inSessionStart && strings.Contains(line, "]") {
				break
			}
		}
	}
}

func TestCopilotGenerator(t *testing.T) {
	tmpDir := t.TempDir()

	gen := &CopilotGenerator{}
	if gen.Path() != ".github/copilot-instructions.md" {
		t.Errorf("CopilotGenerator.Path() = %q, want %q", gen.Path(), ".github/copilot-instructions.md")
	}

	cfg := Config{
		WorkDir:  tmpDir,
		RoleType: Autonomous,
	}
	if err := gen.Generate(cfg); err != nil {
		t.Fatalf("CopilotGenerator.Generate() error = %v", err)
	}

	configPath := filepath.Join(tmpDir, ".github", "copilot-instructions.md")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	if !strings.Contains(string(content), "Gas Town") {
		t.Error("Copilot instructions should mention Gas Town")
	}
}

func TestCursorGenerator(t *testing.T) {
	tmpDir := t.TempDir()

	gen := &CursorGenerator{}
	if gen.Path() != ".cursor/mcp.json" {
		t.Errorf("CursorGenerator.Path() = %q, want %q", gen.Path(), ".cursor/mcp.json")
	}

	cfg := Config{
		WorkDir:  tmpDir,
		RoleType: Autonomous,
	}
	if err := gen.Generate(cfg); err != nil {
		t.Fatalf("CursorGenerator.Generate() error = %v", err)
	}

	configPath := filepath.Join(tmpDir, ".cursor", "mcp.json")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	if !strings.Contains(string(content), "mcpServers") {
		t.Error("Cursor config should contain mcpServers")
	}
	if !strings.Contains(string(content), "gastown") {
		t.Error("Cursor config should contain gastown server")
	}
}

func TestMCPGenerator(t *testing.T) {
	tmpDir := t.TempDir()

	gen := &MCPGenerator{}
	if gen.Path() != ".mcp/servers.json" {
		t.Errorf("MCPGenerator.Path() = %q, want %q", gen.Path(), ".mcp/servers.json")
	}

	cfg := Config{
		WorkDir:  tmpDir,
		RoleType: Autonomous,
	}
	if err := gen.Generate(cfg); err != nil {
		t.Fatalf("MCPGenerator.Generate() error = %v", err)
	}

	configPath := filepath.Join(tmpDir, ".mcp", "servers.json")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	if !strings.Contains(string(content), "mcpServers") {
		t.Error("MCP config should contain mcpServers")
	}
}

func TestCLIGenerator(t *testing.T) {
	gen := &CLIGenerator{}
	if gen.Path() != "" {
		t.Errorf("CLIGenerator.Path() = %q, want empty string", gen.Path())
	}

	// Test autonomous commands
	cfg := Config{
		RoleType: Autonomous,
		GtPath:   "gt",
	}
	commands := gen.GetCommands(cfg)

	if len(commands) != 3 {
		t.Errorf("Autonomous GetCommands() returned %d commands, want 3", len(commands))
	}

	expectedCommands := []string{"gt prime", "gt mail check --inject", "gt nudge deacon session-started"}
	for i, expected := range expectedCommands {
		if i >= len(commands) || commands[i] != expected {
			t.Errorf("Command %d = %q, want %q", i, commands[i], expected)
		}
	}

	// Test interactive commands (no mail check)
	cfg.RoleType = Interactive
	commands = gen.GetCommands(cfg)

	if len(commands) != 2 {
		t.Errorf("Interactive GetCommands() returned %d commands, want 2", len(commands))
	}

	// Test command string
	cmdStr := gen.GetCommandString(cfg)
	if !strings.Contains(cmdStr, " && ") {
		t.Error("GetCommandString() should join commands with &&")
	}

	// Test with custom gt path
	cfg.GtPath = "/custom/path/gt"
	commands = gen.GetCommands(cfg)
	if !strings.HasPrefix(commands[0], "/custom/path/gt") {
		t.Errorf("Commands should use custom gt path, got: %s", commands[0])
	}
}

func TestCLIFallbackCommands(t *testing.T) {
	// Test autonomous role
	commands := CLIFallbackCommands("polecat", "")
	if len(commands) != 3 {
		t.Errorf("CLIFallbackCommands(polecat) returned %d commands, want 3", len(commands))
	}

	// Test interactive role
	commands = CLIFallbackCommands("crew", "")
	if len(commands) != 2 {
		t.Errorf("CLIFallbackCommands(crew) returned %d commands, want 2", len(commands))
	}

	// Test command string
	cmdStr := CLIFallbackCommandString("polecat", "")
	if !strings.Contains(cmdStr, "mail check") {
		t.Error("Autonomous command string should contain mail check")
	}
}

func TestEnsureConfig(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		WorkDir:  tmpDir,
		RoleType: Autonomous,
	}

	// Test Claude
	if err := EnsureConfig(AgentClaude, cfg); err != nil {
		t.Errorf("EnsureConfig(Claude) error = %v", err)
	}

	// Test unknown agent type
	if err := EnsureConfig("unknown", cfg); err == nil {
		t.Error("EnsureConfig(unknown) should return error")
	}
}

func TestEnsureAllConfigs(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		WorkDir:  tmpDir,
		RoleType: Autonomous,
	}

	if err := EnsureAllConfigs(cfg); err != nil {
		t.Errorf("EnsureAllConfigs() error = %v", err)
	}

	// Verify all config files were created
	expectedPaths := []string{
		".claude/settings.json",
		".github/copilot-instructions.md",
		".cursor/mcp.json",
		".mcp/servers.json",
	}

	for _, path := range expectedPaths {
		fullPath := filepath.Join(tmpDir, path)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("EnsureAllConfigs() did not create %s", path)
		}
	}
}

func TestBuildMCPServersConfig(t *testing.T) {
	config := BuildMCPServersConfig("")
	if config == nil {
		t.Fatal("BuildMCPServersConfig() returned nil")
	}

	if len(config.MCPServers) != 1 {
		t.Errorf("MCPServers has %d entries, want 1", len(config.MCPServers))
	}

	server, ok := config.MCPServers["gastown"]
	if !ok {
		t.Fatal("MCPServers missing 'gastown' entry")
	}

	if server.Command != "gt" {
		t.Errorf("Server command = %q, want %q", server.Command, "gt")
	}

	if len(server.Args) != 2 || server.Args[0] != "mcp" || server.Args[1] != "serve" {
		t.Errorf("Server args = %v, want [mcp serve]", server.Args)
	}

	// Test with custom path
	config = BuildMCPServersConfig("/custom/gt")
	if config.MCPServers["gastown"].Command != "/custom/gt" {
		t.Error("BuildMCPServersConfig should use custom gt path")
	}
}

func TestWriteMCPServersConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "subdir", "mcp.json")

	config := BuildMCPServersConfig("")
	if err := WriteMCPServersConfig(configPath, config); err != nil {
		t.Fatalf("WriteMCPServersConfig() error = %v", err)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	if !strings.Contains(string(content), "gastown") {
		t.Error("Written config should contain gastown")
	}
}
