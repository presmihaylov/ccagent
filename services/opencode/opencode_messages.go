package opencode

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
)

// OpenCodeMessage represents a simplified message interface for OpenCode
type OpenCodeMessage interface {
	GetType() string
	GetSessionID() string
}

// OpenCodeTextMessage represents a text message from OpenCode containing the response
type OpenCodeTextMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionID"`
	Part      struct {
		Text string `json:"text"`
	} `json:"part"`
}

func (t OpenCodeTextMessage) GetType() string {
	return t.Type
}

func (t OpenCodeTextMessage) GetSessionID() string {
	return t.SessionID
}

// OpenCodeStepMessage represents step_start or step_finish messages
type OpenCodeStepMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionID"`
}

func (s OpenCodeStepMessage) GetType() string {
	return s.Type
}

func (s OpenCodeStepMessage) GetSessionID() string {
	return s.SessionID
}

// UnknownOpenCodeMessage represents an unknown message type from OpenCode
type UnknownOpenCodeMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionID"`
}

func (u UnknownOpenCodeMessage) GetType() string {
	return u.Type
}

func (u UnknownOpenCodeMessage) GetSessionID() string {
	return u.SessionID
}

// OpenCodeRawErrorMessage represents a non-JSON error output from OpenCode
// This happens when opencode itself crashes or encounters a startup error
type OpenCodeRawErrorMessage struct {
	RawOutput string
}

func (e OpenCodeRawErrorMessage) GetType() string {
	return "raw_error"
}

func (e OpenCodeRawErrorMessage) GetSessionID() string {
	return ""
}

// MapOpenCodeOutputToMessages parses OpenCode command output into structured messages
func MapOpenCodeOutputToMessages(output string) ([]OpenCodeMessage, error) {
	var messages []OpenCodeMessage

	// Check if the output looks like it might be a non-JSON error
	// OpenCode JSON output always starts with a '{' character on the first non-empty line
	trimmedOutput := strings.TrimSpace(output)
	if trimmedOutput != "" && !strings.HasPrefix(trimmedOutput, "{") {
		// This is likely a raw error output (e.g., JavaScript stack trace)
		// Return it as a raw error message
		return []OpenCodeMessage{
			OpenCodeRawErrorMessage{RawOutput: trimmedOutput},
		}, nil
	}

	// Use a scanner with a larger buffer to handle long lines
	scanner := bufio.NewScanner(strings.NewReader(output))
	// Set a 5MB buffer to handle long JSON lines
	scanner.Buffer(make([]byte, 0, 5*1024*1024), 5*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Parse the message
		message := parseOpenCodeMessage([]byte(line))
		messages = append(messages, message)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

// parseOpenCodeMessage attempts to parse a JSON line into the appropriate message type
func parseOpenCodeMessage(lineBytes []byte) OpenCodeMessage {
	// First, extract just the type to determine which struct to use
	var typeCheck struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(lineBytes, &typeCheck); err != nil {
		// If we can't even parse the type, return unknown message
		return UnknownOpenCodeMessage{
			Type:      "unknown",
			SessionID: "",
		}
	}

	// Parse based on type
	switch typeCheck.Type {
	case "text":
		var textMsg OpenCodeTextMessage
		if err := json.Unmarshal(lineBytes, &textMsg); err == nil {
			return textMsg
		}

	case "step_start", "step_finish":
		var stepMsg OpenCodeStepMessage
		if err := json.Unmarshal(lineBytes, &stepMsg); err == nil {
			return stepMsg
		}
	}

	// For all other types, extract basic info for unknown message
	var unknownMsg struct {
		Type      string `json:"type"`
		SessionID string `json:"sessionID"`
	}

	if err := json.Unmarshal(lineBytes, &unknownMsg); err == nil {
		return UnknownOpenCodeMessage{
			Type:      unknownMsg.Type,
			SessionID: unknownMsg.SessionID,
		}
	}

	// Return default unknown message
	return UnknownOpenCodeMessage{
		Type:      "unknown",
		SessionID: "",
	}
}

// ExtractOpenCodeSessionID extracts session ID from OpenCode messages
func ExtractOpenCodeSessionID(messages []OpenCodeMessage) string {
	// Session ID is present in all messages, take from the first one
	if len(messages) > 0 {
		sessionID := messages[0].GetSessionID()
		if sessionID != "" {
			return sessionID
		}
	}
	return "unknown"
}

// ExtractOpenCodeResult extracts the result text from OpenCode messages
func ExtractOpenCodeResult(messages []OpenCodeMessage) (string, error) {
	// Check for raw error messages first - these indicate opencode crashed or failed to start
	for _, msg := range messages {
		if rawErr, ok := msg.(OpenCodeRawErrorMessage); ok {
			// Extract a meaningful error message from the raw output
			errorSummary := extractErrorSummary(rawErr.RawOutput)
			return "", fmt.Errorf("opencode error: %s", errorSummary)
		}
	}

	// Collect all text messages and concatenate their content
	var textParts []string

	for _, msg := range messages {
		if textMsg, ok := msg.(OpenCodeTextMessage); ok {
			if textMsg.Part.Text != "" {
				textParts = append(textParts, textMsg.Part.Text)
			}
		}
	}

	if len(textParts) == 0 {
		return "", fmt.Errorf("no text message found")
	}

	// Join all text parts (in case there are multiple text messages)
	return strings.Join(textParts, ""), nil
}

// extractErrorSummary extracts a meaningful error summary from raw opencode error output
// It looks for common error patterns like "Error:", exception names, etc.
func extractErrorSummary(rawOutput string) string {
	lines := strings.Split(rawOutput, "\n")

	// Look for lines containing common error indicators
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		// Look for JavaScript/TypeScript error patterns
		if strings.Contains(trimmedLine, "Error:") {
			// Return the error line, but truncate if too long
			if len(trimmedLine) > 200 {
				return trimmedLine[:200] + "..."
			}
			return trimmedLine
		}
	}

	// If no specific error line found, return first non-empty line truncated
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine != "" {
			if len(trimmedLine) > 200 {
				return trimmedLine[:200] + "..."
			}
			return trimmedLine
		}
	}

	// Fallback
	if len(rawOutput) > 200 {
		return rawOutput[:200] + "..."
	}
	return rawOutput
}
