package handlers

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/zishang520/socket.io-client-go/socket"

	"ccagent/core"
	"ccagent/core/log"
	"ccagent/models"
	"ccagent/services"
	"ccagent/usecases"
	"ccagent/utils"
)

type MessageHandler struct {
	claudeService services.CLIAgent
	gitUseCase    *usecases.GitUseCase
	appState      *models.AppState
}

func NewMessageHandler(
	claudeService services.CLIAgent,
	gitUseCase *usecases.GitUseCase,
	appState *models.AppState,
) *MessageHandler {
	return &MessageHandler{
		claudeService: claudeService,
		gitUseCase:    gitUseCase,
		appState:      appState,
	}
}

func (mh *MessageHandler) HandleMessage(msg models.BaseMessage, socketClient *socket.Socket) {
	switch msg.Type {
	case models.MessageTypeStartConversation:
		if err := mh.handleStartConversation(msg, socketClient); err != nil {
			// Extract ProcessedMessageID and JobID from payload for error reporting
			var payload models.StartConversationPayload
			if unmarshalErr := unmarshalPayload(msg.Payload, &payload); unmarshalErr != nil {
				log.Error("Failed to unmarshal StartConversationPayload for error reporting: %v", unmarshalErr)
				return
			}
			if sendErr := mh.sendErrorMessage(socketClient, err, payload.ProcessedMessageID, payload.JobID); sendErr != nil {
				log.Error("Failed to send error message: %v", sendErr)
			}
		}
	case models.MessageTypeUserMessage:
		if err := mh.handleUserMessage(msg, socketClient); err != nil {
			// Extract ProcessedMessageID and JobID from payload for error reporting
			var payload models.UserMessagePayload
			if unmarshalErr := unmarshalPayload(msg.Payload, &payload); unmarshalErr != nil {
				log.Error("Failed to unmarshal UserMessagePayload for error reporting: %v", unmarshalErr)
				return
			}
			if sendErr := mh.sendErrorMessage(socketClient, err, payload.ProcessedMessageID, payload.JobID); sendErr != nil {
				log.Error("Failed to send error message: %v", sendErr)
			}
		}
	case models.MessageTypeCheckIdleJobs:
		if err := mh.handleCheckIdleJobs(msg, socketClient); err != nil {
			log.Info("❌ Error handling CheckIdleJobs message: %v", err)
		}
	default:
		log.Info("⚠️ Unhandled message type: %s", msg.Type)
	}
}

func (mh *MessageHandler) handleStartConversation(msg models.BaseMessage, socketClient *socket.Socket) error {
	log.Info("📋 Starting to handle start conversation message")
	var payload models.StartConversationPayload
	if err := unmarshalPayload(msg.Payload, &payload); err != nil {
		log.Info("❌ Failed to unmarshal start conversation payload: %v", err)
		return fmt.Errorf("failed to unmarshal start conversation payload: %w", err)
	}

	// Send processing message notification that agent is starting to process
	if err := mh.sendProcessingMessage(socketClient, payload.ProcessedMessageID, payload.JobID); err != nil {
		log.Info("❌ Failed to send processing message notification: %v", err)
		return fmt.Errorf("failed to send processing message notification: %w", err)
	}

	log.Info("🚀 Starting new conversation with message: %s", payload.Message)

	// Prepare Git environment for new conversation - FAIL if this doesn't work
	branchName, err := mh.gitUseCase.PrepareForNewConversation(payload.Message)
	if err != nil {
		log.Error("❌ Failed to prepare Git environment: %v", err)
		return fmt.Errorf("failed to prepare Git environment: %w", err)
	}

	// Get appropriate system prompt based on agent type
	systemPrompt := GetClaudeSystemPrompt()
	if mh.claudeService.AgentName() == "cursor" {
		systemPrompt = GetCursorSystemPrompt()
	}

	claudeResult, err := mh.claudeService.StartNewConversationWithSystemPrompt(payload.Message, systemPrompt)
	if err != nil {
		log.Info("❌ Error starting Claude session: %v", err)
		systemErr := mh.sendSystemMessage(
			socketClient,
			fmt.Sprintf("ccagent encountered error: %v", err),
			payload.ProcessedMessageID,
			payload.JobID,
		)
		if systemErr != nil {
			log.Error("❌ Failed to send system message for Claude error: %v", systemErr)
		}
		return fmt.Errorf("error starting Claude session: %w", err)
	}

	// Auto-commit changes if needed
	commitResult, err := mh.gitUseCase.AutoCommitChangesIfNeeded(payload.MessageLink)
	if err != nil {
		log.Info("❌ Auto-commit failed: %v", err)
		return fmt.Errorf("auto-commit failed: %w", err)
	}

	// Update JobData with conversation info (use commitResult.BranchName if available, otherwise branchName)
	finalBranchName := branchName
	if commitResult != nil && commitResult.BranchName != "" {
		finalBranchName = commitResult.BranchName
	}

	// Extract PR ID from commit result if available
	prID := ""
	if commitResult != nil && commitResult.PullRequestID != "" {
		prID = commitResult.PullRequestID
	}

	mh.appState.UpdateJobData(payload.JobID, models.JobData{
		JobID:           payload.JobID,
		BranchName:      finalBranchName,
		ClaudeSessionID: claudeResult.SessionID,
		PullRequestID:   prID,
		UpdatedAt:       time.Now(),
	})

	// Send assistant response back first
	assistantPayload := models.AssistantMessagePayload{
		JobID:              payload.JobID,
		Message:            claudeResult.Output,
		ProcessedMessageID: payload.ProcessedMessageID,
	}

	assistantMsg := models.BaseMessage{
		ID:      core.NewID("msg"),
		Type:    models.MessageTypeAssistantMessage,
		Payload: assistantPayload,
	}
	if err := socketClient.Emit("cc_message", assistantMsg); err != nil {
		log.Info("❌ Failed to send assistant response: %v", err)
		return fmt.Errorf("failed to send assistant response: %w", err)
	}

	log.Info("🤖 Sent assistant response (message ID: %s)", assistantMsg.ID)

	// Add delay to ensure git activity message comes after assistant message
	time.Sleep(200 * time.Millisecond)

	// Send system message after assistant message for git activity
	if err := mh.sendGitActivitySystemMessage(socketClient, commitResult, payload.ProcessedMessageID, payload.JobID); err != nil {
		log.Info("❌ Failed to send git activity system message: %v", err)
		return fmt.Errorf("failed to send git activity system message: %w", err)
	}

	// Validate and restore PR description footer if needed
	if err := mh.gitUseCase.ValidateAndRestorePRDescriptionFooter(payload.MessageLink); err != nil {
		log.Info("❌ Failed to validate PR description footer: %v", err)
		return fmt.Errorf("failed to validate PR description footer: %w", err)
	}

	log.Info("📋 Completed successfully - handled start conversation message")
	return nil
}

func (mh *MessageHandler) handleUserMessage(msg models.BaseMessage, socketClient *socket.Socket) error {
	log.Info("📋 Starting to handle user message")
	var payload models.UserMessagePayload
	if err := unmarshalPayload(msg.Payload, &payload); err != nil {
		log.Info("❌ Failed to unmarshal user message payload: %v", err)
		return fmt.Errorf("failed to unmarshal user message payload: %w", err)
	}

	// Send processing message notification that agent is starting to process
	if err := mh.sendProcessingMessage(socketClient, payload.ProcessedMessageID, payload.JobID); err != nil {
		log.Info("❌ Failed to send processing message notification: %v", err)
		return fmt.Errorf("failed to send processing message notification: %w", err)
	}

	log.Info("💬 Continuing conversation with message: %s", payload.Message)

	// Get the current job data to retrieve the Claude session ID and branch
	jobData, exists := mh.appState.GetJobData(payload.JobID)
	if !exists {
		log.Info("❌ JobID %s not found in AppState", payload.JobID)
		return fmt.Errorf("job %s not found - conversation may have been started elsewhere", payload.JobID)
	}

	sessionID := jobData.ClaudeSessionID
	if sessionID == "" {
		log.Info("❌ No Claude session ID found for job %s", payload.JobID)
		return fmt.Errorf("no active Claude session found for job %s", payload.JobID)
	}

	// Assert that BranchName is never empty
	utils.AssertInvariant(jobData.BranchName != "", "BranchName must not be empty for job "+payload.JobID)

	// Switch to the job's branch before continuing the conversation
	if err := mh.gitUseCase.SwitchToJobBranch(jobData.BranchName); err != nil {
		log.Error("❌ Failed to switch to job branch %s: %v", jobData.BranchName, err)
		return fmt.Errorf("failed to switch to job branch %s: %w", jobData.BranchName, err)
	}
	log.Info("✅ Successfully switched to job branch: %s", jobData.BranchName)

	claudeResult, err := mh.claudeService.ContinueConversation(sessionID, payload.Message)
	if err != nil {
		log.Info("❌ Error continuing Claude session: %v", err)
		systemErr := mh.sendSystemMessage(
			socketClient,
			fmt.Sprintf("ccagent encountered error: %v", err),
			payload.ProcessedMessageID,
			payload.JobID,
		)
		if systemErr != nil {
			log.Error("❌ Failed to send system message for Claude error: %v", systemErr)
		}
		return fmt.Errorf("error continuing Claude session: %w", err)
	}

	// Auto-commit changes if needed
	commitResult, err := mh.gitUseCase.AutoCommitChangesIfNeeded(payload.MessageLink)
	if err != nil {
		log.Info("❌ Auto-commit failed: %v", err)
		return fmt.Errorf("auto-commit failed: %w", err)
	}

	// Update JobData with latest session ID and branch name from commit result
	finalBranchName := jobData.BranchName
	if commitResult != nil && commitResult.BranchName != "" {
		finalBranchName = commitResult.BranchName
	}

	// Extract PR ID from existing job data or commit result
	prID := jobData.PullRequestID
	if commitResult != nil && commitResult.PullRequestID != "" {
		prID = commitResult.PullRequestID
	}

	mh.appState.UpdateJobData(payload.JobID, models.JobData{
		JobID:           payload.JobID,
		BranchName:      finalBranchName,
		ClaudeSessionID: claudeResult.SessionID,
		PullRequestID:   prID,
		UpdatedAt:       time.Now(),
	})

	// Send assistant response back first
	assistantPayload := models.AssistantMessagePayload{
		JobID:              payload.JobID,
		Message:            claudeResult.Output,
		ProcessedMessageID: payload.ProcessedMessageID,
	}

	assistantMsg := models.BaseMessage{
		ID:      core.NewID("msg"),
		Type:    models.MessageTypeAssistantMessage,
		Payload: assistantPayload,
	}
	if err := socketClient.Emit("cc_message", assistantMsg); err != nil {
		log.Info("❌ Failed to send assistant response: %v", err)
		return fmt.Errorf("failed to send assistant response: %w", err)
	}

	log.Info("🤖 Sent assistant response (message ID: %s)", assistantMsg.ID)

	// Add delay to ensure git activity message comes after assistant message
	time.Sleep(200 * time.Millisecond)

	// Send system message after assistant message for git activity
	if err := mh.sendGitActivitySystemMessage(socketClient, commitResult, payload.ProcessedMessageID, payload.JobID); err != nil {
		log.Info("❌ Failed to send git activity system message: %v", err)
		return fmt.Errorf("failed to send git activity system message: %w", err)
	}

	// Validate and restore PR description footer if needed
	if err := mh.gitUseCase.ValidateAndRestorePRDescriptionFooter(payload.MessageLink); err != nil {
		log.Info("❌ Failed to validate PR description footer: %v", err)
		return fmt.Errorf("failed to validate PR description footer: %w", err)
	}

	log.Info("📋 Completed successfully - handled user message")
	return nil
}

func (mh *MessageHandler) handleCheckIdleJobs(msg models.BaseMessage, socketClient *socket.Socket) error {
	log.Info("📋 Starting to handle check idle jobs message")
	var payload models.CheckIdleJobsPayload
	if err := unmarshalPayload(msg.Payload, &payload); err != nil {
		log.Info("❌ Failed to unmarshal check idle jobs payload: %v", err)
		return fmt.Errorf("failed to unmarshal check idle jobs payload: %w", err)
	}

	log.Info("🔍 Checking all assigned jobs for idleness")

	// Get all job data from app state
	allJobData := mh.appState.GetAllJobs()
	if len(allJobData) == 0 {
		log.Info("📋 No jobs assigned to this agent")
		return nil
	}

	log.Info("🔍 Found %d jobs assigned to this agent", len(allJobData))

	// Check each job for idleness
	for jobID, jobData := range allJobData {
		log.Info("🔍 Checking job %s on branch %s", jobID, jobData.BranchName)

		if err := mh.checkJobIdleness(jobID, jobData, socketClient); err != nil {
			log.Info("❌ Failed to check idleness for job %s: %v", jobID, err)
			// Continue checking other jobs even if one fails
			continue
		}
	}

	log.Info("📋 Completed successfully - checked all jobs for idleness")
	return nil
}

func (mh *MessageHandler) checkJobIdleness(jobID string, jobData models.JobData, socketClient *socket.Socket) error {
	log.Info("📋 Starting to check idleness for job %s", jobID)

	var prStatus string
	var err error

	// Use stored PR ID if available, otherwise fall back to branch-based check
	if jobData.PullRequestID != "" {
		log.Info("ℹ️ Using stored PR ID %s for job %s", jobData.PullRequestID, jobID)
		prStatus, err = mh.gitUseCase.CheckPRStatusByID(jobData.PullRequestID)
		if err != nil {
			log.Error("❌ Failed to check PR status by ID %s: %v", jobData.PullRequestID, err)
			return fmt.Errorf("failed to check PR status by ID %s: %w", jobData.PullRequestID, err)
		}
	} else {
		log.Info("ℹ️ No stored PR ID for job %s, using branch-based check", jobID)
		prStatus, err = mh.gitUseCase.CheckPRStatus(jobData.BranchName)
		if err != nil {
			log.Error("❌ Failed to check PR status for branch %s: %v", jobData.BranchName, err)
			return fmt.Errorf("failed to check PR status for branch %s: %w", jobData.BranchName, err)
		}
	}

	var reason string
	var shouldComplete bool

	switch prStatus {
	case "merged":
		reason = "Job complete - Pull request was merged"
		shouldComplete = true
		log.Info("✅ Job %s PR was merged - marking as complete", jobID)
	case "closed":
		reason = "Job complete - Pull request was closed"
		shouldComplete = true
		log.Info("✅ Job %s PR was closed - marking as complete", jobID)
	case "open":
		log.Info("ℹ️ Job %s has open PR - not marking as complete", jobID)
		shouldComplete = false
	case "no_pr":
		log.Info("ℹ️ Job %s has no PR - checking timeout", jobID)
		jobData, exists := mh.appState.GetJobData(jobID)
		if !exists {
			log.Info("❌ Job %s not found in app state - cannot check idleness", jobID)
			return fmt.Errorf("job %s not found in app state", jobID)
		}

		if jobData.UpdatedAt.Add(1 * time.Hour).After(time.Now()) {
			log.Info("ℹ️ Job %s has no PR but is still active - not marking as complete", jobID)
			shouldComplete = false
		} else {
			log.Info("⏰ Job %s has no PR and is idle - marking as complete", jobID)
			reason = "Job complete - Thread is inactive"
			shouldComplete = true
		}
	default:
		log.Info("ℹ️ Job %s PR status unclear (%s) - keeping active", jobID, prStatus)
		shouldComplete = false
	}

	if shouldComplete {
		if err := mh.sendJobCompleteMessage(socketClient, jobID, reason); err != nil {
			log.Error("❌ Failed to send job complete message for job %s: %v", jobID, err)
			return fmt.Errorf("failed to send job complete message: %w", err)
		}

		// Remove job from app state since it's complete
		mh.appState.RemoveJob(jobID)
		log.Info("🗑️ Removed completed job %s from app state", jobID)
	}

	log.Info("📋 Completed successfully - checked idleness for job %s", jobID)
	return nil
}

func (mh *MessageHandler) sendJobCompleteMessage(socketClient *socket.Socket, jobID, reason string) error {
	log.Info("📋 Sending job complete message for job %s with reason: %s", jobID, reason)

	payload := models.JobCompletePayload{
		JobID:  jobID,
		Reason: reason,
	}

	jobMsg := models.BaseMessage{
		ID:      core.NewID("msg"),
		Type:    models.MessageTypeJobComplete,
		Payload: payload,
	}
	if err := socketClient.Emit("cc_message", jobMsg); err != nil {
		log.Info("❌ Failed to send job complete message: %v", err)
		return fmt.Errorf("failed to send job complete message: %w", err)
	}

	log.Info("📤 Sent job complete message for job: %s (message ID: %s)", jobID, jobMsg.ID)

	return nil
}

func (mh *MessageHandler) sendSystemMessage(socketClient *socket.Socket, message, slackMessageID, jobID string) error {
	payload := models.SystemMessagePayload{
		Message:            message,
		ProcessedMessageID: slackMessageID,
		JobID:              jobID,
	}

	sysMsg := models.BaseMessage{
		ID:      core.NewID("msg"),
		Type:    models.MessageTypeSystemMessage,
		Payload: payload,
	}
	if err := socketClient.Emit("cc_message", sysMsg); err != nil {
		log.Info("❌ Failed to send system message: %v", err)
		return fmt.Errorf("failed to send system message: %w", err)
	}

	log.Info("⚙️ Sent system message: %s (message ID: %s)", message, sysMsg.ID)

	return nil
}

// sendErrorMessage sends an error as a system message. The Claude service handles
// all error processing internally, so we just need to format and send the error.
func (mh *MessageHandler) sendErrorMessage(socketClient *socket.Socket, err error, slackMessageID, jobID string) error {
	messageToSend := fmt.Sprintf("ccagent encountered error: %v", err)
	return mh.sendSystemMessage(socketClient, messageToSend, slackMessageID, jobID)
}

func (mh *MessageHandler) sendProcessingMessage(socketClient *socket.Socket, processedMessageID, jobID string) error {
	processingMessageMsg := models.BaseMessage{
		ID:   core.NewID("msg"),
		Type: models.MessageTypeProcessingMessage,
		Payload: models.ProcessingMessagePayload{
			ProcessedMessageID: processedMessageID,
			JobID:              jobID,
		},
	}

	if err := socketClient.Emit("cc_message", processingMessageMsg); err != nil {
		log.Info("❌ Failed to send processing message notification: %v", err)
		return fmt.Errorf("failed to send processing message notification: %w", err)
	}

	log.Info("🔔 Sent processing message notification for message: %s", processedMessageID)
	return nil
}

func extractPRNumber(prURL string) string {
	if prURL == "" {
		return ""
	}

	// Extract PR number from URL like https://github.com/user/repo/pull/1234
	parts := strings.Split(prURL, "/")
	if len(parts) > 0 && parts[len(parts)-1] != "" {
		return "#" + parts[len(parts)-1]
	}

	return ""
}

func (mh *MessageHandler) sendGitActivitySystemMessage(
	socketClient *socket.Socket,
	commitResult *usecases.AutoCommitResult,
	slackMessageID string,
	jobID string,
) error {
	if commitResult == nil {
		return nil
	}

	if commitResult.JustCreatedPR && commitResult.PullRequestLink != "" {
		// New PR created
		message := fmt.Sprintf("Agent opened a [pull request](%s)", commitResult.PullRequestLink)
		if err := mh.sendSystemMessage(socketClient, message, slackMessageID, jobID); err != nil {
			log.Info("❌ Failed to send PR creation system message: %v", err)
			return fmt.Errorf("failed to send PR creation system message: %w", err)
		}
	} else if !commitResult.JustCreatedPR && commitResult.CommitHash != "" && commitResult.RepositoryURL != "" {
		// Commit added to existing PR
		shortHash := commitResult.CommitHash
		if len(shortHash) > 7 {
			shortHash = shortHash[:7]
		}
		commitURL := fmt.Sprintf("%s/commit/%s", commitResult.RepositoryURL, commitResult.CommitHash)
		message := fmt.Sprintf("New commit added: [%s](%s)", shortHash, commitURL)

		// Add PR link if available
		if commitResult.PullRequestLink != "" {
			prNumber := extractPRNumber(commitResult.PullRequestLink)
			if prNumber != "" {
				message += fmt.Sprintf(" in [%s](%s)", prNumber, commitResult.PullRequestLink)
			}
		}

		if err := mh.sendSystemMessage(socketClient, message, slackMessageID, jobID); err != nil {
			log.Info("❌ Failed to send commit system message: %v", err)
			return fmt.Errorf("failed to send commit system message: %w", err)
		}
	}

	return nil
}

func unmarshalPayload(payload any, target any) error {
	if payload == nil {
		return nil
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return json.Unmarshal(payloadBytes, target)
}
