package clients

import (
	"os"
	"strings"
	"testing"
)

func TestFilterEnvForAgent(t *testing.T) {
	env := []string{
		"PATH=/usr/bin",
		"CCAGENT_API_KEY=secret_api_key",
		"ANTHROPIC_API_KEY=sk-ant-xxx",
		"CCAGENT_WS_API_URL=wss://api.example.com",
		"AGENT_EXEC_USER=agentrunner",
		"HOME=/home/user",
	}

	filtered := FilterEnvForAgent(env)

	// Check blocked vars are removed
	for _, e := range filtered {
		for blocked := range BlockedEnvVars {
			if strings.HasPrefix(e, blocked+"=") {
				t.Errorf("Blocked var %s should be filtered out, but found: %s", blocked, e)
			}
		}
	}

	// Check allowed vars are preserved
	expectedVars := map[string]bool{
		"PATH":              false,
		"ANTHROPIC_API_KEY": false,
		"HOME":              false,
	}

	for _, e := range filtered {
		for expected := range expectedVars {
			if strings.HasPrefix(e, expected+"=") {
				expectedVars[expected] = true
			}
		}
	}

	for varName, found := range expectedVars {
		if !found {
			t.Errorf("Expected var %s should be preserved but was not found", varName)
		}
	}

	// Verify count: 6 original - 3 blocked = 3 remaining
	if len(filtered) != 3 {
		t.Errorf("Expected 3 filtered vars, got %d", len(filtered))
	}
}

func TestFilterEnvForAgent_EmptyEnv(t *testing.T) {
	filtered := FilterEnvForAgent([]string{})
	if len(filtered) != 0 {
		t.Errorf("Expected empty filtered env, got %d items", len(filtered))
	}
}

func TestFilterEnvForAgent_NoBlockedVars(t *testing.T) {
	env := []string{
		"PATH=/usr/bin",
		"HOME=/home/user",
	}

	filtered := FilterEnvForAgent(env)
	if len(filtered) != 2 {
		t.Errorf("Expected 2 filtered vars, got %d", len(filtered))
	}
}

func TestBuildShellCommand(t *testing.T) {
	tests := []struct {
		name     string
		cmdName  string
		args     []string
		expected string
	}{
		{
			name:     "simple command",
			cmdName:  "claude",
			args:     []string{"--version"},
			expected: "claude '--version'",
		},
		{
			name:     "multiple args",
			cmdName:  "claude",
			args:     []string{"--model", "claude-3", "-p", "hello"},
			expected: "claude '--model' 'claude-3' '-p' 'hello'",
		},
		{
			name:     "args with single quotes",
			cmdName:  "claude",
			args:     []string{"-p", "Hello 'world'"},
			expected: "claude '-p' 'Hello '\\''world'\\'''",
		},
		{
			name:     "empty args",
			cmdName:  "claude",
			args:     []string{},
			expected: "claude",
		},
		{
			name:     "args with spaces",
			cmdName:  "claude",
			args:     []string{"-p", "hello world"},
			expected: "claude '-p' 'hello world'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildShellCommand(tt.cmdName, tt.args)
			if result != tt.expected {
				t.Errorf("buildShellCommand(%q, %v) = %q, want %q",
					tt.cmdName, tt.args, result, tt.expected)
			}
		})
	}
}

func TestAgentExecUser(t *testing.T) {
	// Save original value
	original := os.Getenv("AGENT_EXEC_USER")
	defer os.Setenv("AGENT_EXEC_USER", original)

	// Test when not set
	os.Unsetenv("AGENT_EXEC_USER")
	if user := AgentExecUser(); user != "" {
		t.Errorf("AgentExecUser() = %q, want empty string", user)
	}

	// Test when set
	os.Setenv("AGENT_EXEC_USER", "agentrunner")
	if user := AgentExecUser(); user != "agentrunner" {
		t.Errorf("AgentExecUser() = %q, want %q", user, "agentrunner")
	}
}

func TestBuildAgentCommand_SelfHosted(t *testing.T) {
	// Save original value
	original := os.Getenv("AGENT_EXEC_USER")
	defer os.Setenv("AGENT_EXEC_USER", original)

	os.Unsetenv("AGENT_EXEC_USER")
	cmd := BuildAgentCommand("echo", "hello")

	// In self-hosted mode, should run the command directly
	if cmd.Args[0] != "echo" {
		t.Errorf("Expected echo command in self-hosted mode, got %v", cmd.Args)
	}

	// Check that blocked env vars are filtered
	for _, e := range cmd.Env {
		for blocked := range BlockedEnvVars {
			if strings.HasPrefix(e, blocked+"=") {
				t.Errorf("Blocked var %s should be filtered", blocked)
			}
		}
	}
}

func TestBuildAgentCommand_Managed(t *testing.T) {
	// Save original value
	original := os.Getenv("AGENT_EXEC_USER")
	defer os.Setenv("AGENT_EXEC_USER", original)

	os.Setenv("AGENT_EXEC_USER", "agentrunner")
	cmd := BuildAgentCommand("echo", "hello")

	// In managed mode, should use su
	if cmd.Args[0] != "su" {
		t.Errorf("Expected su command in managed mode, got %v", cmd.Args)
	}

	// Verify su arguments structure: su -s /bin/sh -c "command" username
	expectedArgs := []string{"su", "-s", "/bin/sh", "-c", "echo 'hello'", "agentrunner"}
	if len(cmd.Args) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d: %v", len(expectedArgs), len(cmd.Args), cmd.Args)
	}

	for i, arg := range expectedArgs {
		if i < len(cmd.Args) && cmd.Args[i] != arg {
			t.Errorf("Arg %d: expected %q, got %q", i, arg, cmd.Args[i])
		}
	}
}
