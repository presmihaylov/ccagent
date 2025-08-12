package clients

// ClaudeOptions contains optional parameters for Claude CLI interactions
type ClaudeOptions struct {
	SystemPrompt    string
	DisallowedTools []string
}

// CursorOptions contains optional parameters for Cursor CLI interactions
type CursorOptions struct {
	SystemPrompt string
	Model        string
}

// ClaudeClient defines the interface for Claude CLI interactions
type ClaudeClient interface {
	StartNewSession(prompt string, options *ClaudeOptions) (string, error)
	ContinueSession(sessionID, prompt string, options *ClaudeOptions) (string, error)
}

// CursorClient defines the interface for Cursor CLI interactions
type CursorClient interface {
	StartNewSession(prompt string, options *CursorOptions) (string, error)
	ContinueSession(sessionID, prompt string, options *CursorOptions) (string, error)
}
