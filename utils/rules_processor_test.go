package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test front matter parsing

func TestParseFrontMatter_Valid(t *testing.T) {
	// Create temporary file with front matter
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test-rule.md")

	content := `---
title: Code Style Guidelines
description: Use this to learn what style guidelines to follow in this codebase
---

# Code Style

Some content here.`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Parse front matter
	frontMatter, err := ParseFrontMatter(testFile)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if frontMatter.Title != "Code Style Guidelines" {
		t.Errorf("Expected title 'Code Style Guidelines', got: %s", frontMatter.Title)
	}

	if frontMatter.Description != "Use this to learn what style guidelines to follow in this codebase" {
		t.Errorf("Expected description, got: %s", frontMatter.Description)
	}
}

func TestParseFrontMatter_NoFrontMatter(t *testing.T) {
	// Create temporary file without front matter
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test-rule.md")

	content := `# Code Style

Some content without front matter.`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Parse front matter
	frontMatter, err := ParseFrontMatter(testFile)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should return empty front matter
	if frontMatter.Title != "" {
		t.Errorf("Expected empty title, got: %s", frontMatter.Title)
	}

	if frontMatter.Description != "" {
		t.Errorf("Expected empty description, got: %s", frontMatter.Description)
	}
}

func TestParseFrontMatter_PartialFrontMatter(t *testing.T) {
	// Create temporary file with partial front matter
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test-rule.md")

	content := `---
title: Code Style Guidelines
---

# Code Style`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Parse front matter
	frontMatter, err := ParseFrontMatter(testFile)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if frontMatter.Title != "Code Style Guidelines" {
		t.Errorf("Expected title 'Code Style Guidelines', got: %s", frontMatter.Title)
	}

	if frontMatter.Description != "" {
		t.Errorf("Expected empty description, got: %s", frontMatter.Description)
	}
}

func TestParseFrontMatter_CaseInsensitive(t *testing.T) {
	// Create temporary file with mixed case keys
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test-rule.md")

	content := `---
Title: Code Style Guidelines
Description: Use this guide
---

# Code Style`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Parse front matter
	frontMatter, err := ParseFrontMatter(testFile)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if frontMatter.Title != "Code Style Guidelines" {
		t.Errorf("Expected title, got: %s", frontMatter.Title)
	}

	if frontMatter.Description != "Use this guide" {
		t.Errorf("Expected description, got: %s", frontMatter.Description)
	}
}

// Test GetRuleFiles

func TestGetRuleFiles_EmptyDirectory(t *testing.T) {
	// Create temporary ccagent rules directory
	tempDir := t.TempDir()
	rulesDir := filepath.Join(tempDir, ".config", "ccagent", "rules")

	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatalf("Failed to create rules directory: %v", err)
	}

	// Temporarily override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Get rule files
	files, err := GetRuleFiles()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("Expected 0 files, got: %d", len(files))
	}
}

func TestGetRuleFiles_WithMarkdownFiles(t *testing.T) {
	// Create temporary ccagent rules directory
	tempDir := t.TempDir()
	rulesDir := filepath.Join(tempDir, ".config", "ccagent", "rules")

	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatalf("Failed to create rules directory: %v", err)
	}

	// Create test markdown files
	testFiles := []string{"rule1.md", "rule2.md", "rule3.MD"}
	for _, file := range testFiles {
		filePath := filepath.Join(rulesDir, file)
		if err := os.WriteFile(filePath, []byte("# Test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create non-markdown file (should be ignored)
	if err := os.WriteFile(filepath.Join(rulesDir, "readme.txt"), []byte("text"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Temporarily override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Get rule files
	files, err := GetRuleFiles()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("Expected 3 files, got: %d", len(files))
	}
}

func TestGetRuleFiles_NonexistentDirectory(t *testing.T) {
	// Use temporary directory that doesn't have rules directory
	tempDir := t.TempDir()

	// Temporarily override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Get rule files (should return empty list, not error)
	files, err := GetRuleFiles()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("Expected 0 files, got: %d", len(files))
	}
}

// Test ClaudeCodeRulesProcessor

func TestClaudeCodeRulesProcessor_NoRules(t *testing.T) {
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

	// Process rules (should succeed with no rules)
	processor := NewClaudeCodeRulesProcessor(workDir)
	if err := processor.ProcessRules(); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify no .claude/rules directory was created
	claudeRulesDir := filepath.Join(workDir, ".claude", "rules")
	if _, err := os.Stat(claudeRulesDir); !os.IsNotExist(err) {
		// Directory should not exist if no rules
		// Actually, it will be created but empty - let's check if it's empty
		entries, _ := os.ReadDir(claudeRulesDir)
		if len(entries) > 0 {
			t.Errorf("Expected empty rules directory, found %d files", len(entries))
		}
	}
}

func TestClaudeCodeRulesProcessor_WithRules(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	workDir := filepath.Join(tempDir, "workspace")
	rulesDir := filepath.Join(tempDir, ".config", "ccagent", "rules")

	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work directory: %v", err)
	}

	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatalf("Failed to create rules directory: %v", err)
	}

	// Create test rule files
	testRules := map[string]string{
		"code-style.md": "# Code Style\nFollow these guidelines.",
		"testing.md":    "# Testing\nWrite tests for everything.",
	}

	for filename, content := range testRules {
		filePath := filepath.Join(rulesDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test rule: %v", err)
		}
	}

	// Temporarily override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Process rules
	processor := NewClaudeCodeRulesProcessor(workDir)
	if err := processor.ProcessRules(); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify .claude/rules directory was created
	claudeRulesDir := filepath.Join(workDir, ".claude", "rules")
	if _, err := os.Stat(claudeRulesDir); os.IsNotExist(err) {
		t.Fatalf("Expected .claude/rules directory to exist")
	}

	// Verify rule files were copied
	for filename, expectedContent := range testRules {
		destPath := filepath.Join(claudeRulesDir, filename)

		content, err := os.ReadFile(destPath)
		if err != nil {
			t.Errorf("Expected rule file %s to exist, got error: %v", filename, err)
			continue
		}

		if string(content) != expectedContent {
			t.Errorf("Expected content '%s', got '%s'", expectedContent, string(content))
		}
	}
}

func TestClaudeCodeRulesProcessor_RemovesStaleRules(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	workDir := filepath.Join(tempDir, "workspace")
	rulesDir := filepath.Join(tempDir, ".config", "ccagent", "rules")
	claudeRulesDir := filepath.Join(workDir, ".claude", "rules")

	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work directory: %v", err)
	}

	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatalf("Failed to create rules directory: %v", err)
	}

	if err := os.MkdirAll(claudeRulesDir, 0755); err != nil {
		t.Fatalf("Failed to create claude rules directory: %v", err)
	}

	// Create a stale rule in .claude/rules
	staleRulePath := filepath.Join(claudeRulesDir, "stale-rule.md")
	if err := os.WriteFile(staleRulePath, []byte("# Stale"), 0644); err != nil {
		t.Fatalf("Failed to create stale rule: %v", err)
	}

	// Create a fresh rule in ccagent rules directory
	freshRulePath := filepath.Join(rulesDir, "fresh-rule.md")
	if err := os.WriteFile(freshRulePath, []byte("# Fresh"), 0644); err != nil {
		t.Fatalf("Failed to create fresh rule: %v", err)
	}

	// Temporarily override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Process rules
	processor := NewClaudeCodeRulesProcessor(workDir)
	if err := processor.ProcessRules(); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify stale rule was removed
	if _, err := os.Stat(staleRulePath); !os.IsNotExist(err) {
		t.Errorf("Expected stale rule to be removed")
	}

	// Verify fresh rule exists
	freshDestPath := filepath.Join(claudeRulesDir, "fresh-rule.md")
	if _, err := os.Stat(freshDestPath); os.IsNotExist(err) {
		t.Errorf("Expected fresh rule to exist")
	}
}

// Test OpenCodeRulesProcessor

func TestOpenCodeRulesProcessor_NoRules(t *testing.T) {
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

	// Process rules (should succeed with no rules)
	processor := NewOpenCodeRulesProcessor(workDir)
	if err := processor.ProcessRules(); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify AGENTS.md was not created
	agentsmdPath := filepath.Join(tempDir, ".config", "opencode", "AGENTS.md")
	if _, err := os.Stat(agentsmdPath); !os.IsNotExist(err) {
		t.Errorf("Expected AGENTS.md not to exist when no rules")
	}
}

func TestOpenCodeRulesProcessor_WithRules(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	workDir := filepath.Join(tempDir, "workspace")
	rulesDir := filepath.Join(tempDir, ".config", "ccagent", "rules")

	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work directory: %v", err)
	}

	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatalf("Failed to create rules directory: %v", err)
	}

	// Create test rule files with front matter
	rule1 := `---
title: Code Style Guidelines
description: Use this to learn coding standards
---

# Code Style
Follow these guidelines.`

	rule2 := `---
title: Testing Best Practices
description: How to write effective tests
---

# Testing
Write tests for everything.`

	if err := os.WriteFile(filepath.Join(rulesDir, "code-style.md"), []byte(rule1), 0644); err != nil {
		t.Fatalf("Failed to create rule1: %v", err)
	}

	if err := os.WriteFile(filepath.Join(rulesDir, "testing.md"), []byte(rule2), 0644); err != nil {
		t.Fatalf("Failed to create rule2: %v", err)
	}

	// Temporarily override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Process rules
	processor := NewOpenCodeRulesProcessor(workDir)
	if err := processor.ProcessRules(); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify rules were copied
	opencodeRulesDir := filepath.Join(tempDir, ".config", "opencode", "rules")
	for _, filename := range []string{"code-style.md", "testing.md"} {
		destPath := filepath.Join(opencodeRulesDir, filename)
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			t.Errorf("Expected rule file %s to exist", filename)
		}
	}

	// Verify AGENTS.md was created
	agentsmdPath := filepath.Join(tempDir, ".config", "opencode", "AGENTS.md")
	content, err := os.ReadFile(agentsmdPath)
	if err != nil {
		t.Fatalf("Expected AGENTS.md to exist, got error: %v", err)
	}

	agentsmdContent := string(content)

	// Verify AGENTS.md contains expected content
	if !strings.Contains(agentsmdContent, "Code Style Guidelines") {
		t.Errorf("Expected AGENTS.md to contain 'Code Style Guidelines'")
	}

	if !strings.Contains(agentsmdContent, "Testing Best Practices") {
		t.Errorf("Expected AGENTS.md to contain 'Testing Best Practices'")
	}

	if !strings.Contains(agentsmdContent, "./rules/code-style.md") {
		t.Errorf("Expected AGENTS.md to contain location './rules/code-style.md'")
	}

	if !strings.Contains(agentsmdContent, "./rules/testing.md") {
		t.Errorf("Expected AGENTS.md to contain location './rules/testing.md'")
	}

	if !strings.Contains(agentsmdContent, "Use this to learn coding standards") {
		t.Errorf("Expected AGENTS.md to contain description")
	}

	if !strings.Contains(agentsmdContent, "IMPORTANT") {
		t.Errorf("Expected AGENTS.md to contain IMPORTANT instructions")
	}
}

func TestOpenCodeRulesProcessor_WithoutFrontMatter(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	workDir := filepath.Join(tempDir, "workspace")
	rulesDir := filepath.Join(tempDir, ".config", "ccagent", "rules")

	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work directory: %v", err)
	}

	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatalf("Failed to create rules directory: %v", err)
	}

	// Create rule without front matter
	ruleContent := "# Simple Rule\nNo front matter here."
	if err := os.WriteFile(filepath.Join(rulesDir, "simple.md"), []byte(ruleContent), 0644); err != nil {
		t.Fatalf("Failed to create rule: %v", err)
	}

	// Temporarily override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Process rules
	processor := NewOpenCodeRulesProcessor(workDir)
	if err := processor.ProcessRules(); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify AGENTS.md was created
	agentsmdPath := filepath.Join(tempDir, ".config", "opencode", "AGENTS.md")
	content, err := os.ReadFile(agentsmdPath)
	if err != nil {
		t.Fatalf("Expected AGENTS.md to exist, got error: %v", err)
	}

	agentsmdContent := string(content)

	// Should use filename as title when no front matter
	if !strings.Contains(agentsmdContent, "simple") {
		t.Errorf("Expected AGENTS.md to contain 'simple' (from filename)")
	}

	if !strings.Contains(agentsmdContent, "./rules/simple.md") {
		t.Errorf("Expected AGENTS.md to contain location")
	}

	if !strings.Contains(agentsmdContent, "See file for details") {
		t.Errorf("Expected default description when no front matter")
	}
}

func TestOpenCodeRulesProcessor_RemovesStaleRules(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	workDir := filepath.Join(tempDir, "workspace")
	rulesDir := filepath.Join(tempDir, ".config", "ccagent", "rules")
	opencodeRulesDir := filepath.Join(tempDir, ".config", "opencode", "rules")

	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work directory: %v", err)
	}

	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatalf("Failed to create rules directory: %v", err)
	}

	if err := os.MkdirAll(opencodeRulesDir, 0755); err != nil {
		t.Fatalf("Failed to create opencode rules directory: %v", err)
	}

	// Create a stale rule in opencode rules directory
	staleRulePath := filepath.Join(opencodeRulesDir, "stale-rule.md")
	if err := os.WriteFile(staleRulePath, []byte("# Stale"), 0644); err != nil {
		t.Fatalf("Failed to create stale rule: %v", err)
	}

	// Create a fresh rule in ccagent rules directory
	freshRulePath := filepath.Join(rulesDir, "fresh-rule.md")
	if err := os.WriteFile(freshRulePath, []byte("# Fresh"), 0644); err != nil {
		t.Fatalf("Failed to create fresh rule: %v", err)
	}

	// Temporarily override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Process rules
	processor := NewOpenCodeRulesProcessor(workDir)
	if err := processor.ProcessRules(); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify stale rule was removed
	if _, err := os.Stat(staleRulePath); !os.IsNotExist(err) {
		t.Errorf("Expected stale rule to be removed")
	}

	// Verify fresh rule exists
	freshDestPath := filepath.Join(opencodeRulesDir, "fresh-rule.md")
	if _, err := os.Stat(freshDestPath); os.IsNotExist(err) {
		t.Errorf("Expected fresh rule to exist")
	}
}

// Test NoOpRulesProcessor

func TestNoOpRulesProcessor(t *testing.T) {
	processor := NewNoOpRulesProcessor()
	if err := processor.ProcessRules(); err != nil {
		t.Fatalf("Expected no error from NoOp processor, got: %v", err)
	}
}
