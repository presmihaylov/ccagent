package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ccagent/clients"
	"ccagent/core"
	"ccagent/core/log"
	"ccagent/services"
)

type ClaudeService struct {
	claudeClient clients.ClaudeClient
	logDir       string
}

func NewClaudeService(claudeClient clients.ClaudeClient, logDir string) *ClaudeService {
	return &ClaudeService{
		claudeClient: claudeClient,
		logDir:       logDir,
	}
}

// writeClaudeSessionLog writes Claude output to a timestamped log file and returns the filepath
func (c *ClaudeService) writeClaudeSessionLog(rawOutput string) (string, error) {
	if err := os.MkdirAll(c.logDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create log directory: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("claude-session-%s.log", timestamp)
	filepath := filepath.Join(c.logDir, filename)

	if err := os.WriteFile(filepath, []byte(rawOutput), 0600); err != nil {
		return "", fmt.Errorf("failed to write log file: %w", err)
	}

	return filepath, nil
}

// CleanupOldLogs removes log files older than the specified number of days
func (c *ClaudeService) CleanupOldLogs(maxAgeDays int) error {
	log.Info("📋 Starting to cleanup old Claude session logs older than %d days", maxAgeDays)

	if maxAgeDays <= 0 {
		return fmt.Errorf("maxAgeDays must be greater than 0")
	}

	files, err := os.ReadDir(c.logDir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info("📋 Log directory does not exist, nothing to clean up")
			return nil
		}
		return fmt.Errorf("failed to read log directory: %w", err)
	}

	cutoffTime := time.Now().AddDate(0, 0, -maxAgeDays)
	removedCount := 0

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Only clean up claude session log files
		if !strings.HasPrefix(file.Name(), "claude-session-") || !strings.HasSuffix(file.Name(), ".log") {
			continue
		}

		filePath := filepath.Join(c.logDir, file.Name())
		info, err := file.Info()
		if err != nil {
			log.Error("Failed to get file info for %s: %v", filePath, err)
			continue
		}

		if info.ModTime().Before(cutoffTime) {
			if err := os.Remove(filePath); err != nil {
				log.Error("Failed to remove old log file %s: %v", filePath, err)
				continue
			}
			removedCount++
		}
	}

	log.Info("📋 Completed successfully - removed %d old Claude session log files", removedCount)
	return nil
}

func (c *ClaudeService) StartNewConversation(prompt string) (*services.CLIAgentResult, error) {
	return c.StartNewConversationWithOptions(prompt, nil)
}

func (c *ClaudeService) StartNewConversationWithOptions(
	prompt string,
	options *clients.ClaudeOptions,
) (*services.CLIAgentResult, error) {
	log.Info("📋 Starting to start new Claude conversation")
	rawOutput, err := c.claudeClient.StartNewSession(prompt, options)
	if err != nil {
		log.Error("Failed to start new Claude session: %v", err)
		return nil, c.handleClaudeClientError(err, "failed to start new Claude session")
	}

	// Always log the Claude session
	logPath, writeErr := c.writeClaudeSessionLog(rawOutput)
	if writeErr != nil {
		log.Error("Failed to write Claude session log: %v", writeErr)
	}

	messages, err := services.MapClaudeOutputToMessages(rawOutput)
	if err != nil {
		log.Error("Failed to parse Claude output: %v", err)

		return nil, &core.ClaudeParseError{
			Message:     fmt.Sprintf("couldn't parse claude response and instead stored the response in %s", logPath),
			LogFilePath: logPath,
			OriginalErr: err,
		}
	}

	sessionID := c.extractSessionID(messages)
	output, err := c.extractClaudeResult(messages)
	if err != nil {
		log.Error("Failed to extract Claude result: %v", err)
		return nil, fmt.Errorf("failed to extract Claude result: %w", err)
	}

	log.Info("📋 Claude response extracted successfully, session: %s, output length: %d", sessionID, len(output))
	result := &services.CLIAgentResult{
		Output:    output,
		SessionID: sessionID,
	}

	log.Info("📋 Completed successfully - started new Claude conversation with session: %s", sessionID)
	return result, nil
}

func (c *ClaudeService) StartNewConversationWithSystemPrompt(
	prompt, systemPrompt string,
) (*services.CLIAgentResult, error) {
	return c.StartNewConversationWithOptions(prompt, &clients.ClaudeOptions{
		SystemPrompt: systemPrompt,
	})
}

func (c *ClaudeService) StartNewConversationWithDisallowedTools(
	prompt string,
	disallowedTools []string,
) (*services.CLIAgentResult, error) {
	return c.StartNewConversationWithOptions(prompt, &clients.ClaudeOptions{
		DisallowedTools: disallowedTools,
	})
}

func (c *ClaudeService) ContinueConversation(sessionID, prompt string) (*services.CLIAgentResult, error) {
	return c.ContinueConversationWithOptions(sessionID, prompt, nil)
}

func (c *ClaudeService) ContinueConversationWithOptions(
	sessionID, prompt string,
	options *clients.ClaudeOptions,
) (*services.CLIAgentResult, error) {
	log.Info("📋 Starting to continue Claude conversation: %s", sessionID)
	rawOutput, err := c.claudeClient.ContinueSession(sessionID, prompt, options)
	if err != nil {
		log.Error("Failed to continue Claude session: %v", err)
		return nil, c.handleClaudeClientError(err, "failed to continue Claude session")
	}

	// Always log the Claude session
	logPath, writeErr := c.writeClaudeSessionLog(rawOutput)
	if writeErr != nil {
		log.Error("Failed to write Claude session log: %v", writeErr)
	}

	messages, err := services.MapClaudeOutputToMessages(rawOutput)
	if err != nil {
		log.Error("Failed to parse Claude output: %v", err)

		return nil, &core.ClaudeParseError{
			Message:     fmt.Sprintf("couldn't parse claude response and instead stored the response in %s", logPath),
			LogFilePath: logPath,
			OriginalErr: err,
		}
	}

	actualSessionID := c.extractSessionID(messages)
	output, err := c.extractClaudeResult(messages)
	if err != nil {
		log.Error("Failed to extract Claude result: %v", err)
		return nil, fmt.Errorf("failed to extract Claude result: %w", err)
	}

	log.Info("📋 Claude response extracted successfully, session: %s, output length: %d", actualSessionID, len(output))
	result := &services.CLIAgentResult{
		Output:    output,
		SessionID: actualSessionID,
	}

	log.Info("📋 Completed successfully - continued Claude conversation with session: %s", actualSessionID)
	return result, nil
}

func (c *ClaudeService) extractSessionID(messages []services.ClaudeMessage) string {
	if len(messages) > 0 {
		return messages[0].GetSessionID()
	}
	return "unknown"
}

func (c *ClaudeService) extractClaudeResult(messages []services.ClaudeMessage) (string, error) {
	// First priority: Look for ExitPlanMode messages (highest priority)
	for i := len(messages) - 1; i >= 0; i-- {
		if exitPlanMsg, ok := messages[i].(services.ExitPlanModeMessage); ok {
			plan := exitPlanMsg.GetPlan()
			if plan != "" {
				return plan, nil
			}
		}
	}

	// Second priority: Look for result message type
	for i := len(messages) - 1; i >= 0; i-- {
		if resultMsg, ok := messages[i].(services.ResultMessage); ok {
			if resultMsg.Result != "" {
				return resultMsg.Result, nil
			}
		}
	}

	// Third priority: Fallback to assistant message (existing approach)
	for i := len(messages) - 1; i >= 0; i-- {
		if assistantMsg, ok := messages[i].(services.AssistantMessage); ok {
			for _, contentRaw := range assistantMsg.Message.Content {
				// Parse the content to check if it's a text content item
				var contentItem struct {
					Type string `json:"type"`
					Text string `json:"text,omitempty"`
				}
				if err := json.Unmarshal(contentRaw, &contentItem); err == nil {
					if contentItem.Type == "text" && contentItem.Text != "" {
						return contentItem.Text, nil
					}
				}
			}
		}
	}
	return "", fmt.Errorf("no ExitPlanMode, result, or assistant message with text content found")
}

// handleClaudeClientError processes errors from Claude client calls.
// If the error is a Claude command error, it attempts to extract the assistant message
// and returns a new error with the clean message. Otherwise, returns the original error.
func (c *ClaudeService) handleClaudeClientError(err error, operation string) error {
	if err == nil {
		return nil
	}

	// Check if this is a Claude command error
	claudeErr, isClaudeErr := core.IsClaudeCommandErr(err)
	if !isClaudeErr {
		// Not a Claude command error, return original error wrapped
		return fmt.Errorf("%s: %w", operation, err)
	}

	// Try to parse the output as Claude messages using internal parsing
	messages, parseErr := services.MapClaudeOutputToMessages(claudeErr.Output)
	if parseErr != nil {
		// If parsing fails, return original error wrapped
		log.Error("Failed to parse Claude output from error: %v", parseErr)
		return fmt.Errorf("%s: %w", operation, err)
	}

	// First priority: Try to extract ExitPlanMode message (highest priority)
	for i := len(messages) - 1; i >= 0; i-- {
		if exitPlanMsg, ok := messages[i].(services.ExitPlanModeMessage); ok {
			plan := exitPlanMsg.GetPlan()
			if plan != "" {
				log.Info("✅ Successfully extracted Claude ExitPlanMode message from error: %s", plan)
				return fmt.Errorf("%s: %s", operation, plan)
			}
		}
	}

	// Second priority: Try to extract the result message
	for i := len(messages) - 1; i >= 0; i-- {
		if resultMsg, ok := messages[i].(services.ResultMessage); ok {
			if resultMsg.Result != "" {
				log.Info("✅ Successfully extracted Claude result message from error: %s", resultMsg.Result)
				return fmt.Errorf("%s: %s", operation, resultMsg.Result)
			}
		}
	}

	// Third priority: Fallback to assistant message (existing logic)
	for i := len(messages) - 1; i >= 0; i-- {
		if assistantMsg, ok := messages[i].(services.AssistantMessage); ok {
			for _, contentRaw := range assistantMsg.Message.Content {
				// Parse the content to check if it's a text content item
				var contentItem struct {
					Type string `json:"type"`
					Text string `json:"text,omitempty"`
				}
				if err := json.Unmarshal(contentRaw, &contentItem); err == nil {
					if contentItem.Type == "text" && contentItem.Text != "" {
						log.Info("✅ Successfully extracted Claude assistant message from error: %s", contentItem.Text)
						return fmt.Errorf("%s: %s", operation, contentItem.Text)
					}
				}
			}
		}
	}

	// No assistant message found, return original error wrapped
	log.Info("⚠️ No assistant message found in Claude command output, returning original error")
	return fmt.Errorf("%s: %w", operation, err)
}

// AgentName identifies this service implementation
func (c *ClaudeService) AgentName() string {
	return "claude"
}
