package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ccagent/core/log"
)

// MCPProcessor defines the interface for processing agent-specific MCP configurations
type MCPProcessor interface {
	// ProcessMCPConfigs processes MCP configs from the ccagent MCP directory
	// and applies them to the agent-specific location
	ProcessMCPConfigs() error
}

// GetCcagentMCPDir returns the path to the ccagent MCP directory
func GetCcagentMCPDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(homeDir, ".config", "ccagent", "mcp"), nil
}

// GetMCPConfigFiles returns a list of JSON files in the ccagent MCP directory
func GetMCPConfigFiles() ([]string, error) {
	mcpDir, err := GetCcagentMCPDir()
	if err != nil {
		return nil, err
	}

	// Check if MCP directory exists
	if _, err := os.Stat(mcpDir); os.IsNotExist(err) {
		log.Info("🔌 MCP directory does not exist: %s", mcpDir)
		return []string{}, nil
	}

	// Read directory
	entries, err := os.ReadDir(mcpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read MCP directory: %w", err)
	}

	// Filter JSON files
	var mcpFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			mcpFiles = append(mcpFiles, filepath.Join(mcpDir, entry.Name()))
		}
	}

	return mcpFiles, nil
}

// CleanCcagentMCPDir removes all files from the ccagent MCP directory
// This should be called before downloading new MCP configs from the server to ensure
// stale configs that were deleted on the server are also removed locally.
func CleanCcagentMCPDir() error {
	mcpDir, err := GetCcagentMCPDir()
	if err != nil {
		return err
	}

	// Check if MCP directory exists
	if _, err := os.Stat(mcpDir); os.IsNotExist(err) {
		log.Info("🔌 MCP directory does not exist, nothing to clean: %s", mcpDir)
		return nil
	}

	log.Info("🔌 Cleaning ccagent MCP directory: %s", mcpDir)

	// Remove and recreate the directory to ensure a clean state
	if err := os.RemoveAll(mcpDir); err != nil {
		return fmt.Errorf("failed to remove MCP directory: %w", err)
	}

	// Recreate empty directory
	if err := os.MkdirAll(mcpDir, 0755); err != nil {
		return fmt.Errorf("failed to recreate MCP directory: %w", err)
	}

	log.Info("✅ Successfully cleaned ccagent MCP directory")
	return nil
}

// MergeMCPConfigs reads all MCP JSON files and merges them into a single mcpServers object
// Returns a map[string]interface{} representing the merged MCP server configurations
func MergeMCPConfigs() (map[string]interface{}, error) {
	mcpFiles, err := GetMCPConfigFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP config files: %w", err)
	}

	if len(mcpFiles) == 0 {
		return map[string]interface{}{}, nil
	}

	log.Info("🔌 Merging %d MCP config file(s)", len(mcpFiles))

	mergedServers := make(map[string]interface{})

	for _, mcpFile := range mcpFiles {
		// Read file
		content, err := os.ReadFile(mcpFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read MCP config file %s: %w", mcpFile, err)
		}

		// Parse JSON
		var serverConfig map[string]interface{}
		if err := json.Unmarshal(content, &serverConfig); err != nil {
			return nil, fmt.Errorf("failed to parse MCP config file %s: %w", mcpFile, err)
		}

		// Merge into the main map
		// Each file is expected to be a single MCP server config
		// The key is determined by the filename (without extension)
		fileName := filepath.Base(mcpFile)
		serverName := strings.TrimSuffix(fileName, filepath.Ext(fileName))

		// If the JSON has a top-level object, use it; otherwise wrap the content
		mergedServers[serverName] = serverConfig
	}

	return mergedServers, nil
}

// ClaudeCodeMCPProcessor handles MCP config processing for Claude Code
type ClaudeCodeMCPProcessor struct{}

// NewClaudeCodeMCPProcessor creates a new Claude Code MCP processor
func NewClaudeCodeMCPProcessor(workDir string) *ClaudeCodeMCPProcessor {
	return &ClaudeCodeMCPProcessor{}
}

// ProcessMCPConfigs implements MCPProcessor for Claude Code
// It reads all MCP configs, merges them, and updates ~/.claude.json
func (p *ClaudeCodeMCPProcessor) ProcessMCPConfigs() error {
	log.Info("🔌 Processing MCP configs for Claude Code agent")

	// Get merged MCP server configs
	mcpServers, err := MergeMCPConfigs()
	if err != nil {
		return fmt.Errorf("failed to merge MCP configs: %w", err)
	}

	if len(mcpServers) == 0 {
		log.Info("🔌 No MCP configs found in ccagent MCP directory")
		return nil
	}

	log.Info("🔌 Found %d MCP server(s) to configure", len(mcpServers))

	// Get home directory for Claude Code config
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	claudeConfigPath := filepath.Join(homeDir, ".claude.json")

	// Read existing config if it exists
	var existingConfig map[string]interface{}
	if content, err := os.ReadFile(claudeConfigPath); err == nil {
		if err := json.Unmarshal(content, &existingConfig); err != nil {
			log.Info("⚠️  Failed to parse existing .claude.json, creating new config: %v", err)
			existingConfig = make(map[string]interface{})
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read existing .claude.json: %w", err)
	} else {
		existingConfig = make(map[string]interface{})
	}

	// Update mcpServers key with merged configs
	existingConfig["mcpServers"] = mcpServers

	// Write updated config back
	configJSON, err := json.MarshalIndent(existingConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal .claude.json: %w", err)
	}

	log.Info("🔌 Updating .claude.json at: %s", claudeConfigPath)

	if err := os.WriteFile(claudeConfigPath, configJSON, 0644); err != nil {
		return fmt.Errorf("failed to write .claude.json: %w", err)
	}

	log.Info("✅ Successfully configured %d MCP server(s) for Claude Code", len(mcpServers))
	return nil
}

// OpenCodeMCPProcessor handles MCP config processing for OpenCode
type OpenCodeMCPProcessor struct {
	workDir string
}

// NewOpenCodeMCPProcessor creates a new OpenCode MCP processor
func NewOpenCodeMCPProcessor(workDir string) *OpenCodeMCPProcessor {
	return &OpenCodeMCPProcessor{
		workDir: workDir,
	}
}

// ProcessMCPConfigs implements MCPProcessor for OpenCode
// It reads all MCP configs, merges them, and updates ~/.config/opencode/opencode.json
func (p *OpenCodeMCPProcessor) ProcessMCPConfigs() error {
	log.Info("🔌 Processing MCP configs for OpenCode agent")

	// Get merged MCP server configs
	mcpServers, err := MergeMCPConfigs()
	if err != nil {
		return fmt.Errorf("failed to merge MCP configs: %w", err)
	}

	if len(mcpServers) == 0 {
		log.Info("🔌 No MCP configs found in ccagent MCP directory")
		return nil
	}

	log.Info("🔌 Found %d MCP server(s) to configure", len(mcpServers))

	// Get home directory for OpenCode config
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	opencodeConfigDir := filepath.Join(homeDir, ".config", "opencode")
	opencodeConfigPath := filepath.Join(opencodeConfigDir, "opencode.json")

	// Ensure OpenCode config directory exists
	if err := os.MkdirAll(opencodeConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create OpenCode config directory: %w", err)
	}

	// Read existing config if it exists
	var existingConfig map[string]interface{}
	if content, err := os.ReadFile(opencodeConfigPath); err == nil {
		if err := json.Unmarshal(content, &existingConfig); err != nil {
			log.Info("⚠️  Failed to parse existing opencode.json, creating new config: %v", err)
			existingConfig = make(map[string]interface{})
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read existing opencode.json: %w", err)
	} else {
		existingConfig = make(map[string]interface{})
	}

	// Update mcp key with merged configs
	existingConfig["mcp"] = mcpServers

	// Write updated config back
	configJSON, err := json.MarshalIndent(existingConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal opencode.json: %w", err)
	}

	log.Info("🔌 Updating opencode.json at: %s", opencodeConfigPath)

	if err := os.WriteFile(opencodeConfigPath, configJSON, 0644); err != nil {
		return fmt.Errorf("failed to write opencode.json: %w", err)
	}

	log.Info("✅ Successfully configured %d MCP server(s) for OpenCode", len(mcpServers))
	return nil
}

// NoOpMCPProcessor is a no-op implementation for agents that don't support MCP configs
type NoOpMCPProcessor struct{}

// NewNoOpMCPProcessor creates a new no-op MCP processor
func NewNoOpMCPProcessor() *NoOpMCPProcessor {
	return &NoOpMCPProcessor{}
}

// ProcessMCPConfigs implements MCPProcessor with no operation
func (p *NoOpMCPProcessor) ProcessMCPConfigs() error {
	log.Info("🔌 MCP config processing not supported for this agent type")
	return nil
}
