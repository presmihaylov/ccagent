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
}

// AgentExecUser returns the configured user for running agent processes.
// Returns empty string if not configured (self-hosted mode).
func AgentExecUser() string {
	return os.Getenv("AGENT_EXEC_USER")
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
func BuildAgentCommand(name string, args ...string) *exec.Cmd {
	execUser := AgentExecUser()
	filteredEnv := FilterEnvForAgent(os.Environ())

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
