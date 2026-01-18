package opencode

import (
	"strings"

	"ccagent/clients"
	"ccagent/core"
	"ccagent/core/log"
)

type OpenCodeClient struct {
	// No permissionsMode needed as we only support `bypassPermissions` for now.
}

func NewOpenCodeClient() *OpenCodeClient {
	return &OpenCodeClient{}
}

func (c *OpenCodeClient) StartNewSession(prompt string, options *clients.OpenCodeOptions) (string, error) {
	log.Info("📋 Starting to create new OpenCode session")

	args := []string{
		"run",
		"--format", "json",
		"--agent", "build", // Always use build mode until `acceptEdits` support is added
	}

	// Add model from options if provided
	if options != nil && options.Model != "" {
		args = append(args, "--model", options.Model)
	}

	// Append prompt as the last argument
	args = append(args, prompt)

	log.Info("Starting new OpenCode session with prompt: %s", prompt)
	log.Info("Command arguments: %v", args)

	cmd := clients.BuildAgentCommand("opencode", args...)

	log.Info("Running OpenCode command")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", &core.ErrClaudeCommandErr{
			Err:    err,
			Output: string(output),
		}
	}

	result := strings.TrimSpace(string(output))
	log.Info("OpenCode command completed successfully, outputLength: %d", len(result))
	log.Info("📋 Completed successfully - created new OpenCode session")
	return result, nil
}

func (c *OpenCodeClient) ContinueSession(sessionID, prompt string, options *clients.OpenCodeOptions) (string, error) {
	log.Info("📋 Starting to continue OpenCode session: %s", sessionID)

	args := []string{
		"run",
		"--session", sessionID,
		"--format", "json",
		"--agent", "build", // Always use build mode until `acceptEdits` support is added
	}

	// Add model from options if provided
	if options != nil && options.Model != "" {
		args = append(args, "--model", options.Model)
	}

	// Append prompt as the last argument
	args = append(args, prompt)

	log.Info("Executing OpenCode command with sessionID: %s, prompt: %s", sessionID, prompt)
	log.Info("Command arguments: %v", args)

	cmd := clients.BuildAgentCommand("opencode", args...)

	log.Info("Running OpenCode command")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", &core.ErrClaudeCommandErr{
			Err:    err,
			Output: string(output),
		}
	}

	result := strings.TrimSpace(string(output))
	log.Info("OpenCode command completed successfully, outputLength: %d", len(result))
	log.Info("📋 Completed successfully - continued OpenCode session")
	return result, nil
}
