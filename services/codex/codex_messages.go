package codex

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
)

// CodexMessage represents a simplified message interface for Codex
type CodexMessage interface {
	GetType() string
	GetThreadID() string
}

// ThreadStartedMessage represents the initial message with thread/session ID
type ThreadStartedMessage struct {
	Type     string `json:"type"`
	ThreadID string `json:"thread_id"`
}

func (t ThreadStartedMessage) GetType() string {
	return t.Type
}

func (t ThreadStartedMessage) GetThreadID() string {
	return t.ThreadID
}

// ItemCompletedMessage represents a completed work item from Codex
type ItemCompletedMessage struct {
	Type string `json:"type"`
	Item struct {
		ID     string `json:"id"`
		Type   string `json:"type"` // "reasoning", "agent_message", "command_execution", "file_change", etc.
		Text   string `json:"text,omitempty"`
		Status string `json:"status,omitempty"`
	} `json:"item"`
}

func (i ItemCompletedMessage) GetType() string {
	return i.Type
}

func (i ItemCompletedMessage) GetThreadID() string {
	return "" // Item messages don't contain thread ID
}

// TurnCompletedMessage represents the end of an agent turn with usage stats
type TurnCompletedMessage struct {
	Type  string `json:"type"`
	Usage struct {
		InputTokens       int `json:"input_tokens"`
		CachedInputTokens int `json:"cached_input_tokens"`
		OutputTokens      int `json:"output_tokens"`
	} `json:"usage"`
}

func (t TurnCompletedMessage) GetType() string {
	return t.Type
}

func (t TurnCompletedMessage) GetThreadID() string {
	return "" // Turn completed messages don't contain thread ID
}

// UnknownCodexMessage represents an unknown message type from Codex
type UnknownCodexMessage struct {
	Type     string `json:"type"`
	ThreadID string `json:"thread_id,omitempty"`
}

func (u UnknownCodexMessage) GetType() string {
	return u.Type
}

func (u UnknownCodexMessage) GetThreadID() string {
	return u.ThreadID
}

// MapCodexOutputToMessages parses Codex command output into structured messages
// This is exported to allow reuse across different modules
func MapCodexOutputToMessages(output string) ([]CodexMessage, error) {
	var messages []CodexMessage

	// Use a scanner with a larger buffer to handle long lines
	scanner := bufio.NewScanner(strings.NewReader(output))
	// Set a 10MB buffer to handle very long JSON lines (Codex can produce lines over 1MB)
	scanner.Buffer(make([]byte, 0, 10*1024*1024), 10*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Parse the message
		message := parseCodexMessage([]byte(line))
		messages = append(messages, message)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

// parseCodexMessage attempts to parse a JSON line into the appropriate message type
func parseCodexMessage(lineBytes []byte) CodexMessage {
	// First, extract just the type to determine which struct to use
	var typeCheck struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(lineBytes, &typeCheck); err != nil {
		// If we can't even parse the type, return unknown message
		return UnknownCodexMessage{
			Type: "unknown",
		}
	}

	// Parse based on type
	switch typeCheck.Type {
	case "thread.started":
		var threadMsg ThreadStartedMessage
		if err := json.Unmarshal(lineBytes, &threadMsg); err == nil {
			return threadMsg
		}

	case "item.completed", "item.started":
		var itemMsg ItemCompletedMessage
		if err := json.Unmarshal(lineBytes, &itemMsg); err == nil {
			return itemMsg
		}

	case "turn.completed", "turn.started":
		var turnMsg TurnCompletedMessage
		if err := json.Unmarshal(lineBytes, &turnMsg); err == nil {
			return turnMsg
		}
	}

	// For all other types, extract basic info for unknown message
	var unknownMsg struct {
		Type     string `json:"type"`
		ThreadID string `json:"thread_id,omitempty"`
	}

	if err := json.Unmarshal(lineBytes, &unknownMsg); err == nil {
		return UnknownCodexMessage{
			Type:     unknownMsg.Type,
			ThreadID: unknownMsg.ThreadID,
		}
	}

	// Return default unknown message
	return UnknownCodexMessage{
		Type: "unknown",
	}
}

// ExtractCodexThreadID extracts thread ID from Codex messages
func ExtractCodexThreadID(messages []CodexMessage) string {
	// Thread ID is in the first thread.started message
	for _, msg := range messages {
		if threadMsg, ok := msg.(ThreadStartedMessage); ok {
			return threadMsg.ThreadID
		}
	}
	return "unknown"
}

// ExtractCodexResult extracts the final agent message text from Codex messages
func ExtractCodexResult(messages []CodexMessage) (string, error) {
	// Look for the last item.completed message with item.type == "agent_message"
	for i := len(messages) - 1; i >= 0; i-- {
		if itemMsg, ok := messages[i].(ItemCompletedMessage); ok {
			if itemMsg.Item.Type == "agent_message" && itemMsg.Item.Text != "" {
				return itemMsg.Item.Text, nil
			}
		}
	}
	return "", fmt.Errorf("no agent_message item found")
}
