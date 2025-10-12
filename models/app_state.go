package models

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// JobStatus represents the current state of a job
type JobStatus string

const (
	JobStatusInProgress JobStatus = "in_progress"
	JobStatusCompleted  JobStatus = "completed"
)

// JobData tracks the state of a specific job/conversation
type JobData struct {
	JobID              string    `json:"job_id"`
	BranchName         string    `json:"branch_name"`
	ClaudeSessionID    string    `json:"claude_session_id"`
	PullRequestID      string    `json:"pull_request_id"`      // GitHub PR number (e.g., "123") - empty if no PR created yet
	LastMessage        string    `json:"last_message"`         // The last message sent to Claude for this job
	ProcessedMessageID string    `json:"processed_message_id"` // ID of the chat platform message being processed
	MessageLink        string    `json:"message_link"`         // Link to the original chat message
	Status             JobStatus `json:"status"`               // Current status of the job: "in_progress" or "completed"
	UpdatedAt          time.Time `json:"updated_at"`
}

// QueuedMessage represents a message that has been queued for processing but not yet started
type QueuedMessage struct {
	ProcessedMessageID string    `json:"processed_message_id"` // Unique identifier per chat message
	JobID              string    `json:"job_id"`               // Which conversation this belongs to
	MessageType        string    `json:"message_type"`         // "start_conversation_v1" or "user_message_v1"
	Message            string    `json:"message"`              // User's message text
	MessageLink        string    `json:"message_link"`         // Link to original chat message
	QueuedAt           time.Time `json:"queued_at"`            // When queued (for ordering)
}

// PersistedState represents the state that gets persisted to disk
type PersistedState struct {
	AgentID        string                    `json:"agent_id"`
	Jobs           map[string]*JobData       `json:"jobs"`
	QueuedMessages map[string]*QueuedMessage `json:"queued_messages"` // Key: ProcessedMessageID
}

// LoadedState represents the result of loading persisted state from disk
type LoadedState struct {
	AgentID        string
	Jobs           map[string]*JobData
	QueuedMessages map[string]*QueuedMessage
	Loaded         bool // Indicates whether state was successfully loaded from disk
}

// AppState manages the state of all active jobs
type AppState struct {
	agentID        string
	jobs           map[string]*JobData
	queuedMessages map[string]*QueuedMessage
	statePath      string
	mutex          sync.RWMutex
}

// NewAppState creates a new AppState instance
func NewAppState(agentID string, statePath string) *AppState {
	return &AppState{
		agentID:        agentID,
		jobs:           make(map[string]*JobData),
		queuedMessages: make(map[string]*QueuedMessage),
		statePath:      statePath,
	}
}

// GetAgentID returns the agent ID
func (a *AppState) GetAgentID() string {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.agentID
}

// UpdateJobData updates or creates job data for a given JobID
func (a *AppState) UpdateJobData(jobID string, data JobData) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.jobs[jobID] = &data

	// Persist state after updating
	if err := a.persistStateLocked(); err != nil {
		return fmt.Errorf("failed to persist state: %w", err)
	}

	return nil
}

// GetJobData retrieves job data for a given JobID
func (a *AppState) GetJobData(jobID string) (*JobData, bool) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	data, exists := a.jobs[jobID]
	if !exists {
		return nil, false
	}
	// Return a copy to avoid race conditions
	return &JobData{
		JobID:              data.JobID,
		BranchName:         data.BranchName,
		ClaudeSessionID:    data.ClaudeSessionID,
		PullRequestID:      data.PullRequestID,
		LastMessage:        data.LastMessage,
		ProcessedMessageID: data.ProcessedMessageID,
		MessageLink:        data.MessageLink,
		Status:             data.Status,
		UpdatedAt:          data.UpdatedAt,
	}, true
}

// RemoveJob removes job data for a given JobID
func (a *AppState) RemoveJob(jobID string) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	delete(a.jobs, jobID)

	// Persist state after removing
	if err := a.persistStateLocked(); err != nil {
		return fmt.Errorf("failed to persist state: %w", err)
	}

	return nil
}

// GetAllJobs returns a copy of all job data
func (a *AppState) GetAllJobs() map[string]JobData {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	result := make(map[string]JobData)
	for jobID, data := range a.jobs {
		result[jobID] = JobData{
			JobID:              data.JobID,
			BranchName:         data.BranchName,
			ClaudeSessionID:    data.ClaudeSessionID,
			PullRequestID:      data.PullRequestID,
			LastMessage:        data.LastMessage,
			ProcessedMessageID: data.ProcessedMessageID,
			MessageLink:        data.MessageLink,
			Status:             data.Status,
			UpdatedAt:          data.UpdatedAt,
		}
	}
	return result
}

// AddQueuedMessage adds a queued message to the state and persists it
func (a *AppState) AddQueuedMessage(msg QueuedMessage) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.queuedMessages[msg.ProcessedMessageID] = &msg

	// Persist state after adding
	if err := a.persistStateLocked(); err != nil {
		return fmt.Errorf("failed to persist state: %w", err)
	}

	return nil
}

// RemoveQueuedMessage removes a queued message from the state and persists
func (a *AppState) RemoveQueuedMessage(processedMessageID string) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	delete(a.queuedMessages, processedMessageID)

	// Persist state after removing
	if err := a.persistStateLocked(); err != nil {
		return fmt.Errorf("failed to persist state: %w", err)
	}

	return nil
}

// GetAllQueuedMessages returns a copy of all queued messages
func (a *AppState) GetAllQueuedMessages() []QueuedMessage {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	result := make([]QueuedMessage, 0, len(a.queuedMessages))
	for _, msg := range a.queuedMessages {
		result = append(result, QueuedMessage{
			ProcessedMessageID: msg.ProcessedMessageID,
			JobID:              msg.JobID,
			MessageType:        msg.MessageType,
			Message:            msg.Message,
			MessageLink:        msg.MessageLink,
			QueuedAt:           msg.QueuedAt,
		})
	}
	return result
}

// persistStateLocked persists the current state to disk
// MUST be called with mutex already locked
func (a *AppState) persistStateLocked() error {
	if a.statePath == "" {
		return fmt.Errorf("state path not configured")
	}

	// Create the state object
	state := PersistedState{
		AgentID:        a.agentID,
		Jobs:           a.jobs,
		QueuedMessages: a.queuedMessages,
	}

	// Marshal to JSON with pretty printing
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(a.statePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Write to temporary file first
	tempPath := a.statePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp state file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, a.statePath); err != nil {
		return fmt.Errorf("failed to rename state file: %w", err)
	}

	return nil
}

// LoadState loads persisted state from disk
// Returns LoadedState containing the loaded data and a boolean indicating success, or an error
func LoadState(statePath string) (*LoadedState, error) {
	// Check if state file exists
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		return &LoadedState{
			AgentID:        "",
			Jobs:           nil,
			QueuedMessages: nil,
			Loaded:         false,
		}, nil
	}

	// Read the state file
	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	// Unmarshal the state
	var state PersistedState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &LoadedState{
		AgentID:        state.AgentID,
		Jobs:           state.Jobs,
		QueuedMessages: state.QueuedMessages,
		Loaded:         true,
	}, nil
}
