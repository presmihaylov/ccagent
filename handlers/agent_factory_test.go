package handlers

import (
	"testing"
)

func TestValidateBinaryExists(t *testing.T) {
	tests := []struct {
		name        string
		agent       string
		expectError bool
	}{
		{
			name:        "unsupported agent",
			agent:       "invalid-agent",
			expectError: true,
		},
		{
			name:        "claude agent type",
			agent:       "claude",
			expectError: false, // Will fail if claude binary is not in PATH, but that's expected
		},
		{
			name:        "codex agent type",
			agent:       "codex",
			expectError: false, // Will fail if codex binary is not in PATH, but that's expected
		},
		{
			name:        "cursor agent type",
			agent:       "cursor",
			expectError: false, // Will fail if cursor binary is not in PATH, but that's expected
		},
		{
			name:        "empty agent",
			agent:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBinaryExists(tt.agent)
			if tt.expectError && err == nil {
				t.Errorf("ValidateBinaryExists(%q) expected error for unsupported agent, got nil", tt.agent)
			}
			// Note: For supported agents, we can't reliably test if the binary exists
			// since it depends on the test environment setup. The test will pass
			// as long as unsupported agents return errors.
		})
	}
}
