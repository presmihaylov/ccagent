// Package clients provides utilities for spawning agent processes.
package clients

import (
	"os"
	"os/exec"
	"strings"
)

// BlockedEnvVars lists environment variables that should never be passed to agent processes.
// These contain sensitive credentials that agents should not have access to.
var BlockedEnvVars = map[string]bool{
	"CCAGENT_API_KEY":    true,
	"CCAGENT_WS_API_URL": true,
	"AGENT_EXEC_USER":    true,
	"AGENT_HTTP_PROXY":   true, // This is for ccagent to read, not for agents
}

// AgentExecUser returns the configured user for running agent processes.
// Returns empty string if not configured (self-hosted mode).
func AgentExecUser() string {
	return os.Getenv("AGENT_EXEC_USER")
}

// AgentHTTPProxy returns the HTTP proxy URL that agent processes should use.
// This is read from AGENT_HTTP_PROXY and injected into agent processes as HTTP_PROXY/HTTPS_PROXY.
// Returns empty string if not configured.
func AgentHTTPProxy() string {
	return os.Getenv("AGENT_HTTP_PROXY")
}

// BuildAgentCommand creates an exec.Cmd that runs the given command
// as the configured agent user (or current user if not configured).
//
// When AGENT_EXEC_USER is set, the command is wrapped with 'su' to run
// as the specified user, providing process isolation that prevents
// agents from reading the parent process's /proc/*/environ.
//
// Sensitive environment variables (CCAGENT_API_KEY, etc.) are always
// filtered from the command's environment.
//
// If AGENT_HTTP_PROXY is configured, HTTP_PROXY and HTTPS_PROXY are
// injected into the agent's environment to route traffic through the proxy.
func BuildAgentCommand(name string, args ...string) *exec.Cmd {
	execUser := AgentExecUser()
	filteredEnv := FilterEnvForAgent(os.Environ())

	// Inject HTTP proxy settings for agent processes if configured
	filteredEnv = InjectProxyEnv(filteredEnv)

	if execUser == "" {
		// Self-hosted mode: run as current user
		cmd := exec.Command(name, args...)
		cmd.Env = filteredEnv
		return cmd
	}

	// Managed mode: run as specified user via su
	fullCommand := buildShellCommand(name, args)
	cmd := exec.Command("su", "-s", "/bin/sh", "-c", fullCommand, execUser)
	cmd.Env = filteredEnv
	return cmd
}

// buildShellCommand safely constructs a shell command string with escaped arguments.
// Single quotes are escaped using the '\” pattern.
func buildShellCommand(name string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, name)
	for _, arg := range args {
		// Escape single quotes in arguments
		escaped := strings.ReplaceAll(arg, "'", "'\\''")
		parts = append(parts, "'"+escaped+"'")
	}
	return strings.Join(parts, " ")
}

// FilterEnvForAgent removes sensitive variables from environment.
// This prevents agent processes from accessing credentials like CCAGENT_API_KEY.
func FilterEnvForAgent(env []string) []string {
	var filtered []string
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) < 1 {
			continue
		}
		key := parts[0]
		if !BlockedEnvVars[key] {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// InjectProxyEnv adds HTTP_PROXY and HTTPS_PROXY to the environment if AGENT_HTTP_PROXY is set.
// This ensures agent processes route their traffic through the secret proxy while the
// ccagent process itself does not use the proxy (allowing it to reach the backend).
func InjectProxyEnv(env []string) []string {
	proxyURL := AgentHTTPProxy()
	if proxyURL == "" {
		return env
	}

	// Check if HTTP_PROXY or HTTPS_PROXY already exist in env
	hasHTTPProxy := false
	hasHTTPSProxy := false
	for _, e := range env {
		if strings.HasPrefix(e, "HTTP_PROXY=") || strings.HasPrefix(e, "http_proxy=") {
			hasHTTPProxy = true
		}
		if strings.HasPrefix(e, "HTTPS_PROXY=") || strings.HasPrefix(e, "https_proxy=") {
			hasHTTPSProxy = true
		}
	}

	// Only add if not already present (don't override explicit settings)
	if !hasHTTPProxy {
		env = append(env, "HTTP_PROXY="+proxyURL)
		env = append(env, "http_proxy="+proxyURL) // Some tools use lowercase
	}
	if !hasHTTPSProxy {
		env = append(env, "HTTPS_PROXY="+proxyURL)
		env = append(env, "https_proxy="+proxyURL) // Some tools use lowercase
	}

	return env
}
