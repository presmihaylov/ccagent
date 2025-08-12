package models

import (
	"sync"
	"time"
)

// JobData tracks the state of a specific job/conversation
type JobData struct {
	JobID           string
	BranchName      string
	ClaudeSessionID string
	PullRequestID   string // GitHub PR number (e.g., "123") - empty if no PR created yet
	UpdatedAt       time.Time
}

// AppState manages the state of all active jobs
type AppState struct {
	jobs  map[string]*JobData
	mutex sync.RWMutex
}

// NewAppState creates a new AppState instance
func NewAppState() *AppState {
	return &AppState{
		jobs: make(map[string]*JobData),
	}
}

// UpdateJobData updates or creates job data for a given JobID
func (a *AppState) UpdateJobData(jobID string, data JobData) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.jobs[jobID] = &data
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
		JobID:           data.JobID,
		BranchName:      data.BranchName,
		ClaudeSessionID: data.ClaudeSessionID,
		PullRequestID:   data.PullRequestID,
		UpdatedAt:       data.UpdatedAt,
	}, true
}

// RemoveJob removes job data for a given JobID
func (a *AppState) RemoveJob(jobID string) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	delete(a.jobs, jobID)
}

// GetAllJobs returns a copy of all job data
func (a *AppState) GetAllJobs() map[string]JobData {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	result := make(map[string]JobData)
	for jobID, data := range a.jobs {
		result[jobID] = JobData{
			JobID:           data.JobID,
			BranchName:      data.BranchName,
			ClaudeSessionID: data.ClaudeSessionID,
			PullRequestID:   data.PullRequestID,
		}
	}
	return result
}
