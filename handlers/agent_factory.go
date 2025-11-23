package handlers

import (
	"fmt"
	"os/exec"

	"ccagent/clients"
	claudeclient "ccagent/clients/claude"
	codexclient "ccagent/clients/codex"
	cursorclient "ccagent/clients/cursor"
	"ccagent/core/env"
	"ccagent/services"
	claudeservice "ccagent/services/claude"
	codexservice "ccagent/services/codex"
	cursorservice "ccagent/services/cursor"
)

// ValidateBinaryExists checks if the binary for the given agent exists in PATH
func ValidateBinaryExists(agent string) error {
	var binaryName string
	switch agent {
	case "claude":
		binaryName = "claude"
	case "codex":
		binaryName = "codex"
	case "cursor":
		binaryName = "cursor"
	default:
		return fmt.Errorf("unsupported agent type: %s (supported: claude, codex, cursor)", agent)
	}

	_, err := exec.LookPath(binaryName)
	if err != nil {
		return fmt.Errorf("binary '%s' not found in PATH for agent '%s'", binaryName, agent)
	}

	return nil
}

// CreateCLIAgent creates a CLI agent instance for the given agent type and model
// This function validates the binary exists before creating the agent
func CreateCLIAgent(
	agent, model, permissionMode, logDir, workDir string,
	agentsApiClient *clients.AgentsApiClient,
	envManager *env.EnvManager,
) (services.CLIAgent, error) {
	// Validate binary exists first
	if err := ValidateBinaryExists(agent); err != nil {
		return nil, err
	}

	// Create the appropriate agent
	switch agent {
	case "claude":
		claudeClient := claudeclient.NewClaudeClient(permissionMode)
		return claudeservice.NewClaudeService(claudeClient, logDir, agentsApiClient, envManager), nil
	case "codex":
		codexClient := codexclient.NewCodexClient(permissionMode, workDir)
		return codexservice.NewCodexService(codexClient, logDir, model), nil
	case "cursor":
		cursorClient := cursorclient.NewCursorClient()
		return cursorservice.NewCursorService(cursorClient, logDir, model), nil
	default:
		return nil, fmt.Errorf("unsupported agent type: %s", agent)
	}
}
