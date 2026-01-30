package services

import (
	"strings"
	"testing"
)

func TestRegressionLargeStdoutParsing(t *testing.T) {
	// This test reproduces the actual production failure
	// where tool_use_result.stdout was 64MB and caused parsing to fail.
	// With bufio.Reader (no max line length), this is handled without issues.

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

func TestRegressionLargeTaskOutputParsing(t *testing.T) {
	// This test reproduces the vibegest-agent production failure (2026-01-29)
	// where tool_use_result.task.output was 2.8MB x2 fields = 6.2MB single JSON line.
	// The old bufio.Scanner with 4MB buffer could not handle this.
	// bufio.Reader has no such limit.

	largeTaskOutput := strings.Repeat("a", 3*1024*1024) // 3MB per field

	input := `{"type":"system","subtype":"init","session_id":"test-session-001"}
{"type":"user","message":{"role":"user","content":[{"tool_use_id":"toolu_001","type":"tool_result","content":"short"}]},"session_id":"test-session-001","tool_use_result":{"task":{"output":"` + largeTaskOutput + `","result":"` + largeTaskOutput + `"}}}
{"type":"result","subtype":"success","is_error":false,"result":"Done","session_id":"test-session-001"}`

	t.Logf("Test input size: %.2f MB", float64(len(input))/(1024*1024))

	messages, err := MapClaudeOutputToMessages(input)
	if err != nil {
		t.Fatalf("REGRESSION: Failed to parse output with large task output/result fields: %v", err)
	}

	if len(messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}

	t.Logf("SUCCESS: Parsed %d messages", len(messages))
}
