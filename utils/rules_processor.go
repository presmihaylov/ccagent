package utils

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ccagent/core/log"
)

// RuleFrontMatter represents the parsed front matter from a rule file
type RuleFrontMatter struct {
	Title       string
	Description string
}

// RulesProcessor defines the interface for processing agent-specific rules
type RulesProcessor interface {
	// ProcessRules processes rules from the ccagent rules directory
	// and copies them to the agent-specific location
	ProcessRules() error
}

// ParseFrontMatter extracts title and description from markdown front matter
// Expected format:
// ---
// title: Code Style Guidelines
// description: Use this to learn what style guidelines to follow
// ---
func ParseFrontMatter(filePath string) (*RuleFrontMatter, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	frontMatter := &RuleFrontMatter{}
	scanner := bufio.NewScanner(file)

	// Check if file starts with front matter delimiter
	if !scanner.Scan() || scanner.Text() != "---" {
		// No front matter found - return empty front matter
		return frontMatter, nil
	}

	// Parse front matter
	for scanner.Scan() {
		line := scanner.Text()

		// End of front matter
		if line == "---" {
			break
		}

		// Parse key-value pairs
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch strings.ToLower(key) {
		case "title":
			frontMatter.Title = value
		case "description":
			frontMatter.Description = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return frontMatter, nil
}

// GetCcagentRulesDir returns the path to the ccagent rules directory
func GetCcagentRulesDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(homeDir, ".config", "ccagent", "rules"), nil
}

// GetRuleFiles returns a list of markdown files in the ccagent rules directory
func GetRuleFiles() ([]string, error) {
	rulesDir, err := GetCcagentRulesDir()
	if err != nil {
		return nil, err
	}

	// Check if rules directory exists
	if _, err := os.Stat(rulesDir); os.IsNotExist(err) {
		log.Info("📋 Rules directory does not exist: %s", rulesDir)
		return []string{}, nil
	}

	// Read directory
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules directory: %w", err)
	}

	// Filter markdown files
	var ruleFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			ruleFiles = append(ruleFiles, filepath.Join(rulesDir, entry.Name()))
		}
	}

	return ruleFiles, nil
}

// CleanCcagentRulesDir removes all files from the ccagent rules directory
// This should be called before downloading new rules from the server to ensure
// stale rules that were deleted on the server are also removed locally.
func CleanCcagentRulesDir() error {
	rulesDir, err := GetCcagentRulesDir()
	if err != nil {
		return err
	}

	// Check if rules directory exists
	if _, err := os.Stat(rulesDir); os.IsNotExist(err) {
		log.Info("📋 Rules directory does not exist, nothing to clean: %s", rulesDir)
		return nil
	}

	log.Info("📋 Cleaning ccagent rules directory: %s", rulesDir)

	// Remove and recreate the directory to ensure a clean state
	if err := os.RemoveAll(rulesDir); err != nil {
		return fmt.Errorf("failed to remove rules directory: %w", err)
	}

	// Recreate empty directory
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		return fmt.Errorf("failed to recreate rules directory: %w", err)
	}

	log.Info("✅ Successfully cleaned ccagent rules directory")
	return nil
}

// ClaudeCodeRulesProcessor handles rules processing for Claude Code
type ClaudeCodeRulesProcessor struct{}

// NewClaudeCodeRulesProcessor creates a new Claude Code rules processor
func NewClaudeCodeRulesProcessor(workDir string) *ClaudeCodeRulesProcessor {
	return &ClaudeCodeRulesProcessor{}
}

// ProcessRules implements RulesProcessor for Claude Code
func (p *ClaudeCodeRulesProcessor) ProcessRules() error {
	log.Info("📋 Processing rules for Claude Code agent")

	// Get rule files from ccagent directory
	ruleFiles, err := GetRuleFiles()
	if err != nil {
		return fmt.Errorf("failed to get rule files: %w", err)
	}

	if len(ruleFiles) == 0 {
		log.Info("📋 No rules found in ccagent rules directory")
		return nil
	}

	log.Info("📋 Found %d rule file(s) to process", len(ruleFiles))

	// Get home directory for Claude Code rules
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Create .claude/rules directory in home directory
	claudeRulesDir := filepath.Join(homeDir, ".claude", "rules")

	// Clean up existing rules directory to avoid stale rules
	log.Info("📋 Cleaning Claude Code rules directory: %s", claudeRulesDir)
	if err := os.RemoveAll(claudeRulesDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing rules directory: %w", err)
	}

	// Create fresh rules directory
	if err := os.MkdirAll(claudeRulesDir, 0755); err != nil {
		return fmt.Errorf("failed to create Claude rules directory: %w", err)
	}

	// Copy each rule file
	for _, ruleFile := range ruleFiles {
		fileName := filepath.Base(ruleFile)
		destPath := filepath.Join(claudeRulesDir, fileName)

		log.Info("📋 Copying rule: %s -> %s", fileName, destPath)

		// Read source file
		content, err := os.ReadFile(ruleFile)
		if err != nil {
			return fmt.Errorf("failed to read rule file %s: %w", ruleFile, err)
		}

		// Write to destination
		if err := os.WriteFile(destPath, content, 0644); err != nil {
			return fmt.Errorf("failed to write rule file %s: %w", destPath, err)
		}
	}

	log.Info("✅ Successfully processed %d rule(s) for Claude Code", len(ruleFiles))
	return nil
}

// OpenCodeRulesProcessor handles rules processing for OpenCode
type OpenCodeRulesProcessor struct {
	workDir string
}

// OpenCodeConfig represents the opencode.json configuration structure
type OpenCodeConfig struct {
	Instructions []string `json:"instructions"`
}

// NewOpenCodeRulesProcessor creates a new OpenCode rules processor
func NewOpenCodeRulesProcessor(workDir string) *OpenCodeRulesProcessor {
	return &OpenCodeRulesProcessor{
		workDir: workDir,
	}
}

// ProcessRules implements RulesProcessor for OpenCode
// It creates an opencode.json with an instructions array that references the ccagent
// rules directory directly using a glob pattern. OpenCode will load rules from there
// without needing to copy files.
func (p *OpenCodeRulesProcessor) ProcessRules() error {
	log.Info("📋 Processing rules for OpenCode agent")

	// Get rule files from ccagent directory
	ruleFiles, err := GetRuleFiles()
	if err != nil {
		return fmt.Errorf("failed to get rule files: %w", err)
	}

	if len(ruleFiles) == 0 {
		log.Info("📋 No rules found in ccagent rules directory")
		return nil
	}

	log.Info("📋 Found %d rule file(s) to process", len(ruleFiles))

	// Get home directory for OpenCode config
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	opencodeConfigDir := filepath.Join(homeDir, ".config", "opencode")

	// Ensure OpenCode config directory exists
	if err := os.MkdirAll(opencodeConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create OpenCode config directory: %w", err)
	}

	// Generate opencode.json with instructions pointing to the ccagent rules directory
	// Using glob pattern with ~ prefix which OpenCode expands to home directory
	opencodeConfigPath := filepath.Join(opencodeConfigDir, "opencode.json")

	config := OpenCodeConfig{
		Instructions: []string{"~/.config/ccagent/rules/*.md"},
	}

	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal opencode.json: %w", err)
	}

	log.Info("📋 Creating opencode.json at: %s", opencodeConfigPath)

	if err := os.WriteFile(opencodeConfigPath, configJSON, 0644); err != nil {
		return fmt.Errorf("failed to write opencode.json: %w", err)
	}

	// Clean up old OpenCode rules directory if it exists (from previous approach)
	oldOpencodeRulesDir := filepath.Join(opencodeConfigDir, "rules")
	if err := os.RemoveAll(oldOpencodeRulesDir); err != nil && !os.IsNotExist(err) {
		log.Info("⚠️  Failed to remove old OpenCode rules directory: %v", err)
	}

	log.Info("✅ Successfully processed %d rule(s) for OpenCode", len(ruleFiles))
	return nil
}

// NoOpRulesProcessor is a no-op implementation for agents that don't support rules
type NoOpRulesProcessor struct{}

// NewNoOpRulesProcessor creates a new no-op rules processor
func NewNoOpRulesProcessor() *NoOpRulesProcessor {
	return &NoOpRulesProcessor{}
}

// ProcessRules implements RulesProcessor with no operation
func (p *NoOpRulesProcessor) ProcessRules() error {
	log.Info("📋 Rules processing not supported for this agent type")
	return nil
}
