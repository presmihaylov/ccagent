package handlers

import (
	"fmt"
	"time"

	"github.com/gammazero/workerpool"
	"github.com/zishang520/socket.io-client-go/socket"

	"ccagent/core"
	"ccagent/core/log"
	"ccagent/models"
	"ccagent/usecases"
)

// RecoverInProgressJobs recovers jobs that were in_progress when the agent restarted
// It removes stale jobs (>24h old) and requeues valid jobs for processing
func RecoverInProgressJobs(
	appState *models.AppState,
	gitUseCase *usecases.GitUseCase,
	socketClient *socket.Socket,
	blockingWorkerPool *workerpool.WorkerPool,
	messageHandler *MessageHandler,
) {
	log.Info("🔄 Starting job recovery process")

	allJobs := appState.GetAllJobs()
	if len(allJobs) == 0 {
		log.Info("✅ No jobs to recover")
		return
	}

	recoveredCount := 0
	removedCount := 0
	now := time.Now()

	for jobID, jobData := range allJobs {
		// Only process jobs that were in_progress
		if jobData.Status != models.JobStatusInProgress {
			continue
		}

		// Check staleness - remove jobs older than 24h
		fmt.Println("DEBUG_2", now)
		fmt.Println("DEBUG_3", jobData.UpdatedAt)

		jobAge := now.Sub(jobData.UpdatedAt)
		if jobAge > 24*time.Hour {
			log.Info("🗑️ Removing stale job %s (age: %v)", jobID, jobAge)
			if err := appState.RemoveJob(jobID); err != nil {
				log.Error("❌ Failed to remove stale job %s: %v", jobID, err)
			} else {
				removedCount++
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
				removedCount++
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

		recoveredCount++
	}

	if recoveredCount > 0 || removedCount > 0 {
		log.Info("✅ Job recovery complete: %d recovered, %d removed", recoveredCount, removedCount)
	} else {
		log.Info("✅ No in_progress jobs to recover")
	}
}
