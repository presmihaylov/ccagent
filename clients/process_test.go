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

func TestAgentHTTPProxy(t *testing.T) {
	// Save original value
	original := os.Getenv("AGENT_HTTP_PROXY")
	defer os.Setenv("AGENT_HTTP_PROXY", original)

	// Test when not set
	os.Unsetenv("AGENT_HTTP_PROXY")
	if proxy := AgentHTTPProxy(); proxy != "" {
		t.Errorf("AgentHTTPProxy() = %q, want empty string", proxy)
	}

	// Test when set
	os.Setenv("AGENT_HTTP_PROXY", "http://proxy:8080")
	if proxy := AgentHTTPProxy(); proxy != "http://proxy:8080" {
		t.Errorf("AgentHTTPProxy() = %q, want %q", proxy, "http://proxy:8080")
	}
}

func TestInjectProxyEnv_NoProxy(t *testing.T) {
	// Save original value
	original := os.Getenv("AGENT_HTTP_PROXY")
	defer os.Setenv("AGENT_HTTP_PROXY", original)

	os.Unsetenv("AGENT_HTTP_PROXY")

	env := []string{"PATH=/usr/bin", "HOME=/home/user"}
	result := InjectProxyEnv(env)

	// Should return unchanged when no proxy configured
	if len(result) != len(env) {
		t.Errorf("Expected %d vars, got %d", len(env), len(result))
	}
}

func TestInjectProxyEnv_WithProxy(t *testing.T) {
	// Save original value
	original := os.Getenv("AGENT_HTTP_PROXY")
	defer os.Setenv("AGENT_HTTP_PROXY", original)

	os.Setenv("AGENT_HTTP_PROXY", "http://proxy:8080")

	env := []string{"PATH=/usr/bin", "HOME=/home/user"}
	result := InjectProxyEnv(env)

	// Should add HTTP_PROXY, http_proxy, HTTPS_PROXY, https_proxy
	expectedLen := len(env) + 4
	if len(result) != expectedLen {
		t.Errorf("Expected %d vars, got %d", expectedLen, len(result))
	}

	// Check that proxy vars are present
	hasHTTPProxy := false
	hasHTTPSProxy := false
	hasLowerHTTPProxy := false
	hasLowerHTTPSProxy := false

	for _, e := range result {
		switch {
		case strings.HasPrefix(e, "HTTP_PROXY=http://proxy:8080"):
			hasHTTPProxy = true
		case strings.HasPrefix(e, "HTTPS_PROXY=http://proxy:8080"):
			hasHTTPSProxy = true
		case strings.HasPrefix(e, "http_proxy=http://proxy:8080"):
			hasLowerHTTPProxy = true
		case strings.HasPrefix(e, "https_proxy=http://proxy:8080"):
			hasLowerHTTPSProxy = true
		}
	}

	if !hasHTTPProxy {
		t.Error("HTTP_PROXY not found in result")
	}
	if !hasHTTPSProxy {
		t.Error("HTTPS_PROXY not found in result")
	}
	if !hasLowerHTTPProxy {
		t.Error("http_proxy not found in result")
	}
	if !hasLowerHTTPSProxy {
		t.Error("https_proxy not found in result")
	}
}

func TestInjectProxyEnv_DoesNotOverride(t *testing.T) {
	// Save original value
	original := os.Getenv("AGENT_HTTP_PROXY")
	defer os.Setenv("AGENT_HTTP_PROXY", original)

	os.Setenv("AGENT_HTTP_PROXY", "http://proxy:8080")

	// Env already has proxy vars
	env := []string{
		"PATH=/usr/bin",
		"HTTP_PROXY=http://existing:3128",
		"HTTPS_PROXY=http://existing:3128",
	}
	result := InjectProxyEnv(env)

	// Should not add new proxy vars if they already exist
	if len(result) != len(env) {
		t.Errorf("Expected %d vars (no additions), got %d", len(env), len(result))
	}

	// Verify existing proxy values are preserved
	for _, e := range result {
		if strings.HasPrefix(e, "HTTP_PROXY=") && e != "HTTP_PROXY=http://existing:3128" {
			t.Errorf("HTTP_PROXY was overridden: %s", e)
		}
		if strings.HasPrefix(e, "HTTPS_PROXY=") && e != "HTTPS_PROXY=http://existing:3128" {
			t.Errorf("HTTPS_PROXY was overridden: %s", e)
		}
	}
}

func TestBuildAgentCommand_InjectsProxy(t *testing.T) {
	// Save original values
	origUser := os.Getenv("AGENT_EXEC_USER")
	origProxy := os.Getenv("AGENT_HTTP_PROXY")
	defer func() {
		os.Setenv("AGENT_EXEC_USER", origUser)
		os.Setenv("AGENT_HTTP_PROXY", origProxy)
	}()

	os.Unsetenv("AGENT_EXEC_USER")
	os.Setenv("AGENT_HTTP_PROXY", "http://secret-proxy:8080")

	cmd := BuildAgentCommand("echo", "hello")

	// Check that proxy vars are in the command's environment
	hasHTTPProxy := false
	hasHTTPSProxy := false

	for _, e := range cmd.Env {
		if strings.HasPrefix(e, "HTTP_PROXY=http://secret-proxy:8080") {
			hasHTTPProxy = true
		}
		if strings.HasPrefix(e, "HTTPS_PROXY=http://secret-proxy:8080") {
			hasHTTPSProxy = true
		}
	}

	if !hasHTTPProxy {
		t.Error("HTTP_PROXY not injected into command environment")
	}
	if !hasHTTPSProxy {
		t.Error("HTTPS_PROXY not injected into command environment")
	}
}
