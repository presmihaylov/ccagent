package cursor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
)

// CursorMessage represents a simplified message interface for Cursor
type CursorMessage interface {
	GetType() string
	GetSessionID() string
}

// CursorResultMessage represents a result message from Cursor (simplified version)
type CursorResultMessage struct {
	Type      string `json:"type"`
	Result    string `json:"result"`
	SessionID string `json:"session_id"`
}

func (r CursorResultMessage) GetType() string {
	return r.Type
}

func (r CursorResultMessage) GetSessionID() string {
	return r.SessionID
}

// UnknownCursorMessage represents an unknown message type from Cursor
type UnknownCursorMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
}

func (u UnknownCursorMessage) GetType() string {
	return u.Type
}

func (u UnknownCursorMessage) GetSessionID() string {
	return u.SessionID
}

// MapCursorOutputToMessages parses Cursor command output focusing only on result messages
// This is exported to allow reuse across different modules
func MapCursorOutputToMessages(output string) ([]CursorMessage, error) {
	var messages []CursorMessage

	// Use a scanner with a larger buffer to handle long lines
	scanner := bufio.NewScanner(strings.NewReader(output))
	// Set a 1MB buffer to handle very long JSON lines
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Parse the message focusing only on result type
		message := parseCursorMessage([]byte(line))
		messages = append(messages, message)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

// parseCursorMessage attempts to parse a JSON line into the appropriate message type
func parseCursorMessage(lineBytes []byte) CursorMessage {
	// First, extract just the type to determine which struct to use
	var typeCheck struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(lineBytes, &typeCheck); err != nil {
		// If we can't even parse the type, return unknown message
		return UnknownCursorMessage{
			Type:      "unknown",
			SessionID: "",
		}
	}

	// Parse based on type - only focus on result messages for simplicity
	switch typeCheck.Type {
	case "result":
		var resultMsg CursorResultMessage
		if err := json.Unmarshal(lineBytes, &resultMsg); err == nil {
			return resultMsg
		}
	}

	// For all other types, extract basic info for unknown message
	var unknownMsg struct {
		Type      string `json:"type"`
		SessionID string `json:"session_id"`
	}

	if err := json.Unmarshal(lineBytes, &unknownMsg); err == nil {
		return UnknownCursorMessage{
			Type:      unknownMsg.Type,
			SessionID: unknownMsg.SessionID,
		}
	}

	// Return default unknown message
	return UnknownCursorMessage{
		Type:      "unknown",
		SessionID: "",
	}
}

// ExtractCursorSessionID extracts session ID from cursor messages
func ExtractCursorSessionID(messages []CursorMessage) string {
	if len(messages) > 0 {
		return messages[0].GetSessionID()
	}
	return "unknown"
}

// ExtractCursorResult extracts the result text from cursor messages
func ExtractCursorResult(messages []CursorMessage) (string, error) {
	// Look for result message type (only type we care about)
	for i := len(messages) - 1; i >= 0; i-- {
		if resultMsg, ok := messages[i].(CursorResultMessage); ok {
			if resultMsg.Result != "" {
				return resultMsg.Result, nil
			}
		}
	}
	return "", fmt.Errorf("no result message found")
}
