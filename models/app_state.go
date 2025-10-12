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
	PullRequestID      string    `json:"pull_request_id"` // GitHub PR number (e.g., "123") - empty if no PR created yet
	LastMessage        string    `json:"last_message"`    // The last message sent to Claude for this job
	ProcessedMessageID string    `json:"processed_message_id"` // ID of the chat platform message being processed
	MessageLink        string    `json:"message_link"`    // Link to the original chat message
	Status             JobStatus `json:"status"`          // Current status of the job: "in_progress" or "completed"
	UpdatedAt          time.Time `json:"updated_at"`
}

// PersistedState represents the state that gets persisted to disk
type PersistedState struct {
	AgentID string              `json:"agent_id"`
	Jobs    map[string]*JobData `json:"jobs"`
}

// AppState manages the state of all active jobs
type AppState struct {
	agentID   string
	jobs      map[string]*JobData
	statePath string
	mutex     sync.RWMutex
}

// NewAppState creates a new AppState instance
func NewAppState(agentID string, statePath string) *AppState {
	return &AppState{
		agentID:   agentID,
		jobs:      make(map[string]*JobData),
		statePath: statePath,
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

// persistStateLocked persists the current state to disk
// MUST be called with mutex already locked
func (a *AppState) persistStateLocked() error {
	if a.statePath == "" {
		return fmt.Errorf("state path not configured")
	}

	// Create the state object
	state := PersistedState{
		AgentID: a.agentID,
		Jobs:    a.jobs,
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
// Returns the agent ID and whether state was loaded successfully
func LoadState(statePath string) (agentID string, jobs map[string]*JobData, loaded bool, err error) {
	// Check if state file exists
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		return "", nil, false, nil
	}

	// Read the state file
	data, err := os.ReadFile(statePath)
	if err != nil {
		return "", nil, false, fmt.Errorf("failed to read state file: %w", err)
	}

	// Unmarshal the state
	var state PersistedState
	if err := json.Unmarshal(data, &state); err != nil {
		return "", nil, false, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return state.AgentID, state.Jobs, true, nil
}
