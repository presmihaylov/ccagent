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

	// Create test MCP config files with top-level mcpServers key
	file1Config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"github": map[string]interface{}{
				"command": "uvx",
				"args":    []string{"mcp-server-github"},
				"env": map[string]interface{}{
					"GITHUB_TOKEN": "token123",
				},
			},
		},
	}

	file2Config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"postgres": map[string]interface{}{
				"command": "docker",
				"args":    []string{"run", "postgres-mcp"},
				"env": map[string]interface{}{
					"DB_URL": "postgres://localhost",
				},
			},
		},
	}

	file1JSON, _ := json.Marshal(file1Config)
	file2JSON, _ := json.Marshal(file2Config)

	if err := os.WriteFile(filepath.Join(mcpDir, "github.json"), file1JSON, 0644); err != nil {
		t.Fatalf("Failed to create github config: %v", err)
	}

	if err := os.WriteFile(filepath.Join(mcpDir, "postgres.json"), file2JSON, 0644); err != nil {
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

func TestMergeMCPConfigs_WithDuplicateServerNames(t *testing.T) {
	// Create temporary ccagent MCP directory
	tempDir := t.TempDir()
	mcpDir := filepath.Join(tempDir, ".config", "ccagent", "mcp")

	if err := os.MkdirAll(mcpDir, 0755); err != nil {
		t.Fatalf("Failed to create MCP directory: %v", err)
	}

	// Create test MCP config files with duplicate server names
	file1Config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"shared-server": map[string]interface{}{
				"command": "server-v1",
				"args":    []string{},
			},
			"unique-server-1": map[string]interface{}{
				"command": "unique1",
				"args":    []string{},
			},
		},
	}

	file2Config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"shared-server": map[string]interface{}{
				"command": "server-v2",
				"args":    []string{},
			},
			"unique-server-2": map[string]interface{}{
				"command": "unique2",
				"args":    []string{},
			},
		},
	}

	file3Config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"shared-server": map[string]interface{}{
				"command": "server-v3",
				"args":    []string{},
			},
		},
	}

	file1JSON, _ := json.Marshal(file1Config)
	file2JSON, _ := json.Marshal(file2Config)
	file3JSON, _ := json.Marshal(file3Config)

	if err := os.WriteFile(filepath.Join(mcpDir, "file1.json"), file1JSON, 0644); err != nil {
		t.Fatalf("Failed to create file1 config: %v", err)
	}

	if err := os.WriteFile(filepath.Join(mcpDir, "file2.json"), file2JSON, 0644); err != nil {
		t.Fatalf("Failed to create file2 config: %v", err)
	}

	if err := os.WriteFile(filepath.Join(mcpDir, "file3.json"), file3JSON, 0644); err != nil {
		t.Fatalf("Failed to create file3 config: %v", err)
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

	// Expect 5 servers total: shared-server, shared-server-2, shared-server-3, unique-server-1, unique-server-2
	if len(merged) != 5 {
		t.Errorf("Expected 5 servers, got: %d", len(merged))
	}

	// Verify first instance has original name
	if _, ok := merged["shared-server"]; !ok {
		t.Errorf("Expected 'shared-server' to exist")
	}

	// Verify second instance has suffix -2
	if _, ok := merged["shared-server-2"]; !ok {
		t.Errorf("Expected 'shared-server-2' to exist")
	}

	// Verify third instance has suffix -3
	if _, ok := merged["shared-server-3"]; !ok {
		t.Errorf("Expected 'shared-server-3' to exist")
	}

	// Verify unique servers exist
	if _, ok := merged["unique-server-1"]; !ok {
		t.Errorf("Expected 'unique-server-1' to exist")
	}

	if _, ok := merged["unique-server-2"]; !ok {
		t.Errorf("Expected 'unique-server-2' to exist")
	}
}

func TestMergeMCPConfigs_WithInvalidJSON(t *testing.T) {
	// Create temporary ccagent MCP directory
	tempDir := t.TempDir()
	mcpDir := filepath.Join(tempDir, ".config", "ccagent", "mcp")

	if err := os.MkdirAll(mcpDir, 0755); err != nil {
		t.Fatalf("Failed to create MCP directory: %v", err)
	}

	// Create invalid JSON file
	invalidJSON := []byte("{invalid json content")
	if err := os.WriteFile(filepath.Join(mcpDir, "invalid.json"), invalidJSON, 0644); err != nil {
		t.Fatalf("Failed to create invalid JSON file: %v", err)
	}

	// Temporarily override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Merge MCP configs - should return error
	_, err := MergeMCPConfigs()
	if err == nil {
		t.Errorf("Expected error for invalid JSON, got nil")
	}
}

func TestMergeMCPConfigs_WithMissingMCPServersKey(t *testing.T) {
	// Create temporary ccagent MCP directory
	tempDir := t.TempDir()
	mcpDir := filepath.Join(tempDir, ".config", "ccagent", "mcp")

	if err := os.MkdirAll(mcpDir, 0755); err != nil {
		t.Fatalf("Failed to create MCP directory: %v", err)
	}

	// Create config file without mcpServers key
	configWithoutKey := map[string]interface{}{
		"someOtherKey": "value",
	}
	configJSON, _ := json.Marshal(configWithoutKey)
	if err := os.WriteFile(filepath.Join(mcpDir, "no-key.json"), configJSON, 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Temporarily override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Merge MCP configs - should succeed but return empty map
	merged, err := MergeMCPConfigs()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(merged) != 0 {
		t.Errorf("Expected empty map when mcpServers key is missing, got: %d entries", len(merged))
	}
}

func TestMergeMCPConfigs_WithEmptyMCPServersObject(t *testing.T) {
	// Create temporary ccagent MCP directory
	tempDir := t.TempDir()
	mcpDir := filepath.Join(tempDir, ".config", "ccagent", "mcp")

	if err := os.MkdirAll(mcpDir, 0755); err != nil {
		t.Fatalf("Failed to create MCP directory: %v", err)
	}

	// Create config file with empty mcpServers object
	emptyConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{},
	}
	configJSON, _ := json.Marshal(emptyConfig)
	if err := os.WriteFile(filepath.Join(mcpDir, "empty.json"), configJSON, 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Temporarily override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Merge MCP configs - should succeed but return empty map
	merged, err := MergeMCPConfigs()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(merged) != 0 {
		t.Errorf("Expected empty map when mcpServers is empty, got: %d entries", len(merged))
	}
}

func TestMergeMCPConfigs_WithMultipleServersInSingleFile(t *testing.T) {
	// Create temporary ccagent MCP directory
	tempDir := t.TempDir()
	mcpDir := filepath.Join(tempDir, ".config", "ccagent", "mcp")

	if err := os.MkdirAll(mcpDir, 0755); err != nil {
		t.Fatalf("Failed to create MCP directory: %v", err)
	}

	// Create config file with multiple servers
	multiServerConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"github": map[string]interface{}{
				"command": "uvx",
				"args":    []string{"mcp-server-github"},
			},
			"postgres": map[string]interface{}{
				"command": "docker",
				"args":    []string{"run", "postgres-mcp"},
			},
			"filesystem": map[string]interface{}{
				"command": "node",
				"args":    []string{"mcp-server-filesystem"},
			},
		},
	}
	configJSON, _ := json.Marshal(multiServerConfig)
	if err := os.WriteFile(filepath.Join(mcpDir, "multiple.json"), configJSON, 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
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

	// Should have all 3 servers
	if len(merged) != 3 {
		t.Errorf("Expected 3 servers, got: %d", len(merged))
	}

	// Verify all servers exist
	expectedServers := []string{"github", "postgres", "filesystem"}
	for _, serverName := range expectedServers {
		if _, ok := merged[serverName]; !ok {
			t.Errorf("Expected '%s' server to exist", serverName)
		}
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

	// Create test MCP config file with top-level mcpServers key
	fileConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"github": map[string]interface{}{
				"command": "uvx",
				"args":    []string{"mcp-server-github"},
			},
		},
	}

	fileJSON, _ := json.Marshal(fileConfig)
	if err := os.WriteFile(filepath.Join(mcpDir, "github.json"), fileJSON, 0644); err != nil {
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

func TestClaudeCodeMCPProcessor_WithMultipleMCPConfigs(t *testing.T) {
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

	// Create multiple MCP config files
	file1Config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"github": map[string]interface{}{
				"command": "uvx",
				"args":    []string{"mcp-server-github"},
			},
		},
	}

	file2Config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"postgres": map[string]interface{}{
				"command": "docker",
				"args":    []string{"run", "postgres-mcp"},
			},
		},
	}

	file1JSON, _ := json.Marshal(file1Config)
	file2JSON, _ := json.Marshal(file2Config)

	if err := os.WriteFile(filepath.Join(mcpDir, "github.json"), file1JSON, 0644); err != nil {
		t.Fatalf("Failed to create github config: %v", err)
	}

	if err := os.WriteFile(filepath.Join(mcpDir, "postgres.json"), file2JSON, 0644); err != nil {
		t.Fatalf("Failed to create postgres config: %v", err)
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

	// Verify .claude.json was created with both servers
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

	serversMap, ok := mcpServers.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected mcpServers to be a map")
	}

	// Verify both servers are present
	if len(serversMap) != 2 {
		t.Errorf("Expected 2 servers, got: %d", len(serversMap))
	}

	if _, ok := serversMap["github"]; !ok {
		t.Errorf("Expected 'github' server to exist in mcpServers")
	}

	if _, ok := serversMap["postgres"]; !ok {
		t.Errorf("Expected 'postgres' server to exist in mcpServers")
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

	// Create test MCP config with top-level mcpServers key
	fileConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"github": map[string]interface{}{
				"command": "uvx",
				"args":    []string{"mcp-server-github"},
			},
		},
	}
	fileJSON, _ := json.Marshal(fileConfig)
	if err := os.WriteFile(filepath.Join(mcpDir, "github.json"), fileJSON, 0644); err != nil {
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

	// Create test MCP config file with top-level mcpServers key
	fileConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"github": map[string]interface{}{
				"command": "uvx",
				"args":    []string{"mcp-server-github"},
			},
		},
	}

	fileJSON, _ := json.Marshal(fileConfig)
	if err := os.WriteFile(filepath.Join(mcpDir, "github.json"), fileJSON, 0644); err != nil {
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

func TestOpenCodeMCPProcessor_WithMultipleMCPConfigs(t *testing.T) {
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

	// Create multiple MCP config files
	file1Config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"github": map[string]interface{}{
				"command": "uvx",
				"args":    []string{"mcp-server-github"},
			},
		},
	}

	file2Config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"postgres": map[string]interface{}{
				"command": "docker",
				"args":    []string{"run", "postgres-mcp"},
			},
		},
	}

	file1JSON, _ := json.Marshal(file1Config)
	file2JSON, _ := json.Marshal(file2Config)

	if err := os.WriteFile(filepath.Join(mcpDir, "github.json"), file1JSON, 0644); err != nil {
		t.Fatalf("Failed to create github config: %v", err)
	}

	if err := os.WriteFile(filepath.Join(mcpDir, "postgres.json"), file2JSON, 0644); err != nil {
		t.Fatalf("Failed to create postgres config: %v", err)
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

	// Verify opencode.json was created with both servers
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

	serversMap, ok := mcpServers.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected mcp to be a map")
	}

	// Verify both servers are present
	if len(serversMap) != 2 {
		t.Errorf("Expected 2 servers, got: %d", len(serversMap))
	}

	if _, ok := serversMap["github"]; !ok {
		t.Errorf("Expected 'github' server to exist in mcp")
	}

	if _, ok := serversMap["postgres"]; !ok {
		t.Errorf("Expected 'postgres' server to exist in mcp")
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

	// Create test MCP config with top-level mcpServers key
	fileConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"github": map[string]interface{}{
				"command": "uvx",
				"args":    []string{"mcp-server-github"},
			},
		},
	}
	fileJSON, _ := json.Marshal(fileConfig)
	if err := os.WriteFile(filepath.Join(mcpDir, "github.json"), fileJSON, 0644); err != nil {
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
