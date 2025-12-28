package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Test GetMCPConfigFiles

func TestGetMCPConfigFiles_EmptyDirectory(t *testing.T) {
	// Create temporary ccagent MCP directory
	tempDir := t.TempDir()
	mcpDir := filepath.Join(tempDir, ".config", "ccagent", "mcp")

	if err := os.MkdirAll(mcpDir, 0755); err != nil {
		t.Fatalf("Failed to create MCP directory: %v", err)
	}

	// Temporarily override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Get MCP config files
	files, err := GetMCPConfigFiles()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("Expected 0 files, got: %d", len(files))
	}
}

func TestGetMCPConfigFiles_WithJSONFiles(t *testing.T) {
	// Create temporary ccagent MCP directory
	tempDir := t.TempDir()
	mcpDir := filepath.Join(tempDir, ".config", "ccagent", "mcp")

	if err := os.MkdirAll(mcpDir, 0755); err != nil {
		t.Fatalf("Failed to create MCP directory: %v", err)
	}

	// Create test JSON files
	testFiles := []string{"github.json", "postgres.json", "server.JSON"}
	for _, file := range testFiles {
		filePath := filepath.Join(mcpDir, file)
		if err := os.WriteFile(filePath, []byte("{}"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create non-JSON file (should be ignored)
	if err := os.WriteFile(filepath.Join(mcpDir, "readme.txt"), []byte("text"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Temporarily override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Get MCP config files
	files, err := GetMCPConfigFiles()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("Expected 3 files, got: %d", len(files))
	}
}

func TestGetMCPConfigFiles_NonexistentDirectory(t *testing.T) {
	// Use temporary directory that doesn't have MCP directory
	tempDir := t.TempDir()

	// Temporarily override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Get MCP config files (should return empty list, not error)
	files, err := GetMCPConfigFiles()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("Expected 0 files, got: %d", len(files))
	}
}

// Test CleanCcagentMCPDir

func TestCleanCcagentMCPDir_WithExistingConfigs(t *testing.T) {
	// Create temporary ccagent MCP directory with files
	tempDir := t.TempDir()
	mcpDir := filepath.Join(tempDir, ".config", "ccagent", "mcp")

	if err := os.MkdirAll(mcpDir, 0755); err != nil {
		t.Fatalf("Failed to create MCP directory: %v", err)
	}

	// Create test MCP config files
	testFiles := []string{"github.json", "postgres.json", "stale-server.json"}
	for _, file := range testFiles {
		filePath := filepath.Join(mcpDir, file)
		if err := os.WriteFile(filePath, []byte("{}"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Temporarily override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Clean the MCP directory
	if err := CleanCcagentMCPDir(); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify directory exists but is empty
	entries, err := os.ReadDir(mcpDir)
	if err != nil {
		t.Fatalf("Failed to read MCP directory: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("Expected empty directory, found %d files", len(entries))
	}
}

func TestCleanCcagentMCPDir_NonexistentDirectory(t *testing.T) {
	// Use temporary directory that doesn't have MCP directory
	tempDir := t.TempDir()

	// Temporarily override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Clean should succeed even if directory doesn't exist
	if err := CleanCcagentMCPDir(); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func TestCleanCcagentMCPDir_RecreatesDirectory(t *testing.T) {
	// Create temporary ccagent MCP directory
	tempDir := t.TempDir()
	mcpDir := filepath.Join(tempDir, ".config", "ccagent", "mcp")

	if err := os.MkdirAll(mcpDir, 0755); err != nil {
		t.Fatalf("Failed to create MCP directory: %v", err)
	}

	// Temporarily override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Clean the MCP directory
	if err := CleanCcagentMCPDir(); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify directory still exists (was recreated)
	if _, err := os.Stat(mcpDir); os.IsNotExist(err) {
		t.Errorf("Expected MCP directory to be recreated")
	}
}

// Test MergeMCPConfigs

func TestMergeMCPConfigs_EmptyDirectory(t *testing.T) {
	// Create temporary ccagent MCP directory
	tempDir := t.TempDir()
	mcpDir := filepath.Join(tempDir, ".config", "ccagent", "mcp")

	if err := os.MkdirAll(mcpDir, 0755); err != nil {
		t.Fatalf("Failed to create MCP directory: %v", err)
	}

	// Temporarily override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Merge MCP configs
	merged, err := MergeMCPConfigs()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(merged) != 0 {
		t.Errorf("Expected empty map, got: %d entries", len(merged))
	}
}

func TestMergeMCPConfigs_WithMultipleConfigs(t *testing.T) {
	// Create temporary ccagent MCP directory
	tempDir := t.TempDir()
	mcpDir := filepath.Join(tempDir, ".config", "ccagent", "mcp")

	if err := os.MkdirAll(mcpDir, 0755); err != nil {
		t.Fatalf("Failed to create MCP directory: %v", err)
	}

	// Create test MCP config files
	githubConfig := map[string]interface{}{
		"command": "uvx",
		"args":    []string{"mcp-server-github"},
		"env": map[string]interface{}{
			"GITHUB_TOKEN": "token123",
		},
	}

	postgresConfig := map[string]interface{}{
		"command": "docker",
		"args":    []string{"run", "postgres-mcp"},
		"env": map[string]interface{}{
			"DB_URL": "postgres://localhost",
		},
	}

	githubJSON, _ := json.Marshal(githubConfig)
	postgresJSON, _ := json.Marshal(postgresConfig)

	if err := os.WriteFile(filepath.Join(mcpDir, "github.json"), githubJSON, 0644); err != nil {
		t.Fatalf("Failed to create github config: %v", err)
	}

	if err := os.WriteFile(filepath.Join(mcpDir, "postgres.json"), postgresJSON, 0644); err != nil {
		t.Fatalf("Failed to create postgres config: %v", err)
	}

	// Temporarily override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Merge MCP configs
	merged, err := MergeMCPConfigs()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(merged) != 2 {
		t.Errorf("Expected 2 servers, got: %d", len(merged))
	}

	// Verify github server exists
	if _, ok := merged["github"]; !ok {
		t.Errorf("Expected 'github' server to exist")
	}

	// Verify postgres server exists
	if _, ok := merged["postgres"]; !ok {
		t.Errorf("Expected 'postgres' server to exist")
	}
}

// Test ClaudeCodeMCPProcessor

func TestClaudeCodeMCPProcessor_NoConfigs(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	workDir := filepath.Join(tempDir, "workspace")

	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work directory: %v", err)
	}

	// Temporarily override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Process MCP configs (should succeed with no configs)
	processor := NewClaudeCodeMCPProcessor(workDir)
	if err := processor.ProcessMCPConfigs(); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify .claude.json was not created when no configs
	claudeConfigPath := filepath.Join(tempDir, ".claude.json")
	if _, err := os.Stat(claudeConfigPath); !os.IsNotExist(err) {
		t.Errorf("Expected .claude.json not to exist when no configs")
	}
}

func TestClaudeCodeMCPProcessor_WithConfigs(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	workDir := filepath.Join(tempDir, "workspace")
	mcpDir := filepath.Join(tempDir, ".config", "ccagent", "mcp")

	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work directory: %v", err)
	}

	if err := os.MkdirAll(mcpDir, 0755); err != nil {
		t.Fatalf("Failed to create MCP directory: %v", err)
	}

	// Create test MCP config files
	githubConfig := map[string]interface{}{
		"command": "uvx",
		"args":    []string{"mcp-server-github"},
	}

	githubJSON, _ := json.Marshal(githubConfig)
	if err := os.WriteFile(filepath.Join(mcpDir, "github.json"), githubJSON, 0644); err != nil {
		t.Fatalf("Failed to create github config: %v", err)
	}

	// Temporarily override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Process MCP configs
	processor := NewClaudeCodeMCPProcessor(workDir)
	if err := processor.ProcessMCPConfigs(); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify .claude.json was created with mcpServers key
	claudeConfigPath := filepath.Join(tempDir, ".claude.json")
	content, err := os.ReadFile(claudeConfigPath)
	if err != nil {
		t.Fatalf("Expected .claude.json to exist, got error: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(content, &config); err != nil {
		t.Fatalf("Failed to parse .claude.json: %v", err)
	}

	// Verify mcpServers key exists
	mcpServers, ok := config["mcpServers"]
	if !ok {
		t.Fatalf("Expected mcpServers key to exist")
	}

	// Verify github server is in mcpServers
	serversMap, ok := mcpServers.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected mcpServers to be a map")
	}

	if _, ok := serversMap["github"]; !ok {
		t.Errorf("Expected 'github' server to exist in mcpServers")
	}
}

func TestClaudeCodeMCPProcessor_PreservesExistingConfig(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	workDir := filepath.Join(tempDir, "workspace")
	mcpDir := filepath.Join(tempDir, ".config", "ccagent", "mcp")

	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work directory: %v", err)
	}

	if err := os.MkdirAll(mcpDir, 0755); err != nil {
		t.Fatalf("Failed to create MCP directory: %v", err)
	}

	// Create existing .claude.json with other keys
	claudeConfigPath := filepath.Join(tempDir, ".claude.json")
	existingConfig := map[string]interface{}{
		"theme":    "dark",
		"fontSize": 14,
	}
	existingJSON, _ := json.Marshal(existingConfig)
	if err := os.WriteFile(claudeConfigPath, existingJSON, 0644); err != nil {
		t.Fatalf("Failed to create existing .claude.json: %v", err)
	}

	// Create test MCP config
	githubConfig := map[string]interface{}{
		"command": "uvx",
		"args":    []string{"mcp-server-github"},
	}
	githubJSON, _ := json.Marshal(githubConfig)
	if err := os.WriteFile(filepath.Join(mcpDir, "github.json"), githubJSON, 0644); err != nil {
		t.Fatalf("Failed to create github config: %v", err)
	}

	// Temporarily override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Process MCP configs
	processor := NewClaudeCodeMCPProcessor(workDir)
	if err := processor.ProcessMCPConfigs(); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify .claude.json preserves existing keys
	content, err := os.ReadFile(claudeConfigPath)
	if err != nil {
		t.Fatalf("Expected .claude.json to exist, got error: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(content, &config); err != nil {
		t.Fatalf("Failed to parse .claude.json: %v", err)
	}

	// Verify existing keys are preserved
	if theme, ok := config["theme"]; !ok || theme != "dark" {
		t.Errorf("Expected theme to be preserved")
	}

	if fontSize, ok := config["fontSize"]; !ok || fontSize.(float64) != 14 {
		t.Errorf("Expected fontSize to be preserved")
	}

	// Verify mcpServers was added
	if _, ok := config["mcpServers"]; !ok {
		t.Errorf("Expected mcpServers key to be added")
	}
}

// Test OpenCodeMCPProcessor

func TestOpenCodeMCPProcessor_NoConfigs(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	workDir := filepath.Join(tempDir, "workspace")

	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work directory: %v", err)
	}

	// Temporarily override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Process MCP configs (should succeed with no configs)
	processor := NewOpenCodeMCPProcessor(workDir)
	if err := processor.ProcessMCPConfigs(); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify opencode.json was not created when no configs
	opencodeConfigPath := filepath.Join(tempDir, ".config", "opencode", "opencode.json")
	if _, err := os.Stat(opencodeConfigPath); !os.IsNotExist(err) {
		t.Errorf("Expected opencode.json not to exist when no configs")
	}
}

func TestOpenCodeMCPProcessor_WithConfigs(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	workDir := filepath.Join(tempDir, "workspace")
	mcpDir := filepath.Join(tempDir, ".config", "ccagent", "mcp")

	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work directory: %v", err)
	}

	if err := os.MkdirAll(mcpDir, 0755); err != nil {
		t.Fatalf("Failed to create MCP directory: %v", err)
	}

	// Create test MCP config files
	githubConfig := map[string]interface{}{
		"command": "uvx",
		"args":    []string{"mcp-server-github"},
	}

	githubJSON, _ := json.Marshal(githubConfig)
	if err := os.WriteFile(filepath.Join(mcpDir, "github.json"), githubJSON, 0644); err != nil {
		t.Fatalf("Failed to create github config: %v", err)
	}

	// Temporarily override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Process MCP configs
	processor := NewOpenCodeMCPProcessor(workDir)
	if err := processor.ProcessMCPConfigs(); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify opencode.json was created with mcp key
	opencodeConfigPath := filepath.Join(tempDir, ".config", "opencode", "opencode.json")
	content, err := os.ReadFile(opencodeConfigPath)
	if err != nil {
		t.Fatalf("Expected opencode.json to exist, got error: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(content, &config); err != nil {
		t.Fatalf("Failed to parse opencode.json: %v", err)
	}

	// Verify mcp key exists
	mcpServers, ok := config["mcp"]
	if !ok {
		t.Fatalf("Expected mcp key to exist")
	}

	// Verify github server is in mcp
	serversMap, ok := mcpServers.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected mcp to be a map")
	}

	if _, ok := serversMap["github"]; !ok {
		t.Errorf("Expected 'github' server to exist in mcp")
	}
}

func TestOpenCodeMCPProcessor_PreservesExistingConfig(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	workDir := filepath.Join(tempDir, "workspace")
	mcpDir := filepath.Join(tempDir, ".config", "ccagent", "mcp")
	opencodeConfigDir := filepath.Join(tempDir, ".config", "opencode")

	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work directory: %v", err)
	}

	if err := os.MkdirAll(mcpDir, 0755); err != nil {
		t.Fatalf("Failed to create MCP directory: %v", err)
	}

	if err := os.MkdirAll(opencodeConfigDir, 0755); err != nil {
		t.Fatalf("Failed to create opencode config directory: %v", err)
	}

	// Create existing opencode.json with other keys (e.g., instructions)
	opencodeConfigPath := filepath.Join(opencodeConfigDir, "opencode.json")
	existingConfig := map[string]interface{}{
		"instructions": []string{"~/.config/ccagent/rules/*.md"},
		"theme":        "dark",
	}
	existingJSON, _ := json.Marshal(existingConfig)
	if err := os.WriteFile(opencodeConfigPath, existingJSON, 0644); err != nil {
		t.Fatalf("Failed to create existing opencode.json: %v", err)
	}

	// Create test MCP config
	githubConfig := map[string]interface{}{
		"command": "uvx",
		"args":    []string{"mcp-server-github"},
	}
	githubJSON, _ := json.Marshal(githubConfig)
	if err := os.WriteFile(filepath.Join(mcpDir, "github.json"), githubJSON, 0644); err != nil {
		t.Fatalf("Failed to create github config: %v", err)
	}

	// Temporarily override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Process MCP configs
	processor := NewOpenCodeMCPProcessor(workDir)
	if err := processor.ProcessMCPConfigs(); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify opencode.json preserves existing keys
	content, err := os.ReadFile(opencodeConfigPath)
	if err != nil {
		t.Fatalf("Expected opencode.json to exist, got error: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(content, &config); err != nil {
		t.Fatalf("Failed to parse opencode.json: %v", err)
	}

	// Verify existing keys are preserved
	if instructions, ok := config["instructions"]; !ok {
		t.Errorf("Expected instructions to be preserved")
	} else {
		instructionsArr, ok := instructions.([]interface{})
		if !ok || len(instructionsArr) != 1 {
			t.Errorf("Expected instructions array to be preserved")
		}
	}

	if theme, ok := config["theme"]; !ok || theme != "dark" {
		t.Errorf("Expected theme to be preserved")
	}

	// Verify mcp was added
	if _, ok := config["mcp"]; !ok {
		t.Errorf("Expected mcp key to be added")
	}
}

// Test NoOpMCPProcessor

func TestNoOpMCPProcessor(t *testing.T) {
	processor := NewNoOpMCPProcessor()
	if err := processor.ProcessMCPConfigs(); err != nil {
		t.Fatalf("Expected no error from NoOp processor, got: %v", err)
	}
}
