package codex

import (
	"strings"
	"testing"
)

func TestMapCodexOutputToMessages(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedCount int
		expectedTypes []string
		expectedError bool
	}{
		{
			name:          "single thread.started message",
			input:         `{"type":"thread.started","thread_id":"thread_abc123"}`,
			expectedCount: 1,
			expectedTypes: []string{"thread.started"},
			expectedError: false,
		},
		{
			name: "multiple item.completed messages",
			input: `{"type":"item.completed","item":{"id":"item_1","type":"reasoning","text":"Analyzing the request"}}
{"type":"item.completed","item":{"id":"item_2","type":"agent_message","text":"Here is my response"}}`,
			expectedCount: 2,
			expectedTypes: []string{"item.completed", "item.completed"},
			expectedError: false,
		},
		{
			name: "mixed message types",
			input: `{"type":"thread.started","thread_id":"thread_123"}
{"type":"item.started","item":{"id":"item_1","type":"reasoning"}}
{"type":"item.completed","item":{"id":"item_1","type":"reasoning","text":"Thinking..."}}
{"type":"item.completed","item":{"id":"item_2","type":"agent_message","text":"Response"}}
{"type":"turn.completed","usage":{"input_tokens":100,"cached_input_tokens":50,"output_tokens":75}}`,
			expectedCount: 5,
			expectedTypes: []string{"thread.started", "item.started", "item.completed", "item.completed", "turn.completed"},
			expectedError: false,
		},
		{
			name: "unknown message types fallback",
			input: `{"type":"custom","data":"some data"}
{"type":"unknown_event","value":"test"}`,
			expectedCount: 2,
			expectedTypes: []string{"custom", "unknown_event"},
			expectedError: false,
		},
		{
			name: "empty lines and whitespace",
			input: `{"type":"thread.started","thread_id":"thread_123"}

{"type":"item.completed","item":{"id":"item_1","type":"agent_message","text":"First"}}

{"type":"turn.completed","usage":{"input_tokens":100,"output_tokens":50}}`,
			expectedCount: 3,
			expectedTypes: []string{"thread.started", "item.completed", "turn.completed"},
			expectedError: false,
		},
		{
			name: "invalid JSON line creates unknown message",
			input: `{"type":"thread.started","thread_id":"thread_123"}
{invalid json here}
{"type":"turn.completed","usage":{"input_tokens":100,"output_tokens":50}}`,
			expectedCount: 3,
			expectedTypes: []string{"thread.started", "unknown", "turn.completed"},
			expectedError: false,
		},
		{
			name:          "empty input",
			input:         "",
			expectedCount: 0,
			expectedTypes: []string{},
			expectedError: false,
		},
		{
			name:          "only whitespace",
			input:         "   \n  \n  ",
			expectedCount: 0,
			expectedTypes: []string{},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages, err := MapCodexOutputToMessages(tt.input)

			if tt.expectedError && err == nil {
				t.Errorf("Expected error but got none")
				return
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(messages) != tt.expectedCount {
				t.Errorf("Expected %d messages, got %d", tt.expectedCount, len(messages))
				return
			}

			for i, expectedType := range tt.expectedTypes {
				if i >= len(messages) {
					t.Errorf(
						"Expected message %d with type %s, but only got %d messages",
						i,
						expectedType,
						len(messages),
					)
					continue
				}

				actualType := messages[i].GetType()
				if actualType != expectedType {
					t.Errorf("Message %d: expected type %s, got %s", i, expectedType, actualType)
				}
			}
		})
	}
}

func TestThreadStartedMessageParsing(t *testing.T) {
	input := `{"type":"thread.started","thread_id":"thread_abc123xyz"}`

	messages, err := MapCodexOutputToMessages(input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	threadMsg, ok := messages[0].(ThreadStartedMessage)
	if !ok {
		t.Fatalf("Expected ThreadStartedMessage, got %T", messages[0])
	}

	// Test field values
	if threadMsg.Type != "thread.started" {
		t.Errorf("Expected type 'thread.started', got '%s'", threadMsg.Type)
	}

	if threadMsg.ThreadID != "thread_abc123xyz" {
		t.Errorf("Expected thread_id 'thread_abc123xyz', got '%s'", threadMsg.ThreadID)
	}

	// Test interface methods
	if threadMsg.GetType() != "thread.started" {
		t.Errorf("GetType() expected 'thread.started', got '%s'", threadMsg.GetType())
	}

	if threadMsg.GetThreadID() != "thread_abc123xyz" {
		t.Errorf("GetThreadID() expected 'thread_abc123xyz', got '%s'", threadMsg.GetThreadID())
	}
}

func TestItemCompletedMessageParsing(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedType string
		expectedText string
	}{
		{
			name:         "agent_message type",
			input:        `{"type":"item.completed","item":{"id":"item_1","type":"agent_message","text":"Hello from Codex"}}`,
			expectedType: "agent_message",
			expectedText: "Hello from Codex",
		},
		{
			name:         "reasoning type",
			input:        `{"type":"item.completed","item":{"id":"item_2","type":"reasoning","text":"Analyzing the code structure"}}`,
			expectedType: "reasoning",
			expectedText: "Analyzing the code structure",
		},
		{
			name:         "command_execution type",
			input:        `{"type":"item.completed","item":{"id":"item_3","type":"command_execution","text":"Running tests","status":"success"}}`,
			expectedType: "command_execution",
			expectedText: "Running tests",
		},
		{
			name:         "file_change type",
			input:        `{"type":"item.completed","item":{"id":"item_4","type":"file_change","text":"Modified file.go"}}`,
			expectedType: "file_change",
			expectedText: "Modified file.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages, err := MapCodexOutputToMessages(tt.input)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(messages) != 1 {
				t.Fatalf("Expected 1 message, got %d", len(messages))
			}

			itemMsg, ok := messages[0].(ItemCompletedMessage)
			if !ok {
				t.Fatalf("Expected ItemCompletedMessage, got %T", messages[0])
			}

			if itemMsg.Type != "item.completed" {
				t.Errorf("Expected type 'item.completed', got '%s'", itemMsg.Type)
			}

			if itemMsg.Item.Type != tt.expectedType {
				t.Errorf("Expected item type '%s', got '%s'", tt.expectedType, itemMsg.Item.Type)
			}

			if itemMsg.Item.Text != tt.expectedText {
				t.Errorf("Expected item text '%s', got '%s'", tt.expectedText, itemMsg.Item.Text)
			}

			// Test interface methods
			if itemMsg.GetType() != "item.completed" {
				t.Errorf("GetType() expected 'item.completed', got '%s'", itemMsg.GetType())
			}

			if itemMsg.GetThreadID() != "" {
				t.Errorf("GetThreadID() expected empty string, got '%s'", itemMsg.GetThreadID())
			}
		})
	}
}

func TestTurnCompletedMessageParsing(t *testing.T) {
	input := `{"type":"turn.completed","usage":{"input_tokens":250,"cached_input_tokens":100,"output_tokens":150}}`

	messages, err := MapCodexOutputToMessages(input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	turnMsg, ok := messages[0].(TurnCompletedMessage)
	if !ok {
		t.Fatalf("Expected TurnCompletedMessage, got %T", messages[0])
	}

	if turnMsg.Type != "turn.completed" {
		t.Errorf("Expected type 'turn.completed', got '%s'", turnMsg.Type)
	}

	if turnMsg.Usage.InputTokens != 250 {
		t.Errorf("Expected input_tokens 250, got %d", turnMsg.Usage.InputTokens)
	}

	if turnMsg.Usage.CachedInputTokens != 100 {
		t.Errorf("Expected cached_input_tokens 100, got %d", turnMsg.Usage.CachedInputTokens)
	}

	if turnMsg.Usage.OutputTokens != 150 {
		t.Errorf("Expected output_tokens 150, got %d", turnMsg.Usage.OutputTokens)
	}

	// Test interface methods
	if turnMsg.GetType() != "turn.completed" {
		t.Errorf("GetType() expected 'turn.completed', got '%s'", turnMsg.GetType())
	}

	if turnMsg.GetThreadID() != "" {
		t.Errorf("GetThreadID() expected empty string, got '%s'", turnMsg.GetThreadID())
	}
}

func TestExtractCodexThreadID(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedThread string
	}{
		{
			name: "thread ID from thread.started message",
			input: `{"type":"thread.started","thread_id":"thread_xyz789"}
{"type":"item.completed","item":{"id":"item_1","type":"agent_message","text":"Response"}}`,
			expectedThread: "thread_xyz789",
		},
		{
			name: "no thread.started message",
			input: `{"type":"item.completed","item":{"id":"item_1","type":"agent_message","text":"Response"}}
{"type":"turn.completed","usage":{"input_tokens":100,"output_tokens":50}}`,
			expectedThread: "unknown",
		},
		{
			name:           "empty messages",
			input:          "",
			expectedThread: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages, err := MapCodexOutputToMessages(tt.input)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			threadID := ExtractCodexThreadID(messages)
			if threadID != tt.expectedThread {
				t.Errorf("Expected thread ID '%s', got '%s'", tt.expectedThread, threadID)
			}
		})
	}
}

func TestExtractCodexResult(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedResult string
		expectedError  bool
	}{
		{
			name: "last agent_message item",
			input: `{"type":"thread.started","thread_id":"thread_123"}
{"type":"item.completed","item":{"id":"item_1","type":"reasoning","text":"Thinking..."}}
{"type":"item.completed","item":{"id":"item_2","type":"agent_message","text":"This is the final response"}}
{"type":"turn.completed","usage":{"input_tokens":100,"output_tokens":50}}`,
			expectedResult: "This is the final response",
			expectedError:  false,
		},
		{
			name: "multiple agent_message items - takes last",
			input: `{"type":"item.completed","item":{"id":"item_1","type":"agent_message","text":"First message"}}
{"type":"item.completed","item":{"id":"item_2","type":"reasoning","text":"Thinking..."}}
{"type":"item.completed","item":{"id":"item_3","type":"agent_message","text":"Last message"}}`,
			expectedResult: "Last message",
			expectedError:  false,
		},
		{
			name: "no agent_message item",
			input: `{"type":"thread.started","thread_id":"thread_123"}
{"type":"item.completed","item":{"id":"item_1","type":"reasoning","text":"Thinking..."}}
{"type":"turn.completed","usage":{"input_tokens":100,"output_tokens":50}}`,
			expectedResult: "",
			expectedError:  true,
		},
		{
			name: "agent_message with empty text",
			input: `{"type":"item.completed","item":{"id":"item_1","type":"agent_message","text":""}}
{"type":"item.completed","item":{"id":"item_2","type":"reasoning","text":"Thinking..."}}`,
			expectedResult: "",
			expectedError:  true,
		},
		{
			name:           "empty messages",
			input:          "",
			expectedResult: "",
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages, err := MapCodexOutputToMessages(tt.input)
			if err != nil {
				t.Fatalf("Unexpected error parsing messages: %v", err)
			}

			result, err := ExtractCodexResult(messages)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expectedResult {
				t.Errorf("Expected result '%s', got '%s'", tt.expectedResult, result)
			}
		})
	}
}

func TestLargeLineHandling(t *testing.T) {
	// Test that very long JSON lines can be handled (Codex can produce lines over 1MB)
	// Create a large agent_message text (simulate a response with lots of code)
	largeText := strings.Repeat("This is a long response with lots of text. ", 100000) // ~4.4MB

	input := `{"type":"thread.started","thread_id":"thread_large"}
{"type":"item.completed","item":{"id":"item_1","type":"agent_message","text":"` + largeText + `"}}`

	messages, err := MapCodexOutputToMessages(input)
	if err != nil {
		t.Fatalf("Failed to parse large JSON line: %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(messages))
	}

	// Extract result to verify large text was preserved
	result, err := ExtractCodexResult(messages)
	if err != nil {
		t.Fatalf("Failed to extract result: %v", err)
	}

	if result != largeText {
		t.Errorf("Large text was not preserved correctly (got %d bytes, expected %d bytes)", len(result), len(largeText))
	}
}

func TestRealWorldCodexExample(t *testing.T) {
	// Simulate a realistic Codex session output
	input := `{"type":"thread.started","thread_id":"thread_real_abc123"}
{"type":"item.started","item":{"id":"item_1","type":"reasoning"}}
{"type":"item.completed","item":{"id":"item_1","type":"reasoning","text":"I need to analyze the file structure first"}}
{"type":"item.started","item":{"id":"item_2","type":"command_execution"}}
{"type":"item.completed","item":{"id":"item_2","type":"command_execution","text":"ls -la","status":"success"}}
{"type":"item.started","item":{"id":"item_3","type":"agent_message"}}
{"type":"item.completed","item":{"id":"item_3","type":"agent_message","text":"I've analyzed the directory structure. Here's what I found..."}}
{"type":"turn.completed","usage":{"input_tokens":342,"cached_input_tokens":0,"output_tokens":156}}`

	messages, err := MapCodexOutputToMessages(input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(messages) != 8 {
		t.Errorf("Expected 8 messages, got %d", len(messages))
	}

	// Verify thread ID extraction
	threadID := ExtractCodexThreadID(messages)
	expectedThreadID := "thread_real_abc123"
	if threadID != expectedThreadID {
		t.Errorf("Expected thread ID '%s', got '%s'", expectedThreadID, threadID)
	}

	// Verify result extraction
	result, err := ExtractCodexResult(messages)
	if err != nil {
		t.Fatalf("Failed to extract result: %v", err)
	}

	expectedResult := "I've analyzed the directory structure. Here's what I found..."
	if result != expectedResult {
		t.Errorf("Expected result '%s', got '%s'", expectedResult, result)
	}
}
