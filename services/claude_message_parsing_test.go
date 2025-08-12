package services

import (
	"encoding/json"
	"os"
	"testing"
)

func TestMapClaudeOutputToMessages_WithComplexFixture(t *testing.T) {
	t.Skip("Skipping test that requires fixture file")
	// Load the fixture file
	fixtureData, err := os.ReadFile("fixtures/claude_response1.jsonl")
	if err != nil {
		t.Fatalf("Failed to read fixture file: %v", err)
	}

	// Parse the fixture using current logic
	messages, err := MapClaudeOutputToMessages(string(fixtureData))
	if err != nil {
		t.Fatalf("Failed to parse messages: %v", err)
	}

	t.Logf("Parsed %d messages from fixture", len(messages))

	// Let's examine what we got and what types were parsed correctly
	assistantCount := 0
	unknownCount := 0

	for i, msg := range messages {
		t.Logf("Message %d: Type=%s, SessionID=%s", i, msg.GetType(), msg.GetSessionID())

		switch msg.GetType() {
		case "assistant":
			assistantCount++
			// For assistant messages, let's see if we can extract content
			if assMsg, ok := msg.(AssistantMessage); ok {
				if len(assMsg.Message.Content) > 0 {
					// Try to parse the first content item to see what type it is
					var contentItem struct {
						Type string `json:"type"`
						Text string `json:"text,omitempty"`
						Name string `json:"name,omitempty"` // For tool_use
					}
					if err := json.Unmarshal(assMsg.Message.Content[0], &contentItem); err == nil {
						t.Logf("  Assistant content[0]: Type=%s", contentItem.Type)
						if contentItem.Type == "text" && contentItem.Text != "" {
							t.Logf("  Text: %s", contentItem.Text)
						} else if contentItem.Type == "tool_use" && contentItem.Name != "" {
							t.Logf("  Tool: %s", contentItem.Name)
						}
					}
				}
			}
		case "unknown":
			unknownCount++
		}
	}

	t.Logf("Summary: %d assistant messages, %d unknown messages", assistantCount, unknownCount)

	// The test should show that many messages are being parsed as "unknown"
	// when they should be parsed as specific types like system, user, etc.

	// Let's look for expected message types that should be parsed but aren't
	expectedMessageTypes := map[string]bool{
		"system":    false,
		"user":      false,
		"assistant": false,
	}

	for _, msg := range messages {
		msgType := msg.GetType()
		if _, exists := expectedMessageTypes[msgType]; exists {
			expectedMessageTypes[msgType] = true
		}
	}

	// Report what message types we found/didn't find
	for msgType, found := range expectedMessageTypes {
		if found {
			t.Logf("✓ Found %s messages", msgType)
		} else {
			t.Logf("✗ Missing %s messages (likely parsed as unknown)", msgType)
		}
	}

	// This test demonstrates the parsing limitations:
	// 1. System messages with subtype are not handled
	// 2. User messages are not handled
	// 3. Assistant messages with tool_use content may not be handled properly

	if unknownCount > assistantCount {
		t.Logf(
			"WARNING: More unknown (%d) than assistant (%d) messages - parsing logic may be incomplete",
			unknownCount,
			assistantCount,
		)
	}
}
