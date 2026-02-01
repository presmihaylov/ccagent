package handlers

import (
	"encoding/json"
	"sync"

	"github.com/gammazero/workerpool"

	"eksecd/core/log"
	"eksecd/models"
)

// JobDispatcher routes messages to per-job channels to ensure sequential processing
// for the same job while allowing different jobs to process in parallel.
type JobDispatcher struct {
	activeJobs map[string]chan models.BaseMessage
	mutex      sync.Mutex
	handler    *MessageHandler
	workerPool *workerpool.WorkerPool
	appState   *models.AppState
}

// NewJobDispatcher creates a new JobDispatcher instance
func NewJobDispatcher(
	handler *MessageHandler,
	workerPool *workerpool.WorkerPool,
	appState *models.AppState,
) *JobDispatcher {
	return &JobDispatcher{
		activeJobs: make(map[string]chan models.BaseMessage),
		handler:    handler,
		workerPool: workerPool,
		appState:   appState,
	}
}

// Dispatch routes a message to the appropriate per-job channel.
// If no channel exists for the job, it creates one and starts a worker goroutine.
func (d *JobDispatcher) Dispatch(msg models.BaseMessage) {
	jobID := d.extractJobID(msg)
	if jobID == "" {
		// No job ID - process directly via worker pool (e.g., CheckIdleJobs)
		d.workerPool.Submit(func() {
			d.handler.HandleMessage(msg)
		})
		return
	}

	// Check/create channel under lock, but don't call Submit while holding lock
	// (Submit can block if worker pool is full, causing mutex contention)
	d.mutex.Lock()
	ch, exists := d.activeJobs[jobID]
	if !exists {
		ch = make(chan models.BaseMessage, 100)
		d.activeJobs[jobID] = ch
		log.Info("🔀 Created per-job channel for job %s", jobID)
	}
	d.mutex.Unlock()

	// Submit worker outside of lock to avoid blocking other dispatchers
	if !exists {
		d.workerPool.Submit(func() {
			d.processJobMessages(jobID, ch)
		})
	}

	// Send message to the job's channel (non-blocking since buffer is 100)
	select {
	case ch <- msg:
		log.Info("📥 Queued message to job %s channel", jobID)
	default:
		log.Error("❌ Job %s channel is full, dropping message", jobID)
	}
}

// processJobMessages processes messages sequentially for a specific job
func (d *JobDispatcher) processJobMessages(jobID string, ch chan models.BaseMessage) {
	log.Info("🔄 Started message processor for job %s", jobID)

	// Ensure channel is cleaned up when we exit
	defer d.cleanup(jobID)

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				// Channel closed externally
				log.Info("📤 Message processor for job %s exited (channel closed)", jobID)
				return
			}

			log.Info("🔧 Processing message for job %s", jobID)
			d.handler.HandleMessage(msg)

			// Check if job was removed from AppState
			jobData, exists := d.appState.GetJobData(jobID)
			if !exists {
				log.Info("✅ Job %s removed, exiting processor", jobID)
				return
			}

			// If job is completed AND no more messages buffered, exit
			// This ensures we process all queued messages before exiting
			if jobData.Status == models.JobStatusCompleted && len(ch) == 0 {
				log.Info("✅ Job %s completed and channel empty, exiting processor", jobID)
				return
			}
		}
	}
}

// cleanup removes a job's channel from the activeJobs map
func (d *JobDispatcher) cleanup(jobID string) {
	d.mutex.Lock()
	ch, exists := d.activeJobs[jobID]
	if exists {
		// Remove from map first to prevent new messages being sent
		delete(d.activeJobs, jobID)
	}
	d.mutex.Unlock()

	if exists {
		// Close channel outside of mutex to avoid holding it while draining
		close(ch)
		log.Info("🧹 Cleaned up channel for job %s", jobID)
	}
}

// extractJobID extracts the job ID from a message based on its type
func (d *JobDispatcher) extractJobID(msg models.BaseMessage) string {
	switch msg.Type {
	case models.MessageTypeStartConversation:
		var payload models.StartConversationPayload
		if err := d.unmarshalPayload(msg.Payload, &payload); err != nil {
			log.Error("❌ Failed to unmarshal StartConversation payload: %v", err)
			return ""
		}
		return payload.JobID

	case models.MessageTypeUserMessage:
		var payload models.UserMessagePayload
		if err := d.unmarshalPayload(msg.Payload, &payload); err != nil {
			log.Error("❌ Failed to unmarshal UserMessage payload: %v", err)
			return ""
		}
		return payload.JobID

	default:
		// Other message types (CheckIdleJobs, RefreshToken) don't have job IDs
		return ""
	}
}

// unmarshalPayload unmarshals a message payload into the target struct
func (d *JobDispatcher) unmarshalPayload(payload any, target any) error {
	if payload == nil {
		return nil
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return json.Unmarshal(payloadBytes, target)
}
