package services

import (
	"bufio"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// ClaudeMessage represents a message from Claude command output
type ClaudeMessage interface {
	GetType() string
	GetSessionID() string
}

// AssistantMessage represents an assistant message from Claude
type AssistantMessage struct {
	Type    string `json:"type"`
	Message struct {
		ID         string            `json:"id"`
		Type       string            `json:"type"`
		Content    []json.RawMessage `json:"content"`     // Use RawMessage to handle both text and tool_use content
		StopReason string            `json:"stop_reason"` // "end_turn" means final response, "tool_use" means more actions coming
	} `json:"message"`
	SessionID string `json:"session_id"`
}

func (a AssistantMessage) GetType() string {
	return a.Type
}

func (a AssistantMessage) GetSessionID() string {
	return a.SessionID
}

// UnknownClaudeMessage represents an unknown message type from Claude
type UnknownClaudeMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
}

func (u UnknownClaudeMessage) GetType() string {
	return u.Type
}

func (u UnknownClaudeMessage) GetSessionID() string {
	return u.SessionID
}

// SystemMessage represents a system message from Claude
type SystemMessage struct {
	Type      string `json:"type"`
	Subtype   string `json:"subtype,omitempty"`
	SessionID string `json:"session_id"`
}

func (s SystemMessage) GetType() string {
	return s.Type
}

func (s SystemMessage) GetSessionID() string {
	return s.SessionID
}

// UserMessage represents a user message from Claude
type UserMessage struct {
	Type    string `json:"type"`
	Message struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"` // Can be string or array
	} `json:"message"`
	SessionID string `json:"session_id"`
}

func (u UserMessage) GetType() string {
	return u.Type
}

func (u UserMessage) GetSessionID() string {
	return u.SessionID
}

// ResultMessage represents a result message from Claude
type ResultMessage struct {
	Type          string  `json:"type"`
	Subtype       string  `json:"subtype"`
	IsError       bool    `json:"is_error"`
	DurationMs    int     `json:"duration_ms"`
	DurationAPIMs int     `json:"duration_api_ms"`
	NumTurns      int     `json:"num_turns"`
	Result        string  `json:"result"`
	SessionID     string  `json:"session_id"`
	TotalCostUsd  float64 `json:"total_cost_usd"`
}

func (r ResultMessage) GetType() string {
	return r.Type
}

func (r ResultMessage) GetSessionID() string {
	return r.SessionID
}

// ExitPlanModeMessage represents an assistant message containing ExitPlanMode tool use
type ExitPlanModeMessage struct {
	Type    string `json:"type"`
	Message struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Role    string `json:"role"`
		Model   string `json:"model"`
		Content []struct {
			Type  string `json:"type"`
			ID    string `json:"id"`
			Name  string `json:"name"`
			Input struct {
				Plan string `json:"plan"`
			} `json:"input"`
		} `json:"content"`
	} `json:"message"`
	SessionID string `json:"session_id"`
}

func (e ExitPlanModeMessage) GetType() string {
	return "exit_plan_mode"
}

func (e ExitPlanModeMessage) GetSessionID() string {
	return e.SessionID
}

func (e ExitPlanModeMessage) GetPlan() string {
	if len(e.Message.Content) > 0 {
		return e.Message.Content[0].Input.Plan
	}
	return ""
}

// stripBase64Images removes large base64-encoded image data from the output to reduce memory usage.
// Images are never used by the parser (only text content is extracted), but they can make
// individual JSON lines exceed buffer limits (1MB+ for screenshots/images).
// This preserves the JSON structure but replaces the data payload with a placeholder.
func stripBase64Images(output string) string {
	// Match base64 data fields that are larger than 1KB (likely images)
	// Pattern: "data":"<long base64 string>"
	// We keep short data fields as they might be legitimate small payloads
	re := regexp.MustCompile(`("data":")([\w+/=]{1000,})(")`)
	return re.ReplaceAllString(output, `${1}[IMAGE_STRIPPED]${3}`)
}

// stripLargeToolResultContent removes large tool_result content from user messages to reduce memory usage.
// Tool results can be massive (17MB-85MB for grep results, file reads, etc.) but the parser only
// needs the final assistant response and result message. This preserves the JSON structure but
// truncates the content payload when it exceeds a threshold.
//
// The pattern matches: "type":"tool_result","content":"<very long content>"
// and replaces the content with a truncated version plus a marker.
func stripLargeToolResultContent(output string) string {
	// Match tool_result content fields that are larger than 100KB
	// Pattern: "tool_result","content":"<long string that may contain escaped chars>"
	// The content field in tool_results is always a JSON string (not an object)
	//
	// We use a regex that matches the structure and captures content up to a reasonable size
	// then replaces anything larger with a truncated version
	const maxContentSize = 100 * 1024 // 100KB threshold

	// This regex matches: "type":"tool_result","content":"
	// followed by the content string (which we'll process)
	// IMPORTANT: Use [^"\\]* instead of [^"]* to avoid consuming backslashes that are part of escape sequences
	re := regexp.MustCompile(`("type":"tool_result","content":")([^"\\]*(?:\\.[^"\\]*)*)(")`)

	return re.ReplaceAllStringFunc(output, func(match string) string {
		// Extract the parts using the same regex
		submatches := re.FindStringSubmatch(match)
		if len(submatches) != 4 {
			return match
		}

		prefix := submatches[1]  // "type":"tool_result","content":"
		content := submatches[2] // the actual content
		suffix := submatches[3]  // closing quote

		if len(content) > maxContentSize {
			// Truncate and add marker
			truncated := content[:maxContentSize] + "...[CONTENT_TRUNCATED_" + fmt.Sprintf("%d", len(content)) + "_BYTES]"
			return prefix + truncated + suffix
		}

		return match
	})
}

// stripLargeToolUseResultContent removes large stdout/stderr content from tool_use_result fields.
// These fields can be massive (64MB+) when commands output large amounts of data.
// The parser doesn't use these fields, so we can safely truncate them.
//
// The pattern matches: "tool_use_result":{"stdout":"<very long content>"...}
// and truncates the stdout and stderr fields when they exceed a threshold.
func stripLargeToolUseResultContent(output string) string {
	const maxContentSize = 100 * 1024 // 100KB threshold

	// Match stdout field: "stdout":"<content>"
	// IMPORTANT: Use [^"\\]* to properly handle escaped characters
	stdoutRe := regexp.MustCompile(`("stdout":")([^"\\]*(?:\\.[^"\\]*)*)(")`)
	output = stdoutRe.ReplaceAllStringFunc(output, func(match string) string {
		submatches := stdoutRe.FindStringSubmatch(match)
		if len(submatches) != 4 {
			return match
		}

		prefix := submatches[1]
		content := submatches[2]
		suffix := submatches[3]

		if len(content) > maxContentSize {
			truncated := content[:maxContentSize] + "...[STDOUT_TRUNCATED_" + fmt.Sprintf("%d", len(content)) + "_BYTES]"
			return prefix + truncated + suffix
		}

		return match
	})

	// Match stderr field: "stderr":"<content>"
	stderrRe := regexp.MustCompile(`("stderr":")([^"\\]*(?:\\.[^"\\]*)*)(")`)
	output = stderrRe.ReplaceAllStringFunc(output, func(match string) string {
		submatches := stderrRe.FindStringSubmatch(match)
		if len(submatches) != 4 {
			return match
		}

		prefix := submatches[1]
		content := submatches[2]
		suffix := submatches[3]

		if len(content) > maxContentSize {
			truncated := content[:maxContentSize] + "...[STDERR_TRUNCATED_" + fmt.Sprintf("%d", len(content)) + "_BYTES]"
			return prefix + truncated + suffix
		}

		return match
	})

	return output
}

// MapClaudeOutputToMessages parses Claude command output into structured messages
// This is exported to allow reuse across different modules
func MapClaudeOutputToMessages(output string) ([]ClaudeMessage, error) {
	// Strip large base64 images before parsing to reduce memory usage
	// Images are never used (only text is extracted), but can make lines exceed buffer limits
	output = stripBase64Images(output)

	// Strip large tool_result content before parsing to reduce memory usage
	// Tool results can be massive (17MB-85MB) but we only need the final assistant response
	output = stripLargeToolResultContent(output)

	// Strip large tool_use_result stdout/stderr content before parsing
	// These fields can be 64MB+ when commands output large data
	output = stripLargeToolUseResultContent(output)

	var messages []ClaudeMessage

	// Use a scanner with a larger buffer to handle long lines
	scanner := bufio.NewScanner(strings.NewReader(output))
	// Set a 4MB buffer to handle large tool_result outputs (e.g., grep results, file reads)
	// Tool results can exceed 1-2MB when reading large files or searching codebases
	const maxBufferSize = 4 * 1024 * 1024 // 4MB
	scanner.Buffer(make([]byte, 0, maxBufferSize), maxBufferSize)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Parse the message based on type
		message := parseClaudeMessage([]byte(line))
		messages = append(messages, message)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

// isExitPlanModeMessage checks if an assistant message contains ExitPlanMode tool use
func isExitPlanModeMessage(lineBytes []byte) bool {
	var tempMsg struct {
		Type    string `json:"type"`
		Message struct {
			Content []struct {
				Type string `json:"type"`
				Name string `json:"name"`
			} `json:"content"`
		} `json:"message"`
	}

	if err := json.Unmarshal(lineBytes, &tempMsg); err != nil {
		return false
	}

	if tempMsg.Type != "assistant" {
		return false
	}

	for _, content := range tempMsg.Message.Content {
		if content.Type == "tool_use" && content.Name == "ExitPlanMode" {
			return true
		}
	}

	return false
}

// parseClaudeMessage attempts to parse a JSON line into the appropriate message type
func parseClaudeMessage(lineBytes []byte) ClaudeMessage {
	// First, extract just the type to determine which struct to use
	var typeCheck struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(lineBytes, &typeCheck); err != nil {
		// If we can't even parse the type, return unknown message
		return UnknownClaudeMessage{
			Type:      "unknown",
			SessionID: "",
		}
	}

	// Parse based on type
	switch typeCheck.Type {
	case "assistant":
		// First check if this is an ExitPlanMode tool use
		if isExitPlanModeMessage(lineBytes) {
			var exitPlanMsg ExitPlanModeMessage
			if err := json.Unmarshal(lineBytes, &exitPlanMsg); err == nil {
				return exitPlanMsg
			}
		}
		// Otherwise parse as regular assistant message
		var assistantMsg AssistantMessage
		if err := json.Unmarshal(lineBytes, &assistantMsg); err == nil {
			return assistantMsg
		}
	case "system":
		var systemMsg SystemMessage
		if err := json.Unmarshal(lineBytes, &systemMsg); err == nil {
			return systemMsg
		}
	case "user":
		var userMsg UserMessage
		if err := json.Unmarshal(lineBytes, &userMsg); err == nil {
			return userMsg
		}
	case "result":
		var resultMsg ResultMessage
		if err := json.Unmarshal(lineBytes, &resultMsg); err == nil {
			return resultMsg
		}
	}

	// If specific type parsing failed, try to extract basic info for unknown message
	var unknownMsg struct {
		Type      string `json:"type"`
		SessionID string `json:"session_id"`
	}

	if err := json.Unmarshal(lineBytes, &unknownMsg); err == nil {
		return UnknownClaudeMessage{
			Type:      unknownMsg.Type,
			SessionID: unknownMsg.SessionID,
		}
	}

	// Last resort - completely unknown message
	return UnknownClaudeMessage{
		Type:      "unknown",
		SessionID: "",
	}
}
