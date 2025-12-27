package utils

import (
	"bufio"
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

// ruleInfo holds metadata about a rule file for AGENTS.md generation
type ruleInfo struct {
	fileName    string
	title       string
	description string
}

// NewOpenCodeRulesProcessor creates a new OpenCode rules processor
func NewOpenCodeRulesProcessor(workDir string) *OpenCodeRulesProcessor {
	return &OpenCodeRulesProcessor{
		workDir: workDir,
	}
}

// ProcessRules implements RulesProcessor for OpenCode
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

	// Copy each rule file and collect metadata for AGENTS.md
	var rules []ruleInfo

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

		// Parse front matter for AGENTS.md
		frontMatter, err := ParseFrontMatter(ruleFile)
		if err != nil {
			log.Info("⚠️  Failed to parse front matter for %s: %v", fileName, err)
			// Continue anyway with empty metadata
			frontMatter = &RuleFrontMatter{}
		}

		rules = append(rules, ruleInfo{
			fileName:    fileName,
			title:       frontMatter.Title,
			description: frontMatter.Description,
		})
	}

	// Generate AGENTS.md
	agentsmdPath := filepath.Join(opencodeConfigDir, "AGENTS.md")
	agentsmdContent := p.generateAGENTSmd(rules)

	log.Info("📋 Creating AGENTS.md at: %s", agentsmdPath)

	if err := os.WriteFile(agentsmdPath, []byte(agentsmdContent), 0644); err != nil {
		return fmt.Errorf("failed to write AGENTS.md: %w", err)
	}

	log.Info("✅ Successfully processed %d rule(s) for OpenCode", len(ruleFiles))
	return nil
}

// generateAGENTSmd creates the AGENTS.md content with instructions and rule enumeration
func (p *OpenCodeRulesProcessor) generateAGENTSmd(rules []ruleInfo) string {
	var sb strings.Builder

	// Write instructions header
	sb.WriteString("# Agent Rules and Guidelines\n\n")
	sb.WriteString("**IMPORTANT**: Before starting any task, carefully examine the rules below to determine which ones are relevant to the work you're about to perform. ")
	sb.WriteString("Each rule is designed to guide specific aspects of development. ")
	sb.WriteString("Apply the relevant rules thoughtfully to ensure high-quality results.\n\n")
	sb.WriteString("**Instructions**:\n")
	sb.WriteString("1. Read the task description carefully\n")
	sb.WriteString("2. Review the list of available rules below\n")
	sb.WriteString("3. Identify which rules apply to your current task\n")
	sb.WriteString("4. Read the full content of the applicable rules from their file locations\n")
	sb.WriteString("5. Apply the guidelines from those rules throughout your work\n\n")
	sb.WriteString("---\n\n")
	sb.WriteString("## Available Rules\n\n")

	// Enumerate each rule
	for _, rule := range rules {
		// Use title if available, otherwise use filename without extension
		title := rule.title
		if title == "" {
			title = strings.TrimSuffix(rule.fileName, filepath.Ext(rule.fileName))
		}

		sb.WriteString(fmt.Sprintf("### %s\n", title))
		sb.WriteString(fmt.Sprintf("**Location**: `./rules/%s`\n", rule.fileName))

		if rule.description != "" {
			sb.WriteString(fmt.Sprintf("**Description**: %s\n", rule.description))
		} else {
			sb.WriteString("**Description**: See file for details\n")
		}

		sb.WriteString("\n")
	}

	return sb.String()
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
