# ccagent

A Go-based CLI agent that connects AI assistants (Claude Code, Cursor) to team collaboration platforms like Slack and Discord through the [Claude Control platform](https://claudecontrol.com).

## Overview

ccagent is part of the **Claude Control** ecosystem - a platform that enables teams to interact with AI assistants directly from their existing communication channels. 

Instead of context-switching to separate AI tools, team members can mention AI assistants in Slack or Discord and receive intelligent responses with full access to your codebase.

### Supported AI Assistants

- **Claude Code**: Anthropic's official CLI tool for software engineering (default)
- **Cursor**: Popular AI-powered code editor integration

## Installation

### Prerequisites

- Go 1.24 or later
- Git (for repository integration)
- Claude Control account (sign up [here](https://claudecontrol.com))

### From Source

```bash
git clone https://github.com/your-org/ccagent.git
cd ccagent
make build
```

The compiled binary will be available at `bin/ccagent`.

### Environment Setup

ccagent requires the following environment variables:

```bash
# Required: API key from your Claude Control organization
export CCAGENT_API_KEY=your_api_key_here
```

You can generate an API key from the Claude Control dashboard.

## Usage

### Basic Usage

Run ccagent in your development project directory:

```bash
./bin/ccagent
```

By default, ccagent uses Claude Code as the AI assistant with `acceptEdits` permission mode.

### Other Requirements
ccagent must be run from within a Git repository. It will validate the Git environment on startup and exit with an error if requirements aren't met.

### Command Line Options

```bash
./bin/ccagent [OPTIONS]

Options:
  --agent=[claude|cursor]              AI assistant to use (default: claude)
  --claude-bypass-permissions          Use bypassPermissions for Claude (sandbox only)
  --cursor-model=[gpt-5|sonnet-4|sonnet-4-thinking]  Model for Cursor agent
  -v, --version                        Show version information
  -h, --help                           Show help message
```

### Agent-Specific Usage

#### Claude Code Agent (Default)
```bash
# Standard mode - requires approval for file edits
./bin/ccagent --agent claude

# Bypass permissions (Recommended in a secure sandbox environment only)
./bin/ccagent --agent claude --claude-bypass-permissions
```

#### Cursor Agent
```bash
# Use Cursor with specific model
./bin/ccagent --agent cursor --cursor-model sonnet-4
```

### Logging
ccagent automatically creates log files in `~/.config/ccagent/logs/` with timestamp-based naming. Logs are written to both stdout and files for debugging.

## Development

### Building

```bash
# Build for current platform
make build

# Clean build artifacts
make clean
```

### Testing

```bash
# Run all tests
make test

# Run tests with verbose output
make test-verbose
```

### Linting

```bash
# Run linter
make lint

# Auto-fix linting issues
make lint-fix
```

## Security Recommendations

### Permission Modes

ccagent operates in different permission modes depending on the AI assistant and configuration:

#### Secure Mode (Recommended)
- **Claude Code (default)**: Runs in `acceptEdits` mode, requiring explicit approval for all file modifications
- **Best Practice**: Use this mode when running ccagent on your local development machine

#### Bypass Permissions Mode
- **Claude Code with `--claude-bypass-permissions`**: Allows unrestricted system access
- **Cursor Agent**: **Always runs in bypass mode by default**

When running in bypass permissions mode, **anyone with access to your Slack workspace or Discord server can execute arbitrary commands on your system with your user privileges**. It's recommended that you use this mode only if you're running the agent in a secure environment like a docker container or a remote, isolated server.

### Deployment Recommendations

#### Safe Environments for Bypass Mode
Only use bypass permissions mode in these secure environments:
- **Remote servers** with limited access and no sensitive data
- **Docker containers** with restricted capabilities and isolated filesystems
- **Virtual machines** dedicated solely to AI assistance
- **Sandboxed environments** with network and filesystem restrictions

#### Unsafe Environments
**NEVER** use bypass permissions mode on:
- Your primary development machine
- Systems with access to sensitive data, credentials, or production resources
- Shared computers or environments
- Systems connected to corporate networks without proper isolation

### Best Practices

1. **Use Secure Mode by Default**: Always start with Claude Code in standard mode unless you specifically need bypass permissions
2. **Isolate Bypass Deployments**: If you need bypass mode, deploy ccagent in a completely isolated environment
3. **Monitor Access**: Regularly audit who has access to your Slack/Discord channels where AI assistants are active
4. **Limit Repository Scope**: Run ccagent only in repositories that don't contain sensitive information when using bypass mode
5. **Regular Updates**: Keep ccagent updated to receive security patches and improvements

### Agent-Specific Security Considerations

#### Claude Code Agent
- Default secure mode requires manual approval for file changes
- Bypass mode available but should be used with extreme caution
- Explicit warning displayed when bypass mode is enabled

#### Cursor Agent
- **Always operates in bypass mode** - no secure mode available
- Should only be used in isolated, secure environments
- Consider the security implications before using Cursor agent

## Contributing

Fork the repository and open a pull request. Contributions are welcome!

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Support

Contact us at support@claudecontrol.com

