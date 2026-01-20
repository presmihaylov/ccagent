package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"ccagent/core/log"
)

// writeFileAsTargetUser writes content to a file, using sudo if necessary.
// When AGENT_EXEC_USER is set and the target path is in that user's home directory,
// the file is written via 'sudo -u <user> tee' to ensure proper ownership and permissions.
// This solves permission issues where ccagent (running as 'ccagent' user) needs to write
// files to the agent user's home directory (e.g., /home/agentrunner/.claude.json).
func writeFileAsTargetUser(filePath string, content []byte, perm os.FileMode) error {
	execUser := os.Getenv("AGENT_EXEC_USER")
	if execUser == "" {
		// Self-hosted mode: write directly
		return os.WriteFile(filePath, content, perm)
	}

	// Check if the target path is in the agent user's home directory
	agentHome := "/home/" + execUser
	if !strings.HasPrefix(filePath, agentHome) {
		// Not in agent's home, write directly
		return os.WriteFile(filePath, content, perm)
	}

	log.Info("🔌 Writing file as user '%s': %s", execUser, filePath)

	// Use sudo -u <user> tee to write the file with correct ownership
	// The tee command writes stdin to the file, and we redirect stdout to /dev/null
	cmd := exec.Command("sudo", "-u", execUser, "tee", filePath)
	cmd.Stdin = bytes.NewReader(content)
	cmd.Stdout = nil // Discard tee's stdout (it echoes the input)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to write file as user %s: %w (stderr: %s)", execUser, err, stderr.String())
	}

	return nil
}

// MCPProcessor defines the interface for processing agent-specific MCP configurations
type MCPProcessor interface {
	// ProcessMCPConfigs processes MCP configs from the ccagent MCP directory
	// and applies them to the agent-specific location.
	// targetHomeDir specifies the home directory to deploy configs to.
	// If empty, uses the current user's home directory.
	ProcessMCPConfigs(targetHomeDir string) error
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
// Each file is expected to have a top-level "mcpServers" key containing server configurations.
// Duplicate server names across files are handled by adding numeric suffixes (e.g., "server-1", "server-2").
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

		// Parse JSON - expect top-level mcpServers key
		var fileConfig struct {
			MCPServers map[string]interface{} `json:"mcpServers"`
		}
		if err := json.Unmarshal(content, &fileConfig); err != nil {
			return nil, fmt.Errorf("failed to parse MCP config file %s: %w", mcpFile, err)
		}

		// Merge servers from this file into the main map
		for serverName, serverConfig := range fileConfig.MCPServers {
			// Handle duplicate server names by adding numeric suffix
			finalName := serverName
			suffix := 1
			for {
				if _, exists := mergedServers[finalName]; !exists {
					break
				}
				suffix++
				finalName = fmt.Sprintf("%s-%d", serverName, suffix)
			}

			if finalName != serverName {
				log.Info("🔌 Duplicate server name '%s' detected, using '%s' instead", serverName, finalName)
			}

			mergedServers[finalName] = serverConfig
		}
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
// targetHomeDir specifies the home directory to deploy configs to.
// If empty, uses the current user's home directory.
func (p *ClaudeCodeMCPProcessor) ProcessMCPConfigs(targetHomeDir string) error {
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

	// Determine home directory for Claude Code config
	homeDir := targetHomeDir
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
	}

	log.Info("🔌 Deploying MCP configs to home directory: %s", homeDir)

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

	if err := writeFileAsTargetUser(claudeConfigPath, configJSON, 0644); err != nil {
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
// It reads all MCP configs, merges them, transforms them to OpenCode format,
// and updates ~/.config/opencode/opencode.json
// targetHomeDir specifies the home directory to deploy configs to.
// If empty, uses the current user's home directory.
func (p *OpenCodeMCPProcessor) ProcessMCPConfigs(targetHomeDir string) error {
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

	// Transform Claude Code MCP format to OpenCode format
	opencodeMcpServers := make(map[string]interface{})
	for serverName, serverConfig := range mcpServers {
		configMap, ok := serverConfig.(map[string]interface{})
		if !ok {
			log.Info("⚠️  Skipping invalid MCP server config for %s", serverName)
			continue
		}

		opencodeConfig := make(map[string]interface{})

		// Check if this is a remote server (has "url" field)
		if url, hasURL := configMap["url"]; hasURL {
			opencodeConfig["type"] = "remote"
			opencodeConfig["url"] = url
			if headers, ok := configMap["headers"]; ok {
				opencodeConfig["headers"] = headers
			}
		} else {
			// Local server - transform command + args to command array
			opencodeConfig["type"] = "local"

			var commandArray []string

			// Get the command
			if cmd, ok := configMap["command"].(string); ok {
				commandArray = append(commandArray, cmd)
			}

			// Append args to command array
			if args, ok := configMap["args"].([]interface{}); ok {
				for _, arg := range args {
					if argStr, ok := arg.(string); ok {
						commandArray = append(commandArray, argStr)
					}
				}
			}

			opencodeConfig["command"] = commandArray

			// Transform env -> environment
			if env, ok := configMap["env"]; ok {
				opencodeConfig["environment"] = env
			}
		}

		// Always enable the server
		opencodeConfig["enabled"] = true

		opencodeMcpServers[serverName] = opencodeConfig
	}

	// Determine home directory for OpenCode config
	homeDir := targetHomeDir
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
	}

	log.Info("🔌 Deploying OpenCode MCP configs to home directory: %s", homeDir)

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

	// Update mcp key with transformed configs
	existingConfig["mcp"] = opencodeMcpServers

	// Write updated config back
	configJSON, err := json.MarshalIndent(existingConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal opencode.json: %w", err)
	}

	log.Info("🔌 Updating opencode.json at: %s", opencodeConfigPath)

	if err := writeFileAsTargetUser(opencodeConfigPath, configJSON, 0644); err != nil {
		return fmt.Errorf("failed to write opencode.json: %w", err)
	}

	log.Info("✅ Successfully configured %d MCP server(s) for OpenCode", len(opencodeMcpServers))
	return nil
}

// NoOpMCPProcessor is a no-op implementation for agents that don't support MCP configs
type NoOpMCPProcessor struct{}

// NewNoOpMCPProcessor creates a new no-op MCP processor
func NewNoOpMCPProcessor() *NoOpMCPProcessor {
	return &NoOpMCPProcessor{}
}

// ProcessMCPConfigs implements MCPProcessor with no operation
func (p *NoOpMCPProcessor) ProcessMCPConfigs(targetHomeDir string) error {
	log.Info("🔌 MCP config processing not supported for this agent type")
	return nil
}
