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
	defer file.Close()

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

// ClaudeCodeRulesProcessor handles rules processing for Claude Code
type ClaudeCodeRulesProcessor struct {
	workDir string
}

// NewClaudeCodeRulesProcessor creates a new Claude Code rules processor
func NewClaudeCodeRulesProcessor(workDir string) *ClaudeCodeRulesProcessor {
	return &ClaudeCodeRulesProcessor{
		workDir: workDir,
	}
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

	// Create .claude/rules directory in work directory
	claudeRulesDir := filepath.Join(p.workDir, ".claude", "rules")

	// Remove existing rules directory to avoid stale rules
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
// It copies rule files to ~/.config/opencode/rules/ and creates an opencode.json
// with an instructions array that references those rule files using absolute paths.
// This allows OpenCode to automatically load the rules into its context.
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
	opencodeRulesDir := filepath.Join(opencodeConfigDir, "rules")

	// Remove existing rules directory to avoid stale rules
	if err := os.RemoveAll(opencodeRulesDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing rules directory: %w", err)
	}

	// Create fresh rules directory
	if err := os.MkdirAll(opencodeRulesDir, 0755); err != nil {
		return fmt.Errorf("failed to create OpenCode rules directory: %w", err)
	}

	// Copy each rule file
	for _, ruleFile := range ruleFiles {
		fileName := filepath.Base(ruleFile)
		destPath := filepath.Join(opencodeRulesDir, fileName)

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

	// Generate opencode.json with instructions pointing to the rules directory
	// Using glob pattern with ~ prefix which OpenCode expands to home directory
	opencodeConfigPath := filepath.Join(opencodeConfigDir, "opencode.json")

	config := OpenCodeConfig{
		Instructions: []string{"~/.config/opencode/rules/*.md"},
	}

	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal opencode.json: %w", err)
	}

	log.Info("📋 Creating opencode.json at: %s", opencodeConfigPath)

	if err := os.WriteFile(opencodeConfigPath, configJSON, 0644); err != nil {
		return fmt.Errorf("failed to write opencode.json: %w", err)
	}

	// Remove old AGENTS.md if it exists (cleanup from previous approach)
	oldAgentsmdPath := filepath.Join(opencodeConfigDir, "AGENTS.md")
	if err := os.Remove(oldAgentsmdPath); err != nil && !os.IsNotExist(err) {
		log.Info("⚠️  Failed to remove old AGENTS.md: %v", err)
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
