# ccagent

A Go-based CLI agent that connects AI assistants (Claude Code, Cursor) to team collaboration platforms like Slack and Discord through the [Claude Control platform](https://claudecontrol.com).

### Supported AI Assistants

- **Claude Code**: Anthropic's official CLI tool for software engineering (default)
- **Cursor**: Popular AI-powered code editor integration
- **Codex**: OpenAI's coding assistant with model selection
- **OpenCode**: Open-source AI coding agent with multi-provider model support

### Supported Platforms

ccagent runs on **macOS**, **Linux**, and **Windows** with native binaries for both Intel and ARM architectures.

## Installation

### Via Homebrew (Recommended)

```bash
brew install presmihaylov/taps/ccagent
```

To upgrade to the latest version:
```bash
brew upgrade presmihaylov/taps/ccagent
```

### From Source
You will need to have Go 1.24 installed:
```bash
git clone https://github.com/presmihaylov/ccagent.git
cd ccagent
make build
```

The compiled binary will be available at `bin/ccagent`.

## Usage

### Prerequisites

- Git
- GitHub CLI (`gh`) - [Install here](https://cli.github.com/)
- Claude Control account (sign up [here](https://claudecontrol.com))

### Basic Usage

#### Repository Setup

**Important**: ccagent will autonomously create branches, make changes, and create pull requests. To avoid conflicts with your main development workflow, it's **strongly recommended** to clone your repository separately for ccagent use.

#### Prerequisites Setup

Before running ccagent, ensure you have:

1. **GitHub CLI Authentication**: ccagent uses the GitHub CLI to create pull requests
   ```bash
   # Login to GitHub
   gh auth login
   
   # Verify authentication
   gh auth status
   ```

2. **Repository Access**: If using SSH, ensure your key is loaded
   ```bash
   ssh-add ~/.ssh/id_rsa
   ```

#### GitHub Account Options

You can use ccagent with:
- **Your personal GitHub account**: ccagent will create PRs on your behalf
- **Dedicated bot account**: Create a separate GitHub account for ccagent to use (recommended for teams)

#### Environment Setup

ccagent requires the following environment variables:

```bash
# Required: API key from your Claude Control organization
export CCAGENT_API_KEY=your_api_key_here
```

You can generate an API key from the Claude Control dashboard.

#### Running ccagent

Once setup is complete, run ccagent in your repository directory.

By default, ccagent uses Claude Code as the AI assistant with `acceptEdits` permission mode.

### Command Line Options

```bash
ccagent [OPTIONS]

Options:
  --agent=[claude|cursor|codex|opencode]  AI assistant to use (default: claude)
  --claude-bypass-permissions             Use bypassPermissions for Claude/Codex (sandbox only)
  --model=MODEL                           Model to use (agent-specific, see examples below)
  -v, --version                           Show version information
  -h, --help                              Show help message
```

### Agent-Specific Usage

#### Claude Code Agent (Default)
```bash
# Standard mode - requires approval for file edits
ccagent --agent claude

# Use specific model (options: sonnet, haiku, opus, or full model names like claude-sonnet-4-5-20250929)
ccagent --agent claude --model haiku

# Bypass permissions (Recommended in a secure sandbox environment only)
ccagent --agent claude --claude-bypass-permissions
```

#### Cursor Agent
```bash
# Use Cursor with specific model (options: gpt-5, sonnet-4, sonnet-4-thinking)
ccagent --agent cursor --model sonnet-4
```

#### Codex Agent
```bash
# Standard mode - requires approval for file edits
ccagent --agent codex

# Bypass permissions (Recommended in a secure sandbox environment only)
ccagent --agent codex --claude-bypass-permissions

# Use specific model (default: gpt-5, accepts any model string)
ccagent --agent codex --model gpt-5
```

#### OpenCode Agent
```bash
# OpenCode requires bypass permissions mode (default model: opencode/grok-code)
ccagent --agent opencode --claude-bypass-permissions

# Use specific provider/model (format: provider/model)
ccagent --agent opencode --claude-bypass-permissions --model anthropic/claude-3-5-sonnet
```

**Note**: OpenCode only supports `bypassPermissions` mode. The `--claude-bypass-permissions` flag is required.

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

ccagent operates in different permission modes depending on the AI assistant and configuration:

### Secure Mode (Recommended)
- **Claude Code (default)**: Runs in `acceptEdits` mode, requiring explicit approval for all file modifications
- **Codex (default)**: Runs in `acceptEdits` mode with sandbox protections
- **Best Practice**: Use this mode when running ccagent on your local development machine

### Bypass Permissions Mode
- **Claude Code with `--claude-bypass-permissions`**: Allows unrestricted system access
- **Codex with `--claude-bypass-permissions`**: Bypasses approvals and sandbox
- **Cursor Agent**: **Always runs in bypass mode by default**
- **OpenCode Agent**: **Only supports bypass mode**

When running in bypass permissions mode, **anyone with access to your Slack workspace or Discord server can execute arbitrary commands on your system with your user privileges**. It's recommended that you use this mode only if you're running the agent in a secure environment like a docker container or a remote, isolated server.

## Contributing

Fork the repository and open a pull request. Contributions are welcome!

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Support

Contact us at support@claudecontrol.com

---

*Monkeys in the trees*
*Swinging through the canopy*
*Code flows just like them*
