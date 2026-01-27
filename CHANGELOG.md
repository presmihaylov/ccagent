## [v0.0.50] - 2026-01-27

### Features

- Add X-AGENT-ID header to artifacts API calls ([#106](https://github.com/presmihaylov/ccagent/pull/106))
  - Includes agent identification in artifact API requests
  - Enables server-side tracking of which agent uploaded artifacts
  - Improves observability for multi-agent deployments

### Bugfixes

- Fix OpenCode directory permissions for agentrunner user ([#105](https://github.com/presmihaylov/ccagent/pull/105))
  - Fixes MCP config directory creation with proper ownership for non-root users
  - Ensures permissions and rules processors use correct user paths
  - Resolves directory permission errors in managed execution mode

## [v0.0.48] - 2026-01-24

### Features

- Add concurrent job processing with git worktrees ([#83](https://github.com/presmihaylov/ccagent/pull/83))
  - Enables agents to process multiple jobs simultaneously using isolated git worktrees
  - Each concurrent job runs in its own worktree with separate branches
  - Improves throughput for repositories with multiple pending tasks
  - Maintains isolation between concurrent job executions

## [v0.0.47] - 2026-01-21

### Bugfixes

- Fix parsing failure for large MCP tool results ([#82](https://github.com/presmihaylov/ccagent/pull/82))
  - Adds handler for large `text` fields in `tool_use_result` arrays
  - MCP tools (like `mcp__postgres__query`) return results in a different format than regular tools
  - Truncates text fields over 100KB to prevent bufio.Scanner "token too long" errors
  - Fixes parsing failures when postgres queries return large result sets (100MB+)

## [v0.0.46] - 2026-01-21

### Bugfixes

- Set umask 002 when spawning agent processes in managed mode ([#81](https://github.com/presmihaylov/ccagent/pull/81))
  - Wraps agent commands in bash with umask 002 for group-writable file permissions
  - Enables ccagent to delete files created by agent during git clean operations
  - Fixes "Permission denied" errors on git operations with agentrunner-created files

## [v0.0.45] - 2026-01-20

### Bugfixes

- Fix: write .claude.json as target user via sudo ([#80](https://github.com/presmihaylov/ccagent/pull/80))
  - Writes .claude.json configuration file with proper ownership when running as non-root
  - Uses sudo to ensure file is created with target user permissions
  - Fixes permission issues when deploying MCP server configurations

## [v0.0.44] - 2026-01-20

### Bugfixes

- Fix deploy artifacts to agent user's home directory ([#79](https://github.com/presmihaylov/ccagent/pull/79))
  - Deploys MCP servers, rules, permissions, and skills to the agent user's home directory
  - Ensures proper file ownership and permissions for agent processes
  - Improves reliability when running agents as non-root users

## [v0.0.43] - 2026-01-20

### Features

- Add process isolation support for agent execution ([#77](https://github.com/presmihaylov/ccagent/pull/77))
  - Enables process isolation for running multiple agent instances
  - Provides better resource isolation and security boundaries
  - Supports isolated execution environments for agent processes
  - Adds comprehensive test coverage for process isolation

### Bugfixes

- Fix extractSessionID to handle non-JSON output before session data ([#76](https://github.com/presmihaylov/ccagent/pull/76))
  - Properly handles Claude Code output that contains non-JSON content before session data
  - Improves parsing reliability when output includes warnings or other text
  - Adds test coverage for edge cases in session ID extraction

## [v0.0.42] - 2026-01-16

### Bugfixes

- Fix checkout remote branch on container redeploy ([#73](https://github.com/presmihaylov/ccagent/pull/73))
  - Properly checks out the remote branch when containers are redeployed
  - Ensures agents start on the correct branch after container restart
  - Improves reliability for container orchestration workflows

- Fix parsing failure for large tool_result outputs ([#75](https://github.com/presmihaylov/ccagent/pull/75))
  - Resolves parsing issues when tool results contain very large outputs
  - Improves handling of buffer sizes for tool result processing
  - Enhances stability for operations with verbose tool outputs

## [v0.0.41] - 2026-01-11

### Bugfixes

- Increase buffer size to 4MB for handling large tool outputs
  - Fixes issues with processing large responses from Claude Code
  - Prevents buffer overflow errors during heavy tool usage
  - Improves reliability for complex operations with verbose output

- Handle detached HEAD state in GetCurrentBranch ([#72](https://github.com/presmihaylov/ccagent/pull/72))
  - Properly handles repositories in detached HEAD state
  - Prevents errors when working with specific commits instead of branches
  - Improves robustness of branch detection logic

## [v0.0.40] - 2026-01-08

### Features

- Add --repo flag to decouple repo from PWD ([#71](https://github.com/presmihaylov/ccagent/pull/71))
  - Enables specifying repository path via --repo flag
  - Decouples repository location from current working directory
  - Improves flexibility for running agents from any directory
  - Useful for scripts and automation that manage multiple repositories

## [v0.0.39] - 2026-01-07

### Features

- Add X-AGENT-ID header with environment variable support ([#70](https://github.com/presmihaylov/ccagent/pull/70))
  - Adds X-AGENT-ID header to API requests for agent identification
  - Supports CCAGENT_AGENT_ID environment variable for custom agent IDs
  - Improves agent tracing and debugging capabilities

### Bugfixes

- Extract results from tool_use messages when no text response ([#68](https://github.com/presmihaylov/ccagent/pull/68))
  - Fixes handling of API responses that contain only tool_use blocks
  - Properly extracts results from tool_use message content
  - Improves reliability of agent response processing

- Simplify PR title prompts for smaller model compatibility ([#69](https://github.com/presmihaylov/ccagent/pull/69))
  - Streamlines PR title generation prompts for better compatibility
  - Improves support for smaller language models
  - Reduces prompt complexity while maintaining quality

- Handle empty repository on startup gracefully ([#67](https://github.com/presmihaylov/ccagent/pull/67))
  - Fixes crash when starting agent on empty repositories
  - Adds graceful handling of repositories without commits
  - Improves agent startup reliability

## [v0.0.38] - 2026-01-01

### Features

- Add permissions processor to enable yolo mode for OpenCode ([#66](https://github.com/presmihaylov/ccagent/pull/66))
  - Adds new permissions processor to enable yolo mode for OpenCode client
  - Allows OpenCode agents to run with fewer confirmation prompts
  - Improves agent autonomy and workflow efficiency

## [v0.0.37] - 2026-01-01

### Features

- Add skills support for coding agents ([#65](https://github.com/presmihaylov/ccagent/pull/65))
  - Enables skills loading from repository configuration
  - Supports custom skill definitions for enhanced agent capabilities
  - Allows agents to utilize specialized skills during conversations
  - Improves agent flexibility and extensibility for various use cases

## [v0.0.36] - 2025-12-28

### Bugfixes

- Transform MCP configs for OpenCode compatibility ([#64](https://github.com/presmihaylov/ccagent/pull/64))
  - Fixes MCP server configuration handling for OpenCode client
  - Transforms MCP config format to be compatible with OpenCode
  - Ensures proper MCP server integration across both Claude Code and OpenCode clients

## [v0.0.35] - 2025-12-28

### Features

- Add MCP server configuration support ([#63](https://github.com/presmihaylov/ccagent/pull/63))
  - Enables configuration of MCP (Model Context Protocol) servers for agents
  - Supports defining custom MCP servers in repository configuration
  - Allows agents to interact with external tools and data sources via MCP
  - Includes comprehensive test coverage for MCP processor

## [v0.0.34] - 2025-12-28

### Improvements

- Store Claude Code rules in ~/.claude/rules ([#62](https://github.com/presmihaylov/ccagent/pull/62))
  - Moves rule storage location to ~/.claude/rules directory
  - Aligns with Claude Code's standard rules location
  - Improves compatibility with Claude Code's rules management

## [v0.0.33] - 2025-12-28

### Improvements

- Simplify OpenCode rules and add cleanup ([#61](https://github.com/presmihaylov/ccagent/pull/61))
  - Streamlines OpenCode rules processing for better maintainability
  - Adds cleanup functionality for temporary rule files
  - Improves code organization and reduces complexity

## [v0.0.32] - 2025-12-27

### Features

- Add agent-specific rules processing ([#60](https://github.com/presmihaylov/ccagent/pull/60))
  - Enables processing of agent-specific CLAUDE.md rules from repository
  - Supports custom agent behavior configuration per repository
  - Allows repository owners to define agent-specific instructions and constraints
  - Enhances flexibility for repository-level agent customization

## [v0.0.31] - 2025-12-27

### Features

- Add agent artifacts API support ([#59](https://github.com/presmihaylov/ccagent/pull/59))
  - Enables agents to upload and manage artifacts via API
  - Supports storing and retrieving files generated during agent sessions
  - Provides foundation for artifact sharing between agents and users

### Improvements

- Increase job inactivity timeout to 25h ([#58](https://github.com/presmihaylov/ccagent/pull/58))
  - Extends job inactivity timeout from previous limit to 25 hours
  - Prevents premature job termination for long-running tasks
  - Improves reliability for complex, time-consuming operations

## [v0.0.30] - 2025-12-26

### Features

- Add model flag support for Claude agent ([#57](https://github.com/presmihaylov/ccagent/pull/57))
  - Enables model selection via --model flag for Claude client
  - Allows specifying different Claude models (e.g., claude-sonnet-4-5-20250514)
  - Provides flexibility in choosing model based on task requirements

## [v0.0.29] - 2025-12-25

### Bugfixes

- Handle non-JSON opencode output as raw error ([#56](https://github.com/presmihaylov/ccagent/pull/56))
  - Properly handles error responses from OpenCode that aren't valid JSON
  - Returns raw output as error message for better debugging
  - Improves reliability when working with OpenCode client

## [v0.0.28] - 2025-12-24

### Improvements

- Consolidate model flags into --model ([#53](https://github.com/presmihaylov/ccagent/pull/53))
  - Simplifies CLI by replacing multiple model flags with a single --model flag
  - Improves developer experience with cleaner command syntax
  - Reduces flag complexity for model selection

## [v0.0.27] - 2025-12-24

### Features

- Add support for OpenCode client ([#48](https://github.com/presmihaylov/ccagent/pull/48))
  - Integrates OpenCode as a new supported AI coding client
  - Expands agent compatibility with additional coding assistants
  - Provides seamless integration for OpenCode users

- Add automatic PR title trimming ([#49](https://github.com/presmihaylov/ccagent/pull/49))
  - Automatically trims PR titles that exceed GitHub's character limit
  - Prevents PR creation failures due to overly long titles
  - Improves reliability of automated PR workflows

- Show the correct platform in PR description footer ([#51](https://github.com/presmihaylov/ccagent/pull/51))
  - Displays the actual platform (Slack, Discord) in PR footers
  - Improves traceability of PR origins
  - Enhances multi-platform integration clarity

- Skip token operations for self-hosted ([#52](https://github.com/presmihaylov/ccagent/pull/52))
  - Skips unnecessary token operations in self-hosted deployments
  - Reduces overhead for self-managed installations
  - Improves startup performance for self-hosted agents

### Bugfixes

- Increase API client timeout to 60s ([#50](https://github.com/presmihaylov/ccagent/pull/50))
  - Extends API client timeout from default to 60 seconds
  - Prevents timeout errors during slow API responses
  - Improves reliability for complex operations

## [v0.0.26] - 2025-12-19

### Bugfixes

- Abandon job when remote branch deleted ([#47](https://github.com/presmihaylov/ccagent/pull/47))
  - Automatically detects when a remote branch has been deleted
  - Gracefully abandons jobs that can no longer be completed
  - Prevents agents from getting stuck on deleted branches
  - Improves resource utilization by freeing up workers promptly

## [v0.0.25] - 2025-12-02

### Improvements

- Increase response limit and add context guidelines ([#45](https://github.com/presmihaylov/ccagent/pull/45))
  - Expands response limits for more detailed agent outputs
  - Adds context guidelines for improved response quality
  - Enhances user experience with more comprehensive answers

- Reduce system prompt char limit to 800 ([#46](https://github.com/presmihaylov/ccagent/pull/46))
  - Optimizes system prompt length for better performance
  - Reduces token overhead while maintaining functionality
  - Improves efficiency of agent initialization

## [v0.0.24] - 2025-11-30

### Features

- Add ask/execute mode to control file edits ([#43](https://github.com/presmihaylov/ccagent/pull/43))
  - Introduces ask/execute mode for controlled file editing operations
  - Allows users to review and approve file changes before they are applied
  - Enhances safety and control over agent-initiated file modifications

## [v0.0.23] - 2025-11-29

### Bugfixes

- Return message instead of error on no response ([#42](https://github.com/presmihaylov/ccagent/pull/42))
  - Improves handling of cases where Claude returns no response
  - Returns informative message instead of throwing error
  - Enhances robustness for edge cases in conversation handling

## [v0.0.22] - 2025-11-29

### Features

- Add PR template support to descriptions ([#41](https://github.com/presmihaylov/ccagent/pull/41))
  - Supports custom PR description templates for enhanced pull request workflows
  - Enables teams to standardize PR formatting and content
  - Improves consistency across repository contributions

## [v0.0.21] - 2025-11-16

### Bugfixes

- Fix: Collect all assistant messages in conversation response ([#38](https://github.com/presmihaylov/ccagent/pull/38))
  - Ensures all assistant messages are properly collected in multi-turn conversations
  - Fixes message loss issues in conversation responses
  - Improves reliability of conversation handling

## [v0.0.20] - 2025-11-08

### Features

- Add support for codex ([#37](https://github.com/presmihaylov/ccagent/pull/37))
  - Integrates codex functionality for enhanced code analysis
  - Expands agent capabilities with advanced code understanding
  - Improves code-related task performance

### Improvements

- Add skill-creator from anthropics/skills
  - Includes skill-creator skill for creating custom skills
  - Enables users to extend agent capabilities
  - Provides guided workflow for skill development

## [v0.0.19] - 2025-10-29

### Bugfixes

- Always sync token to environment manager ([#34](https://github.com/presmihaylov/ccagent/pull/34))
  - Ensures OAuth tokens are always synchronized to the environment manager
  - Fixes token sync inconsistencies that could cause authentication failures
  - Improves reliability of token management across agent lifecycle
  - Enhances stability for long-running agent instances

## [v0.0.18] - 2025-10-29

### Features

- Refresh tokens before conversations ([#33](https://github.com/presmihaylov/ccagent/pull/33))
  - Ensures OAuth tokens are refreshed before starting new conversations
  - Prevents mid-conversation authentication failures
  - Improves reliability for long-running agents
  - Enhances user experience with seamless authentication

### Improvements

- Decouple token monitoring from socketio retry ([#32](https://github.com/presmihaylov/ccagent/pull/32))
  - Separates token refresh logic from WebSocket connection management
  - Improves system reliability and error handling
  - Reduces coupling between authentication and communication layers
  - Enhances maintainability of token monitoring logic

## [v0.0.17] - 2025-10-29

### Features

- Add token management with auto-refresh ([#31](https://github.com/presmihaylov/ccagent/pull/31))
  - Implements automatic OAuth token refreshing
  - Improves authentication reliability
  - Reduces manual token management overhead
  - Enhances long-running agent stability

## [v0.0.16] - 2025-10-28

### Features

- Add thread context support for conversations ([#30](https://github.com/presmihaylov/ccagent/pull/30))
  - Implements thread context tracking for multi-turn conversations
  - Improves conversation continuity and context management
  - Enhances agent's ability to maintain conversation state
  - Enables better handling of complex, multi-message interactions

## [v0.0.15] - 2025-10-23

### Features

- Add CCAGENT_CONFIG_DIR environment variable ([#28](https://github.com/presmihaylov/ccagent/pull/28))
  - Allows custom configuration directory path
  - Improves deployment flexibility
  - Enables better multi-instance management

### Bugfixes

- Fix parsing of claude responses with large images
  - Resolves issues with handling large image attachments
  - Improves response parsing stability
  - Enhances reliability for image-heavy workflows
- Reduce Socket.IO reconnect max backoff to 10s ([#29](https://github.com/presmihaylov/ccagent/pull/29))
  - Faster reconnection during network issues
  - Reduces downtime during connectivity problems
  - Improves overall agent responsiveness

## [0.0.14] - 2025-10-16

### Features

- Add attachment support with magic bytes ([#26](https://github.com/presmihaylov/ccagent/pull/26))
  - Implements automatic file type detection using magic bytes
  - Supports attachments in agent communication
  - Enhances file handling capabilities

### Bugfix

- Prevent job recovery on socket reconnect ([#27](https://github.com/presmihaylov/ccagent/pull/27))
  - Fixes duplicate job recovery attempts during reconnection
  - Ensures clean reconnection without state conflicts
  - Improves stability during network interruptions

## [0.0.13] - 2025-10-14

### Improvements

- Extend job inactivity timeout to 24 hours ([#23](https://github.com/presmihaylov/ccagent/pull/23))
  - Jobs now remain active for 24 hours instead of 1 hour
  - Prevents premature job termination for long-running tasks
  - Improves reliability for extended coding sessions
- Prevent reconnect blocking by persisting worker pools ([#24](https://github.com/presmihaylov/ccagent/pull/24))
  - Worker pools now persist across socket reconnections
  - Eliminates blocking during reconnection events
  - Ensures continuous operation without interruption

## [0.0.12] - 2025-10-12

### Features

- Add message queue for reliable reconnection ([#22](https://github.com/presmihaylov/ccagent/pull/22))
  - Implements message queue to prevent message loss during reconnection
  - Ensures reliable message delivery with automatic retry mechanism
  - Dramatically improves agent stability and reliability during network interruptions

### Documentation

- Add comprehensive release process documentation
  - Detailed release guide in docs/release_ccagent.md
  - Step-by-step instructions for version bumping and changelog updates
  - Release notes template with emoji formatting examples
  - Troubleshooting section and complete workflow documentation

## [0.0.11] - 2025-10-12

### Features

- Add persistent state with job restoration ([#20](https://github.com/presmihaylov/ccagent/pull/20))
  - Implements state persistence across agent restarts
  - Automatic job restoration on startup
  - Enhanced recovery handling for interrupted tasks
- Add startup logging for config and environment ([#19](https://github.com/presmihaylov/ccagent/pull/19))
  - Improved visibility into agent configuration at startup
  - Environment variable logging for debugging
- Support custom release notes in build script
  - Build script now accepts custom release notes from `/tmp/release_notes.md`

### Documentation

- Add Claude Control context to prompts ([#18](https://github.com/presmihaylov/ccagent/pull/18))
  - Enhanced prompt templates with Claude Control-specific context

## [0.0.3] - 2025-08-22

### Bugfix

- Set the env variables in program env when reloading

## [0.0.2] - 2025-08-17

### Documentation

- Add project overview and architecture guide ([#1](https://github.com/your-org/ccagent/issues/1))

### Refactor

- Improve session context and clean Git methods ([#2](https://github.com/your-org/ccagent/issues/2))

## [0.0.1] - 2025-08-12

### Features

- Testing
- Generate PR titles with git-cliff conventions out of the box
- Add homebrew installation
- Initial ccagent release

### Miscellaneous Tasks

- Fix release script
- Update readme
- Update readme

<!-- generated by git-cliff -->
