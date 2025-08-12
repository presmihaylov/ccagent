package services

import (
	"encoding/json"
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
