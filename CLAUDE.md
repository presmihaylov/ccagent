# ccagent Project Overview

## What is ccagent?

ccagent is a Go-based CLI application that serves as a bridge between AI coding assistants (Claude Code, Cursor) and team collaboration platforms (Slack, Discord) through the Claude Control platform. It enables teams to interact with AI coding assistants directly from their chat platforms while maintaining proper git workflow and branch management.

## Core Architecture

### Main Components

1. **CLI Agent Interface** (`services/services.go`)
   - Abstraction layer for different AI assistants (Claude, Cursor)
   - Handles conversation management and session tracking

2. **Socket.IO Client** (`cmd/main.go`)
   - Maintains persistent connection to Claude Control platform
   - Routes messages between chat platforms and AI assistants
   - Dual worker pool architecture for sequential conversations and parallel PR checks

3. **Message Handlers** (`handlers/messages.go`)
   - Processes different message types (start conversation, user messages, idle job checks)
   - Manages conversation lifecycle and git operations

4. **Git Integration** (`usecases/git.go`)
   - Automatic branch creation and management
   - Auto-commit functionality
   - Pull request creation and status tracking

5. **Application State** (`models/app_state.go`)
   - In-memory tracking of active jobs and conversations
   - Session management for AI assistant interactions

### Supported AI Assistants

- **Claude Code** (default): Anthropic's CLI tool with configurable permission modes
- **Cursor**: AI-powered code editor integration

### Key Features

- **Branch Management**: Auto-creates ccagent-prefixed branches for each conversation
- **Permission Modes**: 
  - `acceptEdits` (secure, requires approval)
  - `bypassPermissions` (sandbox only, unrestricted access)
- **Auto-commit**: Automatically commits changes with descriptive messages
- **PR Management**: Creates and tracks pull requests automatically
- **Job Lifecycle**: Tracks conversation sessions and cleans up completed jobs
- **Directory Locking**: Prevents multiple instances in same directory

## Development Commands

- **Build**: `make build` - Creates binary in `bin/ccagent`
- **Test**: `make test` or `make test-verbose`
- **Lint**: `make lint` or `make lint-fix`
- **Clean**: `make clean` - Removes build artifacts

## Environment Requirements

- `CCAGENT_API_KEY`: Required API key from Claude Control platform
- `CCAGENT_WS_API_URL`: Optional WebSocket URL (defaults to production)
- Git and GitHub CLI (`gh`) must be configured
- Go 1.24+ for building from source

## Security Considerations

- Secure mode (acceptEdits) recommended for local development
- Bypass permissions mode should only be used in controlled sandbox environments
- Directory locking prevents concurrent instances
- All git operations are tracked and logged

## Log Management

- Logs stored in `~/.config/ccagent/logs/`
- Automatic cleanup of logs older than 7 days
- Both stdout and file logging for debugging