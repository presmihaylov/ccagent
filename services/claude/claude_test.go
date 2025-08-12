package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ccagent/clients"
	"ccagent/core"
	"ccagent/services"
)

func TestNewClaudeService(t *testing.T) {
	mockClient := &services.MockClaudeClient{}
	logDir := "/tmp/test-logs"

	service := NewClaudeService(mockClient, logDir)

	if service.claudeClient != mockClient {
		t.Error("Expected claude client to be set correctly")
	}

	if service.logDir != logDir {
		t.Errorf("Expected logDir to be %s, got %s", logDir, service.logDir)
	}
}

func TestClaudeService_StartNewConversation(t *testing.T) {
	tests := []struct {
		name            string
		prompt          string
		mockOutput      string
		mockError       error
		expectError     bool
		expectedOutput  string
		expectedSession string
	}{
		{
			name:            "successful conversation start",
			prompt:          "Hello",
			mockOutput:      `{"type":"assistant","message":{"id":"msg_123","type":"message","content":[{"type":"text","text":"Hello! How can I help?"}]},"session_id":"session_123"}`,
			mockError:       nil,
			expectError:     false,
			expectedOutput:  "Hello! How can I help?",
			expectedSession: "session_123",
		},
		{
			name:        "client returns error",
			prompt:      "Hello",
			mockOutput:  "",
			mockError:   fmt.Errorf("connection failed"),
			expectError: true,
		},
		{
			name:        "invalid JSON response",
			prompt:      "Hello",
			mockOutput:  "invalid json",
			mockError:   nil,
			expectError: true,
		},
		{
			name:            "empty prompt",
			prompt:          "",
			mockOutput:      `{"type":"assistant","message":{"id":"msg_123","type":"message","content":[{"type":"text","text":"Please provide a prompt."}]},"session_id":"session_123"}`,
			mockError:       nil,
			expectError:     false,
			expectedOutput:  "Please provide a prompt.",
			expectedSession: "session_123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for logs
			tmpDir, err := os.MkdirTemp("", "claude_test_logs_*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Set up mock client
			mockClient := &services.MockClaudeClient{
				StartNewSessionFunc: func(prompt string, options *clients.ClaudeOptions) (string, error) {
					if prompt != tt.prompt {
						t.Errorf("Expected prompt %s, got %s", tt.prompt, prompt)
					}
					return tt.mockOutput, tt.mockError
				},
			}

			service := NewClaudeService(mockClient, tmpDir)

			// Execute
			result, err := service.StartNewConversation(tt.prompt)

			// Verify error expectation
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// If no error expected, verify result
			if !tt.expectError && err == nil {
				if result.Output != tt.expectedOutput {
					t.Errorf("Expected output %q, got %q", tt.expectedOutput, result.Output)
				}
				if result.SessionID != tt.expectedSession {
					t.Errorf("Expected session ID %q, got %q", tt.expectedSession, result.SessionID)
				}
			}

			// Mock verification not needed with function-based mocks
		})
	}
}

func TestClaudeService_StartNewConversationWithSystemPrompt(t *testing.T) {
	tests := []struct {
		name            string
		prompt          string
		systemPrompt    string
		mockOutput      string
		mockError       error
		expectError     bool
		expectedOutput  string
		expectedSession string
	}{
		{
			name:            "successful conversation with system prompt",
			prompt:          "Hello",
			systemPrompt:    "You are a helpful assistant.",
			mockOutput:      `{"type":"assistant","message":{"id":"msg_123","type":"message","content":[{"type":"text","text":"Hello! I'm here to help."}]},"session_id":"session_123"}`,
			mockError:       nil,
			expectError:     false,
			expectedOutput:  "Hello! I'm here to help.",
			expectedSession: "session_123",
		},
		{
			name:         "client returns error",
			prompt:       "Hello",
			systemPrompt: "You are a helpful assistant.",
			mockOutput:   "",
			mockError:    fmt.Errorf("connection failed"),
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for logs
			tmpDir, err := os.MkdirTemp("", "claude_test_logs_*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Set up mock client
			mockClient := &services.MockClaudeClient{
				StartNewSessionFunc: func(prompt string, options *clients.ClaudeOptions) (string, error) {
					if prompt != tt.prompt {
						t.Errorf("Expected prompt %s, got %s", tt.prompt, prompt)
					}
					if options == nil || options.SystemPrompt != tt.systemPrompt {
						t.Errorf("Expected system prompt %s, got %s", tt.systemPrompt, options.SystemPrompt)
					}
					return tt.mockOutput, tt.mockError
				},
			}

			service := NewClaudeService(mockClient, tmpDir)

			// Execute
			result, err := service.StartNewConversationWithSystemPrompt(tt.prompt, tt.systemPrompt)

			// Verify error expectation
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// If no error expected, verify result
			if !tt.expectError && err == nil {
				if result.Output != tt.expectedOutput {
					t.Errorf("Expected output %q, got %q", tt.expectedOutput, result.Output)
				}
				if result.SessionID != tt.expectedSession {
					t.Errorf("Expected session ID %q, got %q", tt.expectedSession, result.SessionID)
				}
			}

			// Mock verification not needed with function-based mocks
		})
	}
}

func TestClaudeService_ContinueConversation(t *testing.T) {
	tests := []struct {
		name            string
		sessionID       string
		prompt          string
		mockOutput      string
		mockError       error
		expectError     bool
		expectedOutput  string
		expectedSession string
	}{
		{
			name:            "successful conversation continue",
			sessionID:       "session_123",
			prompt:          "How are you?",
			mockOutput:      `{"type":"assistant","message":{"id":"msg_456","type":"message","content":[{"type":"text","text":"I'm doing well, thank you!"}]},"session_id":"session_123"}`,
			mockError:       nil,
			expectError:     false,
			expectedOutput:  "I'm doing well, thank you!",
			expectedSession: "session_123",
		},
		{
			name:        "client returns error",
			sessionID:   "session_123",
			prompt:      "How are you?",
			mockOutput:  "",
			mockError:   fmt.Errorf("session not found"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for logs
			tmpDir, err := os.MkdirTemp("", "claude_test_logs_*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Set up mock client
			mockClient := &services.MockClaudeClient{
				ContinueSessionFunc: func(sessionID, prompt string, options *clients.ClaudeOptions) (string, error) {
					if sessionID != tt.sessionID {
						t.Errorf("Expected sessionID %s, got %s", tt.sessionID, sessionID)
					}
					if prompt != tt.prompt {
						t.Errorf("Expected prompt %s, got %s", tt.prompt, prompt)
					}
					return tt.mockOutput, tt.mockError
				},
			}

			service := NewClaudeService(mockClient, tmpDir)

			// Execute
			result, err := service.ContinueConversation(tt.sessionID, tt.prompt)

			// Verify error expectation
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// If no error expected, verify result
			if !tt.expectError && err == nil {
				if result.Output != tt.expectedOutput {
					t.Errorf("Expected output %q, got %q", tt.expectedOutput, result.Output)
				}
				if result.SessionID != tt.expectedSession {
					t.Errorf("Expected session ID %q, got %q", tt.expectedSession, result.SessionID)
				}
			}

			// Mock verification not needed with function-based mocks
		})
	}
}

func TestClaudeService_writeClaudeSessionLog(t *testing.T) {
	mockClient := &services.MockClaudeClient{}
	tmpDir, err := os.MkdirTemp("", "claude_test_logs_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	service := NewClaudeService(mockClient, tmpDir)

	rawOutput := "Test Claude session output"
	logPath, err := service.writeClaudeSessionLog(rawOutput)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify log file was created
	if !strings.Contains(logPath, tmpDir) {
		t.Errorf("Log path should be in temp directory")
	}

	if !strings.Contains(logPath, "claude-session-") {
		t.Errorf("Log filename should contain 'claude-session-'")
	}

	// Verify content was written
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if string(content) != rawOutput {
		t.Errorf("Expected log content %q, got %q", rawOutput, string(content))
	}
}

func TestClaudeService_CleanupOldLogs(t *testing.T) {
	mockClient := &services.MockClaudeClient{}
	tmpDir, err := os.MkdirTemp("", "claude_test_logs_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	service := NewClaudeService(mockClient, tmpDir)

	// Create some test log files with different ages
	now := time.Now()
	oldTime := now.AddDate(0, 0, -10)   // 10 days ago
	recentTime := now.AddDate(0, 0, -3) // 3 days ago

	oldLogFile := filepath.Join(tmpDir, "claude-session-20240101-120000.log")
	recentLogFile := filepath.Join(tmpDir, "claude-session-20240110-120000.log")
	nonClaudeFile := filepath.Join(tmpDir, "other-file.log")

	// Create test files
	if err := os.WriteFile(oldLogFile, []byte("old log"), 0644); err != nil {
		t.Fatalf("Failed to create old log file: %v", err)
	}
	if err := os.WriteFile(recentLogFile, []byte("recent log"), 0644); err != nil {
		t.Fatalf("Failed to create recent log file: %v", err)
	}
	if err := os.WriteFile(nonClaudeFile, []byte("other file"), 0644); err != nil {
		t.Fatalf("Failed to create non-claude file: %v", err)
	}

	// Set file times
	if err := os.Chtimes(oldLogFile, oldTime, oldTime); err != nil {
		t.Fatalf("Failed to set old file time: %v", err)
	}
	if err := os.Chtimes(recentLogFile, recentTime, recentTime); err != nil {
		t.Fatalf("Failed to set recent file time: %v", err)
	}

	// Run cleanup for files older than 7 days
	err = service.CleanupOldLogs(7)
	if err != nil {
		t.Fatalf("Unexpected error during cleanup: %v", err)
	}

	// Verify old file was removed
	if _, err := os.Stat(oldLogFile); !os.IsNotExist(err) {
		t.Errorf("Old log file should have been removed")
	}

	// Verify recent file still exists
	if _, err := os.Stat(recentLogFile); err != nil {
		t.Errorf("Recent log file should still exist")
	}

	// Verify non-claude file still exists
	if _, err := os.Stat(nonClaudeFile); err != nil {
		t.Errorf("Non-claude file should still exist")
	}
}

func TestClaudeService_CleanupOldLogs_InvalidMaxAge(t *testing.T) {
	mockClient := &services.MockClaudeClient{}
	tmpDir, err := os.MkdirTemp("", "claude_test_logs_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	service := NewClaudeService(mockClient, tmpDir)

	// Test invalid maxAgeDays values
	invalidValues := []int{0, -1, -10}
	for _, maxAge := range invalidValues {
		err := service.CleanupOldLogs(maxAge)
		if err == nil {
			t.Errorf("Expected error for maxAgeDays=%d, but got none", maxAge)
		}
	}
}

func TestClaudeService_CleanupOldLogs_NonExistentDirectory(t *testing.T) {
	mockClient := &services.MockClaudeClient{}
	service := NewClaudeService(mockClient, "/non/existent/directory")

	// Should not return error for non-existent directory (it's a no-op)
	err := service.CleanupOldLogs(7)
	if err != nil {
		t.Errorf("Expected no error for non-existent directory, but got: %v", err)
	}
}

func TestClaudeService_extractSessionID(t *testing.T) {
	mockClient := &services.MockClaudeClient{}
	service := NewClaudeService(mockClient, "/tmp")

	tests := []struct {
		name     string
		messages []services.ClaudeMessage
		expected string
	}{
		{
			name:     "empty messages",
			messages: []services.ClaudeMessage{},
			expected: "unknown",
		},
		{
			name: "single message with session ID",
			messages: []services.ClaudeMessage{
				services.AssistantMessage{
					Type: "assistant",
					Message: struct {
						ID      string            `json:"id"`
						Type    string            `json:"type"`
						Content []json.RawMessage `json:"content"`
					}{
						ID:      "msg_123",
						Type:    "message",
						Content: []json.RawMessage{},
					},
					SessionID: "session_123",
				},
			},
			expected: "session_123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.extractSessionID(tt.messages)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestClaudeService_extractClaudeResult(t *testing.T) {
	mockClient := &services.MockClaudeClient{}
	service := NewClaudeService(mockClient, "/tmp")

	tests := []struct {
		name        string
		messages    []services.ClaudeMessage
		expected    string
		expectError bool
	}{
		{
			name:        "empty messages",
			messages:    []services.ClaudeMessage{},
			expected:    "",
			expectError: true,
		},
		{
			name: "valid assistant message with text",
			messages: []services.ClaudeMessage{
				services.AssistantMessage{
					Type: "assistant",
					Message: struct {
						ID      string            `json:"id"`
						Type    string            `json:"type"`
						Content []json.RawMessage `json:"content"`
					}{
						ID:   "msg_123",
						Type: "message",
						Content: []json.RawMessage{
							json.RawMessage(`{"type":"text","text":"Hello World!"}`),
						},
					},
					SessionID: "session_123",
				},
			},
			expected:    "Hello World!",
			expectError: false,
		},
		{
			name: "assistant message without text content",
			messages: []services.ClaudeMessage{
				services.AssistantMessage{
					Type: "assistant",
					Message: struct {
						ID      string            `json:"id"`
						Type    string            `json:"type"`
						Content []json.RawMessage `json:"content"`
					}{
						ID:   "msg_123",
						Type: "message",
						Content: []json.RawMessage{
							json.RawMessage(`{"type":"image","url":"http://example.com/image.jpg"}`),
						},
					},
					SessionID: "session_123",
				},
			},
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.extractClaudeResult(tt.messages)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError && result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestClaudeService_handleClaudeClientError(t *testing.T) {
	mockClient := &services.MockClaudeClient{}
	tmpDir, err := os.MkdirTemp("", "claude_test_logs_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	service := NewClaudeService(mockClient, tmpDir)

	tests := []struct {
		name            string
		inputError      error
		operation       string
		expectedContain string
	}{
		{
			name:            "nil error",
			inputError:      nil,
			operation:       "test operation",
			expectedContain: "",
		},
		{
			name:            "regular error",
			inputError:      fmt.Errorf("regular error"),
			operation:       "test operation",
			expectedContain: "test operation: regular error",
		},
		{
			name: "claude command error with valid output",
			inputError: &core.ErrClaudeCommandErr{
				Output: `{"type":"assistant","message":{"id":"msg_123","type":"message","content":[{"type":"text","text":"Claude error message"}]},"session_id":"session_123"}`,
				Err:    fmt.Errorf("command failed"),
			},
			operation:       "test operation",
			expectedContain: "test operation: Claude error message",
		},
		{
			name: "claude command error with invalid output",
			inputError: &core.ErrClaudeCommandErr{
				Output: "invalid json output",
				Err:    fmt.Errorf("command failed"),
			},
			operation:       "test operation",
			expectedContain: "test operation: claude command failed: command failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.handleClaudeClientError(tt.inputError, tt.operation)

			if tt.inputError == nil {
				if result != nil {
					t.Errorf("Expected nil result for nil input error, got: %v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("Expected error result but got nil")
				return
			}

			if !strings.Contains(result.Error(), tt.expectedContain) {
				t.Errorf("Expected error to contain %q, got: %v", tt.expectedContain, result.Error())
			}
		})
	}
}

func TestClaudeService_ParseErrorHandling(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "claude_test_logs_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Mock output that doesn't contain valid assistant message (will fail at extract stage)
	mockClient := &services.MockClaudeClient{
		StartNewSessionFunc: func(prompt string, options *clients.ClaudeOptions) (string, error) {
			if prompt == "test" {
				return "invalid json", nil
			}
			return "", fmt.Errorf("unexpected prompt: %s", prompt)
		},
	}

	service := NewClaudeService(mockClient, tmpDir)

	result, err := service.StartNewConversation("test")

	// Should return error (not ClaudeParseError since parsing succeeds but extraction fails)
	if err == nil {
		t.Errorf("Expected error but got no error")
	}

	if result != nil {
		t.Errorf("Expected nil result on error, got: %v", result)
	}

	// Check that error contains expected message about no ExitPlanMode, result or assistant message
	if !strings.Contains(err.Error(), "no ExitPlanMode, result, or assistant message with text content found") {
		t.Errorf("Expected error about no ExitPlanMode, result or assistant message, got: %v", err)
	}

	// Mock verification not needed with function-based mocks
}

func TestClaudeService_ActualParseError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "claude_test_logs_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a mock output that will cause MapClaudeOutputToMessages to return an error
	// We'll use a string that causes the scanner to fail somehow
	// After checking the code, the scanner is pretty robust, so let's create a scenario
	// where we can force an error by creating extremely long content that exceeds buffer limits
	longLine := strings.Repeat("x", 2*1024*1024) // 2MB line, exceeds the 1MB buffer
	mockClient := &services.MockClaudeClient{
		StartNewSessionFunc: func(prompt string, options *clients.ClaudeOptions) (string, error) {
			if prompt == "test" {
				return longLine, nil
			}
			return "", fmt.Errorf("unexpected prompt: %s", prompt)
		},
	}

	service := NewClaudeService(mockClient, tmpDir)

	result, err := service.StartNewConversation("test")

	// Should return some kind of error
	if err == nil {
		t.Errorf("Expected error but got no error")
	}

	if result != nil {
		t.Errorf("Expected nil result on error, got: %v", result)
	}

	// Mock verification not needed with function-based mocks
}

func TestClaudeService_WriteErrorLogHandling(t *testing.T) {
	// Use non-existent parent directory to cause write error
	nonExistentDir := "/non/existent/parent/dir"

	mockClient := &services.MockClaudeClient{
		StartNewSessionFunc: func(prompt string, options *clients.ClaudeOptions) (string, error) {
			if prompt == "test" {
				return `{"type":"assistant","message":{"id":"msg_123","type":"message","content":[{"type":"text","text":"Hello"}]},"session_id":"session_123"}`, nil
			}
			return "", fmt.Errorf("unexpected prompt: %s", prompt)
		},
	}

	service := NewClaudeService(mockClient, nonExistentDir)

	// This should still work despite log write error
	result, err := service.StartNewConversation("test")

	if err != nil {
		t.Errorf("Expected successful operation despite log write error, got: %v", err)
	}

	if result == nil || result.Output != "Hello" {
		t.Errorf("Expected valid result despite log write error")
	}

	// Mock verification not needed with function-based mocks
}
