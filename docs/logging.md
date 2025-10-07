# Logging Best Practices

## Overview

The ccagent logging system is built on Go's `slog` package and provides both traditional format-string logging and structured logging with key-value pairs.

## Logging Functions

### Traditional Format-String Logging

```go
log.Info("Starting operation for user %s", userID)
log.Debug("Processing item %d of %d", i, total)
log.Warn("Rate limit approaching: %d/%d", current, limit)
log.Error("Failed to connect: %v", err)
```

### Structured Logging (Recommended)

Structured logging provides better queryability and parsing:

```go
log.InfoWith("Starting operation", "user_id", userID, "operation", "sync")
log.DebugWith("Processing item", "index", i, "total", total)
log.WarnWith("Rate limit approaching", "current", current, "limit", limit)
log.ErrorWith("Failed to connect", "error", err, "host", hostname, "port", port)
```

## Performance Tracking

Use timers to track operation duration:

```go
func processData(jobID string) error {
    timer := log.StartTimer("process_data")
    defer timer.LogElapsed("job_id", jobID)

    // ... your operation here ...

    return nil
}

// Custom message with timing
func complexOperation() error {
    timer := log.StartTimer("complex_operation")

    // ... do work ...

    timer.LogElapsedWith("✅ Complex operation completed successfully",
        "records_processed", count,
        "errors", errorCount)

    return nil
}
```

The timer will automatically log:
- `operation`: The name of the operation
- `elapsed_ms`: Duration in milliseconds
- Any additional attributes you provide

## Context Propagation

Add consistent context to multiple log statements:

```go
logWithContext := log.WithContext("job_id", jobID, "session_id", sessionID)

logWithContext("Starting task")
// ... do work ...
logWithContext("Task completed", "records", count)
```

## Best Practices

### 1. Always Include Context in Errors

```go
// ❌ Bad
log.Error("Failed to commit: %v", err)

// ✅ Good
log.ErrorWith("Failed to commit", "error", err, "branch", branchName, "session_id", sessionID)
```

### 2. Use Timers for Long Operations

```go
func AutoCommitChangesIfNeeded(link, sessionID string) error {
    timer := log.StartTimer("auto_commit")
    defer timer.LogElapsed("session_id", sessionID)

    // ... implementation ...
}
```

### 3. Log Key Milestones

```go
log.InfoWith("🚀 Starting new conversation",
    "job_id", jobID,
    "message_length", len(message))

log.InfoWith("✅ Conversation completed",
    "job_id", jobID,
    "branch", branchName,
    "session_id", sessionID)
```

### 4. Include Relevant IDs

Always include identifiers that help trace operations:
- `job_id`: For tracking conversations
- `session_id`: For AI agent sessions
- `branch`: For git operations
- `message_id`: For message routing
- `agent_id`: For multi-agent scenarios

### 5. Use Consistent Emoji Prefixes

- 📋 - General operations
- 🚀 - Starting something
- ✅ - Success
- ❌ - Errors
- ⚠️ - Warnings
- 🔄 - Syncing/refreshing
- 📨 - Message handling
- 🔌 - Connection events
- ⏱️ - Performance/timing
- 🤖 - AI agent operations
- 🌿 - Git operations
- 💓 - Health checks

## Example: Full Function with Good Logging

```go
func (mh *MessageHandler) handleStartConversation(msg models.BaseMessage, client *socket.Socket) error {
    // Track overall performance
    timer := log.StartTimer("start_conversation")
    defer timer.LogElapsed()

    // Log with structured context
    log.InfoWith("📋 Starting to handle start conversation message", "message_id", msg.ID)

    var payload models.StartConversationPayload
    if err := unmarshalPayload(msg.Payload, &payload); err != nil {
        log.ErrorWith("❌ Failed to unmarshal payload", "error", err, "message_id", msg.ID)
        return fmt.Errorf("failed to unmarshal payload: %w", err)
    }

    log.InfoWith("🚀 Starting new conversation",
        "job_id", payload.JobID,
        "message_length", len(payload.Message))

    // ... more operations ...

    log.InfoWith("✅ Completed start conversation successfully",
        "job_id", payload.JobID,
        "branch", branchName,
        "session_id", sessionID)

    return nil
}
```

## Log Output Format

Structured logs are output in the format:
```
time=2025-10-07T13:45:23.123-07:00 level=INFO msg="📋 Starting operation" job_id=ccajob_abc123 session_id=sess_xyz operation=start_conversation elapsed_ms=1234
```

This format is:
- Easy to read for humans
- Simple to parse with tools
- Queryable with log aggregation systems
