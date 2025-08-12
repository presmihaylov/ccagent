package services

// CLIAgentResult represents the result of a CLI agent conversation
type CLIAgentResult struct {
	Output    string
	SessionID string
}

// CLIAgent defines the interface for CLI agent operations like Claude Code, Cursor, etc.
type CLIAgent interface {
	// StartNewConversation starts a new conversation with a prompt
	StartNewConversation(prompt string) (*CLIAgentResult, error)

	// StartNewConversationWithSystemPrompt starts a new conversation with both user and system prompts
	StartNewConversationWithSystemPrompt(prompt, systemPrompt string) (*CLIAgentResult, error)

	// StartNewConversationWithDisallowedTools starts a new conversation with specified disallowed tools
	StartNewConversationWithDisallowedTools(prompt string, disallowedTools []string) (*CLIAgentResult, error)

	// ContinueConversation continues an existing conversation
	ContinueConversation(sessionID, prompt string) (*CLIAgentResult, error)

	// CleanupOldLogs removes old log files based on age
	CleanupOldLogs(maxAgeDays int) error

	// AgentName returns the identifier for the concrete agent implementation
	// (e.g., "claude" or "cursor") so callers can adapt behavior per agent
	AgentName() string
}
