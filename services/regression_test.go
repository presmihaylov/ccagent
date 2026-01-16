package services

import (
	"strings"
	"testing"
)

func TestRegressionLargeStdoutParsing(t *testing.T) {
	// This test reproduces the actual production failure
	// where tool_use_result.stdout was 64MB and caused parsing to fail
	
	largeStdout := strings.Repeat("output line\\n", 500000) // ~6MB

	input := `{"type":"system","subtype":"init","session_id":"test-session-001"}
{"type":"user","message":{"role":"user","content":[{"tool_use_id":"toolu_001","type":"tool_result","content":"output"}]},"session_id":"test-session-001","tool_use_result":{"stdout":"` + largeStdout + `","stderr":"","interrupted":false}}
{"type":"result","subtype":"success","is_error":false,"result":"Done","session_id":"test-session-001"}`

	t.Logf("Test input size: %.2f MB", float64(len(input))/(1024*1024))

	messages, err := MapClaudeOutputToMessages(input)
	if err != nil {
		t.Fatalf("REGRESSION: Failed to parse output with large tool_use_result.stdout: %v", err)
	}

	if len(messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}

	t.Logf("SUCCESS: Parsed %d messages", len(messages))
}

func TestRegressionEscapedQuotesInToolResult(t *testing.T) {
	// This test reproduces the regex bug where escaped quotes caused
	// the regex to capture only part of the content
	
	// Large content with escaped quotes (simulating embedded JSON)
	content := strings.Repeat(`text with \"escaped\" json `, 5000)
	
	input := `{"type":"user","message":{"role":"user","content":[{"tool_use_id":"toolu_001","type":"tool_result","content":"` + content + `"}]},"session_id":"test"}`
	
	// Apply the strip function
	result := stripLargeToolResultContent(input)
	
	// The content is >100KB, so it should be truncated
	if !strings.Contains(result, "[CONTENT_TRUNCATED_") {
		t.Errorf("REGRESSION: Content with escaped quotes was not properly truncated. Got length: %d", len(result))
	}
	
	t.Logf("SUCCESS: Content properly handled")
}
