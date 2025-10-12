package handlers

import (
	"sort"
	"time"

	"github.com/gammazero/workerpool"
	"github.com/zishang520/socket.io-client-go/socket"

	"ccagent/core"
	"ccagent/core/log"
	"ccagent/models"
	"ccagent/usecases"
)

// RecoverJobs recovers both in-progress jobs and queued messages after agent restart
// Phase 1: Recovers jobs that were in_progress when the agent restarted
// Phase 2: Recovers messages that were queued but not yet started
// It removes stale items (>24h old) and requeues valid items for processing
func RecoverJobs(
	appState *models.AppState,
	gitUseCase *usecases.GitUseCase,
	socketClient *socket.Socket,
	blockingWorkerPool *workerpool.WorkerPool,
	messageHandler *MessageHandler,
) {
	log.Info("🔄 Starting job and message recovery process")

	allJobs := appState.GetAllJobs()
	allQueuedMessages := appState.GetAllQueuedMessages()

	if len(allJobs) == 0 && len(allQueuedMessages) == 0 {
		log.Info("✅ No jobs or queued messages to recover")
		return
	}

	recoveredJobsCount := 0
	removedJobsCount := 0
	recoveredQueuedCount := 0
	removedQueuedCount := 0
	now := time.Now()

	// Phase 1: Recover in-progress jobs

	for jobID, jobData := range allJobs {
		// Only process jobs that were in_progress
		if jobData.Status != models.JobStatusInProgress {
			continue
		}

		// Check staleness - remove jobs older than 24h
		jobAge := now.Sub(jobData.UpdatedAt)
		if jobAge > 24*time.Hour {
			log.Info("🗑️ Removing stale job %s (age: %v)", jobID, jobAge)
			if err := appState.RemoveJob(jobID); err != nil {
				log.Error("❌ Failed to remove stale job %s: %v", jobID, err)
			} else {
				removedJobsCount++
			}
			continue
		}

		// Validate branch exists
		branchExists, err := gitUseCase.BranchExists(jobData.BranchName)
		if err != nil {
			log.Error("❌ Failed to check if branch %s exists for job %s: %v", jobData.BranchName, jobID, err)
			continue
		}
		if !branchExists {
			log.Warn("⚠️ Branch %s for job %s no longer exists, removing job", jobData.BranchName, jobID)
			if err := appState.RemoveJob(jobID); err != nil {
				log.Error("❌ Failed to remove job with missing branch %s: %v", jobID, err)
			} else {
				removedJobsCount++
			}
			continue
		}

		// Determine message type based on ClaudeSessionID
		var msg models.BaseMessage
		if jobData.ClaudeSessionID == "" {
			// No session = StartConversation (crashed before Claude)
			log.Info("🔄 Recovering StartConversation job %s (age: %v)", jobID, jobAge)
			msg = models.BaseMessage{
				ID:   core.NewID("msg"),
				Type: models.MessageTypeStartConversation,
				Payload: models.StartConversationPayload{
					JobID:              jobID,
					Message:            jobData.LastMessage,
					ProcessedMessageID: jobData.ProcessedMessageID,
					MessageLink:        jobData.MessageLink,
				},
			}
		} else {
			// Has session = UserMessage (crashed during continuation)
			log.Info("🔄 Recovering UserMessage job %s (age: %v)", jobID, jobAge)
			msg = models.BaseMessage{
				ID:   core.NewID("msg"),
				Type: models.MessageTypeUserMessage,
				Payload: models.UserMessagePayload{
					JobID:              jobID,
					Message:            jobData.LastMessage,
					ProcessedMessageID: jobData.ProcessedMessageID,
					MessageLink:        jobData.MessageLink,
				},
			}
		}

		// Submit to blocking worker pool for processing
		blockingWorkerPool.Submit(func() {
			messageHandler.HandleMessage(msg, socketClient)
		})

		recoveredJobsCount++
	}

	// Phase 2: Recover queued messages
	if len(allQueuedMessages) > 0 {
		log.Info("🔄 Recovering %d queued messages", len(allQueuedMessages))

		// Sort queued messages by QueuedAt timestamp (FIFO order)
		sort.Slice(allQueuedMessages, func(i, j int) bool {
			return allQueuedMessages[i].QueuedAt.Before(allQueuedMessages[j].QueuedAt)
		})

		for _, queuedMsg := range allQueuedMessages {
			// Deduplication check: if a job with the same ProcessedMessageID exists, skip
			jobData, exists := appState.GetJobData(queuedMsg.JobID)
			if exists && jobData.ProcessedMessageID == queuedMsg.ProcessedMessageID {
				log.Info("🔄 Queued message %s already being processed/completed, removing from queue", queuedMsg.ProcessedMessageID)
				if err := appState.RemoveQueuedMessage(queuedMsg.ProcessedMessageID); err != nil {
					log.Error("❌ Failed to remove duplicate queued message %s: %v", queuedMsg.ProcessedMessageID, err)
				}
				continue
			}

			// Check staleness - remove messages older than 24h (same as jobs)
			msgAge := now.Sub(queuedMsg.QueuedAt)
			if msgAge > 24*time.Hour {
				log.Info("🗑️ Removing stale queued message %s (age: %v)", queuedMsg.ProcessedMessageID, msgAge)
				if err := appState.RemoveQueuedMessage(queuedMsg.ProcessedMessageID); err != nil {
					log.Error("❌ Failed to remove stale queued message %s: %v", queuedMsg.ProcessedMessageID, err)
				} else {
					removedQueuedCount++
				}
				continue
			}

			// Reconstruct message based on message type
			var msg models.BaseMessage
			if queuedMsg.MessageType == models.MessageTypeStartConversation {
				log.Info("🔄 Recovering queued StartConversation message %s (age: %v)", queuedMsg.ProcessedMessageID, msgAge)
				msg = models.BaseMessage{
					ID:   core.NewID("msg"),
					Type: models.MessageTypeStartConversation,
					Payload: models.StartConversationPayload{
						JobID:              queuedMsg.JobID,
						Message:            queuedMsg.Message,
						ProcessedMessageID: queuedMsg.ProcessedMessageID,
						MessageLink:        queuedMsg.MessageLink,
					},
				}
			} else if queuedMsg.MessageType == models.MessageTypeUserMessage {
				log.Info("🔄 Recovering queued UserMessage %s (age: %v)", queuedMsg.ProcessedMessageID, msgAge)
				msg = models.BaseMessage{
					ID:   core.NewID("msg"),
					Type: models.MessageTypeUserMessage,
					Payload: models.UserMessagePayload{
						JobID:              queuedMsg.JobID,
						Message:            queuedMsg.Message,
						ProcessedMessageID: queuedMsg.ProcessedMessageID,
						MessageLink:        queuedMsg.MessageLink,
					},
				}
			} else {
				log.Warn("⚠️ Unknown message type %s for queued message %s, skipping", queuedMsg.MessageType, queuedMsg.ProcessedMessageID)
				continue
			}

			// Submit to blocking worker pool for processing
			blockingWorkerPool.Submit(func() {
				messageHandler.HandleMessage(msg, socketClient)
			})

			recoveredQueuedCount++
		}
	}

	// Summary
	totalRecovered := recoveredJobsCount + recoveredQueuedCount
	totalRemoved := removedJobsCount + removedQueuedCount
	if totalRecovered > 0 || totalRemoved > 0 {
		log.Info("✅ Recovery complete: %d jobs recovered, %d jobs removed, %d queued recovered, %d queued removed",
			recoveredJobsCount, removedJobsCount, recoveredQueuedCount, removedQueuedCount)
	} else {
		log.Info("✅ No items to recover")
	}
}
