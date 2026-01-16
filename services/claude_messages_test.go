package services

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestMapClaudeOutputToMessages(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedCount int
		expectedTypes []string
		expectedError bool
	}{
		{
			name:          "single assistant message",
			input:         `{"type":"assistant","message":{"id":"msg_01PW48ecPbBMYDbdvy8eeX6y","type":"message","content":[{"type":"text","text":"Hello! I'm Claude Code"}]},"session_id":"c069b138-4f6c-406b-b79a-1e940179378d"}`,
			expectedCount: 1,
			expectedTypes: []string{"assistant"},
			expectedError: false,
		},
		{
			name: "multiple assistant messages",
			input: `{"type":"assistant","message":{"id":"msg_01","type":"message","content":[{"type":"text","text":"First message"}]},"session_id":"session1"}
{"type":"assistant","message":{"id":"msg_02","type":"message","content":[{"type":"text","text":"Second message"}]},"session_id":"session1"}`,
			expectedCount: 2,
			expectedTypes: []string{"assistant", "assistant"},
			expectedError: false,
		},
		{
			name: "mixed message types",
			input: `{"type":"system","subtype":"init","session_id":"session1"}
{"type":"assistant","message":{"id":"msg_01","type":"message","content":[{"type":"text","text":"Assistant response"}]},"session_id":"session1"}
{"type":"user","message":{"role":"user","content":[{"type":"text","text":"User message"}]},"session_id":"session1"}`,
			expectedCount: 3,
			expectedTypes: []string{"system", "assistant", "user"},
			expectedError: false,
		},
		{
			name: "unknown message types fallback",
			input: `{"type":"custom","data":"some data","session_id":"session1"}
{"type":"result","subtype":"error","session_id":"session1"}`,
			expectedCount: 2,
			expectedTypes: []string{"custom", "result"},
			expectedError: false,
		},
		{
			name: "empty lines and whitespace",
			input: `{"type":"assistant","message":{"id":"msg_01","type":"message","content":[{"type":"text","text":"First"}]},"session_id":"session1"}

{"type":"system","session_id":"session1"}
   
{"type":"assistant","message":{"id":"msg_02","type":"message","content":[{"type":"text","text":"Second"}]},"session_id":"session1"}`,
			expectedCount: 3,
			expectedTypes: []string{"assistant", "system", "assistant"},
			expectedError: false,
		},
		{
			name: "invalid JSON line creates unknown message",
			input: `{"type":"assistant","message":{"id":"msg_01","type":"message","content":[{"type":"text","text":"Valid"}]},"session_id":"session1"}
{invalid json here}
{"type":"system","session_id":"session1"}`,
			expectedCount: 3,
			expectedTypes: []string{"assistant", "unknown", "system"},
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
			messages, err := MapClaudeOutputToMessages(tt.input)

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

func TestAssistantMessageParsing(t *testing.T) {
	input := `{"type":"assistant","message":{"id":"msg_01PW48ecPbBMYDbdvy8eeX6y","type":"message","content":[{"type":"text","text":"Hello! I'm Claude Code, ready to help you."}]},"session_id":"c069b138-4f6c-406b-b79a-1e940179378d"}`

	messages, err := MapClaudeOutputToMessages(input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	assistantMsg, ok := messages[0].(AssistantMessage)
	if !ok {
		t.Fatalf("Expected AssistantMessage, got %T", messages[0])
	}

	// Test field values
	if assistantMsg.Type != "assistant" {
		t.Errorf("Expected type 'assistant', got '%s'", assistantMsg.Type)
	}

	if assistantMsg.SessionID != "c069b138-4f6c-406b-b79a-1e940179378d" {
		t.Errorf("Expected session_id 'c069b138-4f6c-406b-b79a-1e940179378d', got '%s'", assistantMsg.SessionID)
	}

	if assistantMsg.Message.ID != "msg_01PW48ecPbBMYDbdvy8eeX6y" {
		t.Errorf("Expected message ID 'msg_01PW48ecPbBMYDbdvy8eeX6y', got '%s'", assistantMsg.Message.ID)
	}

	if len(assistantMsg.Message.Content) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(assistantMsg.Message.Content))
	}

	// Parse the content to check if it's a text content item
	var contentItem struct {
		Type string `json:"type"`
		Text string `json:"text,omitempty"`
	}
	if err := json.Unmarshal(assistantMsg.Message.Content[0], &contentItem); err != nil {
		t.Fatalf("Failed to parse content: %v", err)
	}

	if contentItem.Type != "text" {
		t.Errorf("Expected content type 'text', got '%s'", contentItem.Type)
	}

	expectedText := "Hello! I'm Claude Code, ready to help you."
	if contentItem.Text != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, contentItem.Text)
	}

	// Test interface methods
	if assistantMsg.GetType() != "assistant" {
		t.Errorf("GetType() expected 'assistant', got '%s'", assistantMsg.GetType())
	}

	if assistantMsg.GetSessionID() != "c069b138-4f6c-406b-b79a-1e940179378d" {
		t.Errorf(
			"GetSessionID() expected 'c069b138-4f6c-406b-b79a-1e940179378d', got '%s'",
			assistantMsg.GetSessionID(),
		)
	}
}

func TestSystemMessageParsing(t *testing.T) {
	input := `{"type":"system","subtype":"init","cwd":"/path","session_id":"79fac4e0-79bd-4489-afb5-6023fa22cc47","tools":["Task","Bash"]}`

	messages, err := MapClaudeOutputToMessages(input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	systemMsg, ok := messages[0].(SystemMessage)
	if !ok {
		t.Fatalf("Expected SystemMessage, got %T", messages[0])
	}

	if systemMsg.Type != "system" {
		t.Errorf("Expected type 'system', got '%s'", systemMsg.Type)
	}

	if systemMsg.Subtype != "init" {
		t.Errorf("Expected subtype 'init', got '%s'", systemMsg.Subtype)
	}

	if systemMsg.SessionID != "79fac4e0-79bd-4489-afb5-6023fa22cc47" {
		t.Errorf("Expected session_id '79fac4e0-79bd-4489-afb5-6023fa22cc47', got '%s'", systemMsg.SessionID)
	}

	// Test interface methods
	if systemMsg.GetType() != "system" {
		t.Errorf("GetType() expected 'system', got '%s'", systemMsg.GetType())
	}

	if systemMsg.GetSessionID() != "79fac4e0-79bd-4489-afb5-6023fa22cc47" {
		t.Errorf("GetSessionID() expected '79fac4e0-79bd-4489-afb5-6023fa22cc47', got '%s'", systemMsg.GetSessionID())
	}
}

func TestExtractLastAssistantMessage(t *testing.T) {
	input := `{"type":"system","subtype":"init","session_id":"session1"}
{"type":"assistant","message":{"id":"msg_01","type":"message","content":[{"type":"text","text":"First assistant message"}]},"session_id":"session1"}
{"type":"user","message":{"role":"user","content":[{"type":"text","text":"User message"}]},"session_id":"session1"}
{"type":"assistant","message":{"id":"msg_02","type":"message","content":[{"type":"text","text":"Last assistant message"}]},"session_id":"session1"}
{"type":"result","subtype":"complete","session_id":"session1"}`

	messages, err := MapClaudeOutputToMessages(input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Simulate extractClaudeResult logic
	var lastAssistantText string
	for i := len(messages) - 1; i >= 0; i-- {
		if assistantMsg, ok := messages[i].(AssistantMessage); ok {
			if len(assistantMsg.Message.Content) > 0 {
				// Parse the content to check if it's a text content item
				var contentItem struct {
					Type string `json:"type"`
					Text string `json:"text,omitempty"`
				}
				if err := json.Unmarshal(assistantMsg.Message.Content[0], &contentItem); err == nil {
					if contentItem.Type == "text" && contentItem.Text != "" {
						lastAssistantText = contentItem.Text
						break
					}
				}
			}
		}
	}

	expectedText := "Last assistant message"
	if lastAssistantText != expectedText {
		t.Errorf("Expected last assistant text '%s', got '%s'", expectedText, lastAssistantText)
	}
}

func TestExitPlanModeMessageParsing(t *testing.T) {
	input := `{"type":"assistant","message":{"id":"msg_0139SNMjfcWzXrNfYBpWk95m","type":"message","role":"assistant","model":"claude-sonnet-4-20250514","content":[{"type":"tool_use","id":"toolu_01LSsuZqZKgXvatJKCL59rb1","name":"ExitPlanMode","input":{"plan":"# Test Plan\n\n## Overview\nThis is a test plan for ExitPlanMode parsing."}}]},"session_id":"82dc5b6b-5683-4862-b95e-837abf08df0d"}`

	messages, err := MapClaudeOutputToMessages(input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	exitPlanMsg, ok := messages[0].(ExitPlanModeMessage)
	if !ok {
		t.Fatalf("Expected ExitPlanModeMessage, got %T", messages[0])
	}

	// Test field values
	if exitPlanMsg.Type != "assistant" {
		t.Errorf("Expected type 'assistant', got '%s'", exitPlanMsg.Type)
	}

	if exitPlanMsg.SessionID != "82dc5b6b-5683-4862-b95e-837abf08df0d" {
		t.Errorf("Expected session_id '82dc5b6b-5683-4862-b95e-837abf08df0d', got '%s'", exitPlanMsg.SessionID)
	}

	if exitPlanMsg.Message.ID != "msg_0139SNMjfcWzXrNfYBpWk95m" {
		t.Errorf("Expected message ID 'msg_0139SNMjfcWzXrNfYBpWk95m', got '%s'", exitPlanMsg.Message.ID)
	}

	if exitPlanMsg.Message.Model != "claude-sonnet-4-20250514" {
		t.Errorf("Expected model 'claude-sonnet-4-20250514', got '%s'", exitPlanMsg.Message.Model)
	}

	if len(exitPlanMsg.Message.Content) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(exitPlanMsg.Message.Content))
	}

	content := exitPlanMsg.Message.Content[0]
	if content.Type != "tool_use" {
		t.Errorf("Expected content type 'tool_use', got '%s'", content.Type)
	}

	if content.Name != "ExitPlanMode" {
		t.Errorf("Expected tool name 'ExitPlanMode', got '%s'", content.Name)
	}

	expectedPlan := "# Test Plan\n\n## Overview\nThis is a test plan for ExitPlanMode parsing."
	if content.Input.Plan != expectedPlan {
		t.Errorf("Expected plan '%s', got '%s'", expectedPlan, content.Input.Plan)
	}

	// Test interface methods
	if exitPlanMsg.GetType() != "exit_plan_mode" {
		t.Errorf("GetType() expected 'exit_plan_mode', got '%s'", exitPlanMsg.GetType())
	}

	if exitPlanMsg.GetSessionID() != "82dc5b6b-5683-4862-b95e-837abf08df0d" {
		t.Errorf(
			"GetSessionID() expected '82dc5b6b-5683-4862-b95e-837abf08df0d', got '%s'",
			exitPlanMsg.GetSessionID(),
		)
	}

	// Test GetPlan() method
	if exitPlanMsg.GetPlan() != expectedPlan {
		t.Errorf("GetPlan() expected '%s', got '%s'", expectedPlan, exitPlanMsg.GetPlan())
	}
}

func TestRealWorldExample(t *testing.T) {
	// Based on the actual output-finish-todo.jsonl structure
	input := `{"type":"system","subtype":"init","cwd":"/Users/pmihaylov/prg/ccpg/cc1","session_id":"79fac4e0-79bd-4489-afb5-6023fa22cc47","tools":["Task","Bash","Glob","Grep","LS","ExitPlanMode","Read","Edit","MultiEdit","Write","NotebookRead","NotebookEdit","WebFetch","TodoWrite","WebSearch"],"mcp_servers":[],"model":"claude-sonnet-4-20250514","permissionMode":"acceptEdits","apiKeySource":"ANTHROPIC_API_KEY"}
{"type":"assistant","message":{"id":"msg_01HCL8z1N6MtR4Z4P9puyAua","type":"message","role":"assistant","model":"claude-sonnet-4-20250514","content":[{"type":"text","text":"I'll study the ccagent codebase to understand its logging architecture and propose options for implementing persistent logging."}],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":4,"cache_creation_input_tokens":16747,"cache_read_input_tokens":0,"output_tokens":3,"service_tier":"standard"}},"parent_tool_use_id":null,"session_id":"79fac4e0-79bd-4489-afb5-6023fa22cc47"}
{"type":"result","subtype":"error_during_execution","duration_ms":70219,"duration_api_ms":69749,"is_error":false,"num_turns":0,"session_id":"79fac4e0-79bd-4489-afb5-6023fa22cc47","total_cost_usd":0.21045915,"usage":{"input_tokens":337,"cache_creation_input_tokens":33704,"cache_read_input_tokens":286445,"output_tokens":4075,"server_tool_use":{"web_search_requests":0},"service_tier":"standard"}}`

	messages, err := MapClaudeOutputToMessages(input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}

	// Check first message (system)
	if messages[0].GetType() != "system" {
		t.Errorf("Expected first message type 'system', got '%s'", messages[0].GetType())
	}

	// Check second message (assistant)
	if messages[1].GetType() != "assistant" {
		t.Errorf("Expected second message type 'assistant', got '%s'", messages[1].GetType())
	}

	assistantMsg, ok := messages[1].(AssistantMessage)
	if !ok {
		t.Fatalf("Expected AssistantMessage, got %T", messages[1])
	}

	expectedText := "I'll study the ccagent codebase to understand its logging architecture and propose options for implementing persistent logging."
	if len(assistantMsg.Message.Content) == 0 {
		t.Errorf("Expected content in assistant message")
	} else {
		// Parse the content to check if it's a text content item
		var contentItem struct {
			Type string `json:"type"`
			Text string `json:"text,omitempty"`
		}
		if err := json.Unmarshal(assistantMsg.Message.Content[0], &contentItem); err != nil {
			t.Errorf("Failed to parse assistant content: %v", err)
		} else if contentItem.Text != expectedText {
			t.Errorf("Expected text '%s', got '%s'", expectedText, contentItem.Text)
		}
	}

	// Check third message (result)
	if messages[2].GetType() != "result" {
		t.Errorf("Expected third message type 'result', got '%s'", messages[2].GetType())
	}

	// All messages should have the same session ID
	expectedSessionID := "79fac4e0-79bd-4489-afb5-6023fa22cc47"
	for i, msg := range messages {
		if msg.GetSessionID() != expectedSessionID {
			t.Errorf("Message %d: expected session_id '%s', got '%s'", i, expectedSessionID, msg.GetSessionID())
		}
	}
}

func TestProductionLogFileParsing(t *testing.T) {
	// Read the production log file that failed to parse
	logFilePath := "fixtures/claude-session-with-image.log"

	content, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Skipf("Production log file not found at %s: %v", logFilePath, err)
		return
	}

	// Try to parse the log file
	messages, err := MapClaudeOutputToMessages(string(content))
	if err != nil {
		t.Fatalf("Failed to parse production log file: %v", err)
	}

	t.Logf("Successfully parsed %d messages from production log", len(messages))

	// Report on message types found
	typeCounts := make(map[string]int)
	for _, msg := range messages {
		typeCounts[msg.GetType()]++
	}

	t.Logf("Message type breakdown:")
	for msgType, count := range typeCounts {
		t.Logf("  %s: %d", msgType, count)
	}

	// Check that we have at least one of each expected type
	expectedTypes := []string{"system", "assistant", "user"}
	for _, expectedType := range expectedTypes {
		if typeCounts[expectedType] == 0 {
			t.Errorf("Expected to find at least one '%s' message, but found none", expectedType)
		}
	}
}

func TestStripBase64Images(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "strips large base64 image data",
			input:    `{"type":"image","source":{"type":"base64","data":"` + strings.Repeat("iVBORw0KGgo", 100) + `=="}}`,
			expected: `{"type":"image","source":{"type":"base64","data":"[IMAGE_STRIPPED]"}}`,
		},
		{
			name:     "preserves short data fields",
			input:    `{"type":"data","data":"short"}`,
			expected: `{"type":"data","data":"short"}`,
		},
		{
			name: "strips multiple images in one line",
			input: `{"content":[{"type":"image","source":{"data":"` + strings.Repeat("A", 1500) + `"}},{"type":"image","source":{"data":"` + strings.Repeat("B", 2000) + `"}}]}`,
			expected: `{"content":[{"type":"image","source":{"data":"[IMAGE_STRIPPED]"}},{"type":"image","source":{"data":"[IMAGE_STRIPPED]"}}]}`,
		},
		{
			name:     "handles JSON with no images",
			input:    `{"type":"text","content":"Hello world"}`,
			expected: `{"type":"text","content":"Hello world"}`,
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripBase64Images(tt.input)
			if result != tt.expected {
				t.Errorf("Expected:\n%s\n\nGot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestProductionLogParsingWithImageStripping(t *testing.T) {
	// This test verifies that the production log with large images can be parsed
	// after image stripping, without needing a huge buffer
	logFilePath := "fixtures/claude-session-with-image.log"

	content, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Skipf("Production log file not found at %s: %v", logFilePath, err)
		return
	}

	// First verify that the log contains large base64 data
	contentStr := string(content)
	if !strings.Contains(contentStr, "iVBORw0KGgo") {
		t.Skip("Production log doesn't contain expected image data")
	}

	// Strip images and verify the output is much smaller
	stripped := stripBase64Images(contentStr)
	reductionPercent := 100 * (1.0 - float64(len(stripped))/float64(len(content)))
	t.Logf("Image stripping reduced size by %.1f%% (%d -> %d bytes)",
		reductionPercent, len(content), len(stripped))

	if len(stripped) >= len(content) {
		t.Error("Expected stripped content to be smaller than original")
	}

	// Verify stripped content can still be parsed successfully
	messages, err := MapClaudeOutputToMessages(contentStr)
	if err != nil {
		t.Fatalf("Failed to parse log after image stripping: %v", err)
	}

	if len(messages) == 0 {
		t.Error("Expected to parse some messages")
	}

	t.Logf("Successfully parsed %d messages after image stripping", len(messages))
}

func TestStripLargeToolResultContent(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectStripped bool
		description    string
	}{
		{
			name:           "small tool_result content unchanged",
			input:          `{"type":"user","message":{"role":"user","content":[{"tool_use_id":"toolu_001","type":"tool_result","content":"small output"}]}}`,
			expectStripped: false,
			description:    "Content under 100KB should not be modified",
		},
		{
			name:           "large tool_result content truncated",
			input:          `{"type":"user","message":{"role":"user","content":[{"tool_use_id":"toolu_001","type":"tool_result","content":"` + strings.Repeat("x", 150*1024) + `"}]}}`,
			expectStripped: true,
			description:    "Content over 100KB should be truncated",
		},
		{
			name:           "non-tool_result content unchanged",
			input:          `{"type":"assistant","message":{"content":[{"type":"text","text":"` + strings.Repeat("y", 150*1024) + `"}]}}`,
			expectStripped: false,
			description:    "Non-tool_result content should not be affected",
		},
		{
			name:           "empty input",
			input:          "",
			expectStripped: false,
			description:    "Empty input should return empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripLargeToolResultContent(tt.input)

			if tt.expectStripped {
				if !strings.Contains(result, "[CONTENT_TRUNCATED_") {
					t.Errorf("%s: expected content to be truncated but it wasn't", tt.description)
				}
				if len(result) >= len(tt.input) {
					t.Errorf("%s: expected result to be smaller than input", tt.description)
				}
			} else {
				if tt.input != "" && result != tt.input {
					t.Errorf("%s: expected content to remain unchanged", tt.description)
				}
			}
		})
	}
}

func TestLargeToolResultParsing(t *testing.T) {
	// This test verifies that the parser can handle output containing very large tool_result
	// content that would previously exceed the 4MB buffer limit.
	//
	// Real-world scenario: Claude reads a large file or runs grep that returns 17-85MB of output.
	// The tool_result JSON line would exceed bufio.Scanner's buffer and cause parsing to fail.
	// With stripLargeToolResultContent, the content is truncated before parsing.

	// Create a simulated large output with a tool_result containing >4MB of content
	// Use escaped newlines (\n) since this is JSON content within a single line
	largeContent := strings.Repeat("Line of grep output matching pattern in file.go:123\\n", 100000)

	input := `{"type":"system","subtype":"init","session_id":"test-session-001"}
{"type":"assistant","message":{"id":"msg_001","type":"message","content":[{"type":"text","text":"Let me search the codebase."}]},"session_id":"test-session-001"}
{"type":"assistant","message":{"id":"msg_001","type":"message","content":[{"type":"tool_use","id":"toolu_001","name":"Grep","input":{"pattern":"error"}}]},"session_id":"test-session-001"}
{"type":"user","message":{"role":"user","content":[{"tool_use_id":"toolu_001","type":"tool_result","content":"` + largeContent + `"}]},"session_id":"test-session-001"}
{"type":"assistant","message":{"id":"msg_002","type":"message","content":[{"type":"text","text":"Found many matches. Here's the analysis."}]},"session_id":"test-session-001"}
{"type":"result","subtype":"success","is_error":false,"result":"Found many matches. Here's the analysis.","session_id":"test-session-001"}`

	t.Logf("Test input size: %d bytes (%.2f MB)", len(input), float64(len(input))/(1024*1024))

	messages, err := MapClaudeOutputToMessages(input)
	if err != nil {
		t.Fatalf("Failed to parse output with large tool_result: %v", err)
	}

	// Should successfully parse all messages
	if len(messages) != 6 {
		t.Errorf("Expected 6 messages, got %d", len(messages))
	}

	// Verify we can extract the final result
	var resultMsg ResultMessage
	var foundResult bool
	for _, msg := range messages {
		if r, ok := msg.(ResultMessage); ok {
			resultMsg = r
			foundResult = true
		}
	}

	if !foundResult {
		t.Error("Expected to find a result message")
	}

	if resultMsg.Result != "Found many matches. Here's the analysis." {
		t.Errorf("Unexpected result: %s", resultMsg.Result)
	}

	t.Logf("Successfully parsed %d messages from large output", len(messages))
}

func TestVeryLargeToolResultContent(t *testing.T) {
	// Test with content size similar to the real failure case (>80MB becomes >4MB after truncation)
	// We simulate this by creating content that would be ~5MB before truncation

	// Create 5MB of content (simulating a real tool result)
	largeContent := strings.Repeat("a", 5*1024*1024)

	input := `{"type":"system","subtype":"init","session_id":"test-001"}
{"type":"user","message":{"role":"user","content":[{"tool_use_id":"toolu_001","type":"tool_result","content":"` + largeContent + `"}]},"session_id":"test-001"}
{"type":"result","subtype":"success","result":"Done","session_id":"test-001"}`

	t.Logf("Original input size: %.2f MB", float64(len(input))/(1024*1024))

	messages, err := MapClaudeOutputToMessages(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}

	// Verify result message is extractable
	if resultMsg, ok := messages[2].(ResultMessage); ok {
		if resultMsg.Result != "Done" {
			t.Errorf("Expected result 'Done', got '%s'", resultMsg.Result)
		}
	} else {
		t.Errorf("Expected ResultMessage, got %T", messages[2])
	}
}

func TestErrorDuringExecutionWithValidResponse(t *testing.T) {
	// This test verifies that when Claude exits with status 1 due to an internal error
	// (like EISDIR), but has already produced a valid response, we can still extract
	// the assistant's response from the output.
	//
	// This simulates the real-world scenario where:
	// 1. Claude starts a session and produces valid responses
	// 2. An internal error occurs (e.g., trying to read a directory as a file)
	// 3. Claude exits with status 1
	// 4. The output contains both valid assistant messages AND an error result
	logFilePath := "fixtures/error_during_execution_with_response.json"

	content, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Skipf("Fixture file not found at %s: %v", logFilePath, err)
		return
	}

	messages, err := MapClaudeOutputToMessages(string(content))
	if err != nil {
		t.Fatalf("Failed to parse fixture: %v", err)
	}

	// Should parse all messages successfully
	if len(messages) == 0 {
		t.Fatal("Expected to parse some messages")
	}

	t.Logf("Parsed %d messages from fixture", len(messages))

	// Find the result message
	var resultMsg ResultMessage
	var foundResult bool
	for _, msg := range messages {
		if r, ok := msg.(ResultMessage); ok {
			resultMsg = r
			foundResult = true
			break
		}
	}

	if !foundResult {
		t.Fatal("Expected to find a result message")
	}

	// Verify the result message has the error_during_execution subtype
	if resultMsg.Subtype != "error_during_execution" {
		t.Errorf("Expected subtype 'error_during_execution', got '%s'", resultMsg.Subtype)
	}

	// Verify is_error is true
	if !resultMsg.IsError {
		t.Error("Expected is_error to be true")
	}

	// Find assistant messages with text content (not just tool_use)
	var assistantTextMessages []string
	for _, msg := range messages {
		if aMsg, ok := msg.(AssistantMessage); ok {
			for _, contentRaw := range aMsg.Message.Content {
				var contentItem struct {
					Type string `json:"type"`
					Text string `json:"text,omitempty"`
				}
				if err := json.Unmarshal(contentRaw, &contentItem); err == nil {
					if contentItem.Type == "text" && contentItem.Text != "" {
						assistantTextMessages = append(assistantTextMessages, contentItem.Text)
					}
				}
			}
		}
	}

	if len(assistantTextMessages) == 0 {
		t.Error("Expected to find assistant messages with text content")
	} else {
		t.Logf("Found %d assistant text messages", len(assistantTextMessages))
		// The last assistant text should contain the actual response
		lastAssistantText := assistantTextMessages[len(assistantTextMessages)-1]
		if !strings.Contains(lastAssistantText, "validation") {
			t.Errorf("Expected assistant text to contain 'validation', got: %s", lastAssistantText)
		}
	}
}

func TestStripLargeToolUseResultContent(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectStripped bool
		truncMarker    string
		description    string
	}{
		{
			name:           "small stdout unchanged",
			input:          `{"type":"user","tool_use_result":{"stdout":"small output","stderr":"","interrupted":false}}`,
			expectStripped: false,
			description:    "stdout under 100KB should not be modified",
		},
		{
			name:           "large stdout truncated",
			input:          `{"type":"user","tool_use_result":{"stdout":"` + strings.Repeat("x", 150*1024) + `","stderr":"","interrupted":false}}`,
			expectStripped: true,
			truncMarker:    "[STDOUT_TRUNCATED_",
			description:    "stdout over 100KB should be truncated",
		},
		{
			name:           "large stderr truncated",
			input:          `{"type":"user","tool_use_result":{"stdout":"","stderr":"` + strings.Repeat("e", 150*1024) + `","interrupted":false}}`,
			expectStripped: true,
			truncMarker:    "[STDERR_TRUNCATED_",
			description:    "stderr over 100KB should be truncated",
		},
		{
			name:           "both stdout and stderr truncated",
			input:          `{"type":"user","tool_use_result":{"stdout":"` + strings.Repeat("o", 150*1024) + `","stderr":"` + strings.Repeat("e", 150*1024) + `","interrupted":false}}`,
			expectStripped: true,
			truncMarker:    "[STDOUT_TRUNCATED_",
			description:    "Both stdout and stderr over 100KB should be truncated",
		},
		{
			name:           "empty input",
			input:          "",
			expectStripped: false,
			description:    "Empty input should return empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripLargeToolUseResultContent(tt.input)

			if tt.expectStripped {
				if !strings.Contains(result, tt.truncMarker) {
					t.Errorf("%s: expected content to be truncated with marker %s but it wasn't", tt.description, tt.truncMarker)
				}
				if len(result) >= len(tt.input) {
					t.Errorf("%s: expected result to be smaller than input", tt.description)
				}
			} else {
				if tt.input != "" && result != tt.input {
					t.Errorf("%s: expected content to remain unchanged", tt.description)
				}
			}
		})
	}
}

func TestToolResultWithEscapedQuotes(t *testing.T) {
	// This test verifies that tool_result content with escaped quotes is properly handled.
	// The bug was that the regex [^"]* would consume backslashes, causing it to stop early
	// when it hit an escaped quote (\") because it would consume the backslash and then
	// stop at the quote.
	//
	// Fixed by using [^"\\]* which doesn't consume backslashes, allowing \\. to properly
	// match escaped characters like \"

	// Content with escaped JSON containing quotes
	contentWithEscapedQuotes := `Some text with \"escaped\" quotes and more \"nested\" content`
	input := `{"type":"user","message":{"role":"user","content":[{"tool_use_id":"toolu_001","type":"tool_result","content":"` + contentWithEscapedQuotes + `"}]}}`

	// The strip function should handle this correctly without breaking
	result := stripLargeToolResultContent(input)

	// Since content is small, it should remain unchanged
	if result != input {
		t.Errorf("Small content with escaped quotes should remain unchanged.\nInput:  %s\nResult: %s", input, result)
	}
}

func TestLargeToolUseResultParsing(t *testing.T) {
	// This test verifies that the parser can handle output containing very large
	// tool_use_result.stdout fields that would previously exceed the 4MB buffer limit.
	//
	// Real-world scenario: Claude runs a command that outputs 64MB+ of data.
	// The tool_use_result.stdout field would exceed bufio.Scanner's buffer.
	// With stripLargeToolUseResultContent, the stdout is truncated before parsing.

	// Create 5MB of stdout content
	largeStdout := strings.Repeat("output line\\n", 500000)

	input := `{"type":"system","subtype":"init","session_id":"test-session-001"}
{"type":"assistant","message":{"id":"msg_001","type":"message","content":[{"type":"tool_use","id":"toolu_001","name":"Bash","input":{"command":"cat bigfile.txt"}}]},"session_id":"test-session-001"}
{"type":"user","message":{"role":"user","content":[{"tool_use_id":"toolu_001","type":"tool_result","content":"output"}]},"session_id":"test-session-001","tool_use_result":{"stdout":"` + largeStdout + `","stderr":"","interrupted":false}}
{"type":"assistant","message":{"id":"msg_002","type":"message","content":[{"type":"text","text":"Done processing."}]},"session_id":"test-session-001"}
{"type":"result","subtype":"success","is_error":false,"result":"Done processing.","session_id":"test-session-001"}`

	t.Logf("Test input size: %d bytes (%.2f MB)", len(input), float64(len(input))/(1024*1024))

	messages, err := MapClaudeOutputToMessages(input)
	if err != nil {
		t.Fatalf("Failed to parse output with large tool_use_result: %v", err)
	}

	// Should successfully parse all messages
	if len(messages) != 5 {
		t.Errorf("Expected 5 messages, got %d", len(messages))
	}

	// Verify we can extract the final result
	var resultMsg ResultMessage
	var foundResult bool
	for _, msg := range messages {
		if r, ok := msg.(ResultMessage); ok {
			resultMsg = r
			foundResult = true
		}
	}

	if !foundResult {
		t.Error("Expected to find a result message")
	}

	if resultMsg.Result != "Done processing." {
		t.Errorf("Unexpected result: %s", resultMsg.Result)
	}

	t.Logf("Successfully parsed %d messages from large output", len(messages))
}
