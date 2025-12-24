package opencode

import (
	"strings"
	"testing"
)

func TestMapOpenCodeOutputToMessages(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedCount  int
		expectedTypes  []string
		expectError    bool
	}{
		{
			name: "successful parsing of complete response",
			input: `{"type":"step_start","timestamp":1759406013703,"sessionID":"ses_123","part":{}}
{"type":"text","timestamp":1759406015783,"sessionID":"ses_123","part":{"type":"text","text":"Hello!"}}
{"type":"step_finish","timestamp":1759406015885,"sessionID":"ses_123","part":{}}`,
			expectedCount: 3,
			expectedTypes: []string{"step_start", "text", "step_finish"},
			expectError:   false,
		},
		{
			name:          "empty input",
			input:         "",
			expectedCount: 0,
			expectedTypes: []string{},
			expectError:   false,
		},
		{
			name:          "single message",
			input:         `{"type":"text","timestamp":1759406015783,"sessionID":"ses_456","part":{"type":"text","text":"Single message"}}`,
			expectedCount: 1,
			expectedTypes: []string{"text"},
			expectError:   false,
		},
		{
			name: "handles empty lines",
			input: `{"type":"step_start","timestamp":1759406013703,"sessionID":"ses_789","part":{}}

{"type":"text","timestamp":1759406015783,"sessionID":"ses_789","part":{"type":"text","text":"Test"}}

`,
			expectedCount: 2,
			expectedTypes: []string{"step_start", "text"},
			expectError:   false,
		},
		{
			name:          "unknown message type",
			input:         `{"type":"unknown_type","timestamp":1759406013703,"sessionID":"ses_abc"}`,
			expectedCount: 1,
			expectedTypes: []string{"unknown_type"},
			expectError:   false,
		},
		{
			name:          "raw error output detected as raw_error type",
			input:         `not json at all`,
			expectedCount: 1,
			expectedTypes: []string{"raw_error"},
			expectError:   false,
		},
		{
			name: "javascript stack trace detected as raw_error",
			input: `154 |           stderr: "pipe",
155 |           stdout: "pipe",
156 |         })
RipgrepExtractionFailedError: RipgrepExtractionFailedError
 data: {
  filepath: "/home/ccagent/.local/share/opencode/bin/rg",
  stderr: "tar: unrecognized option: wildcards",
}`,
			expectedCount: 1,
			expectedTypes: []string{"raw_error"},
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages, err := MapOpenCodeOutputToMessages(tt.input)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if len(messages) != tt.expectedCount {
				t.Errorf("Expected %d messages, got %d", tt.expectedCount, len(messages))
			}

			for i, expectedType := range tt.expectedTypes {
				if i < len(messages) && messages[i].GetType() != expectedType {
					t.Errorf("Message %d: expected type %q, got %q", i, expectedType, messages[i].GetType())
				}
			}
		})
	}
}

func TestExtractOpenCodeSessionID(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		expectedID string
	}{
		{
			name: "extracts session ID from first message",
			input: `{"type":"step_start","timestamp":1759406013703,"sessionID":"ses_first123","part":{}}
{"type":"text","timestamp":1759406015783,"sessionID":"ses_first123","part":{"type":"text","text":"Hello"}}`,
			expectedID: "ses_first123",
		},
		{
			name:       "returns unknown for empty messages",
			input:      "",
			expectedID: "unknown",
		},
		{
			name:       "returns unknown when no session ID present",
			input:      `{"type":"text","timestamp":1759406015783}`,
			expectedID: "unknown",
		},
		{
			name:       "extracts from first message with session ID",
			input:      `{"type":"text","timestamp":1759406015783,"sessionID":"ses_fromtext","part":{"type":"text","text":"Test"}}`,
			expectedID: "ses_fromtext",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages, err := MapOpenCodeOutputToMessages(tt.input)
			if err != nil {
				t.Fatalf("Failed to parse messages: %v", err)
			}

			sessionID := ExtractOpenCodeSessionID(messages)
			if sessionID != tt.expectedID {
				t.Errorf("Expected session ID %q, got %q", tt.expectedID, sessionID)
			}
		})
	}
}

func TestExtractOpenCodeResult(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedResult string
		expectError    bool
		errorContains  string // substring that should be in the error message
	}{
		{
			name: "extracts text from text message",
			input: `{"type":"step_start","timestamp":1759406013703,"sessionID":"ses_123","part":{}}
{"type":"text","timestamp":1759406015783,"sessionID":"ses_123","part":{"type":"text","text":"Hello! How can I help you?"}}
{"type":"step_finish","timestamp":1759406015885,"sessionID":"ses_123","part":{}}`,
			expectedResult: "Hello! How can I help you?",
			expectError:    false,
		},
		{
			name: "concatenates multiple text messages",
			input: `{"type":"text","timestamp":1759406015783,"sessionID":"ses_123","part":{"type":"text","text":"First part. "}}
{"type":"text","timestamp":1759406015784,"sessionID":"ses_123","part":{"type":"text","text":"Second part."}}`,
			expectedResult: "First part. Second part.",
			expectError:    false,
		},
		{
			name:           "returns error when no text messages",
			input:          `{"type":"step_start","timestamp":1759406013703,"sessionID":"ses_123","part":{}}`,
			expectedResult: "",
			expectError:    true,
			errorContains:  "no text message found",
		},
		{
			name:           "returns error for empty input",
			input:          "",
			expectedResult: "",
			expectError:    true,
			errorContains:  "no text message found",
		},
		{
			name:           "ignores text message with empty text",
			input:          `{"type":"text","timestamp":1759406015783,"sessionID":"ses_123","part":{"type":"text","text":""}}`,
			expectedResult: "",
			expectError:    true,
			errorContains:  "no text message found",
		},
		{
			name: "returns opencode error for raw error output",
			input: `154 |           stderr: "pipe",
RipgrepExtractionFailedError: RipgrepExtractionFailedError
 data: {
  stderr: "tar: unrecognized option: wildcards",
}`,
			expectedResult: "",
			expectError:    true,
			errorContains:  "opencode error: RipgrepExtractionFailedError: RipgrepExtractionFailedError",
		},
		{
			name:           "returns opencode error with Error: line extracted",
			input:          `Some prefix\nError: Something went wrong\nStack trace here`,
			expectedResult: "",
			expectError:    true,
			errorContains:  "opencode error:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages, err := MapOpenCodeOutputToMessages(tt.input)
			if err != nil {
				t.Fatalf("Failed to parse messages: %v", err)
			}

			result, err := ExtractOpenCodeResult(messages)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if result != tt.expectedResult {
				t.Errorf("Expected result %q, got %q", tt.expectedResult, result)
			}
			// Check error message contains expected substring
			if tt.expectError && tt.errorContains != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
			}
		})
	}
}

func TestOpenCodeTextMessage_GetType(t *testing.T) {
	msg := OpenCodeTextMessage{
		Type:      "text",
		SessionID: "ses_123",
	}
	if msg.GetType() != "text" {
		t.Errorf("Expected type 'text', got %q", msg.GetType())
	}
}

func TestOpenCodeTextMessage_GetSessionID(t *testing.T) {
	msg := OpenCodeTextMessage{
		Type:      "text",
		SessionID: "ses_123",
	}
	if msg.GetSessionID() != "ses_123" {
		t.Errorf("Expected session ID 'ses_123', got %q", msg.GetSessionID())
	}
}

func TestOpenCodeStepMessage_GetType(t *testing.T) {
	msg := OpenCodeStepMessage{
		Type:      "step_start",
		SessionID: "ses_456",
	}
	if msg.GetType() != "step_start" {
		t.Errorf("Expected type 'step_start', got %q", msg.GetType())
	}
}

func TestOpenCodeStepMessage_GetSessionID(t *testing.T) {
	msg := OpenCodeStepMessage{
		Type:      "step_start",
		SessionID: "ses_456",
	}
	if msg.GetSessionID() != "ses_456" {
		t.Errorf("Expected session ID 'ses_456', got %q", msg.GetSessionID())
	}
}

func TestUnknownOpenCodeMessage_GetType(t *testing.T) {
	msg := UnknownOpenCodeMessage{
		Type:      "custom_type",
		SessionID: "ses_789",
	}
	if msg.GetType() != "custom_type" {
		t.Errorf("Expected type 'custom_type', got %q", msg.GetType())
	}
}

func TestUnknownOpenCodeMessage_GetSessionID(t *testing.T) {
	msg := UnknownOpenCodeMessage{
		Type:      "custom_type",
		SessionID: "ses_789",
	}
	if msg.GetSessionID() != "ses_789" {
		t.Errorf("Expected session ID 'ses_789', got %q", msg.GetSessionID())
	}
}
