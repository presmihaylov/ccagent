package models

// Message types
const (
	MessageTypeStartConversation         = "start_conversation_v1"
	MessageTypeStartConversationResponse = "start_conversation_response_v1"
	MessageTypeUserMessage               = "user_message_v1"
	MessageTypeAssistantMessage          = "assistant_message_v1"
	MessageTypeSystemMessage             = "system_message_v1"
	MessageTypeProcessingMessage         = "processing_message_v1"
	MessageTypeCheckIdleJobs             = "check_idle_jobs_v1"
	MessageTypeJobComplete               = "job_complete_v1"
)

type BaseMessage struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

type Attachment struct {
	Content        string `json:"content"`          // base64-encoded content
	AttachmentType string `json:"attachment_type"`  // "image" or "other"
}

type StartConversationPayload struct {
	JobID              string       `json:"job_id"`
	Message            string       `json:"message"`
	ProcessedMessageID string       `json:"processed_message_id"`
	MessageLink        string       `json:"message_link"`
	Attachments        []Attachment `json:"attachments,omitempty"`
}

type StartConversationResponsePayload struct {
	SessionID string `json:"sessionID"`
	Message   string `json:"message"`
}

type UserMessagePayload struct {
	JobID              string       `json:"job_id"`
	Message            string       `json:"message"`
	ProcessedMessageID string       `json:"processed_message_id"`
	MessageLink        string       `json:"message_link"`
	Attachments        []Attachment `json:"attachments,omitempty"`
}

type AssistantMessagePayload struct {
	JobID              string `json:"job_id"`
	Message            string `json:"message"`
	ProcessedMessageID string `json:"processed_message_id"`
}

type SystemMessagePayload struct {
	Message            string `json:"message"`
	ProcessedMessageID string `json:"processed_message_id"`
	JobID              string `json:"job_id"`
}

type ProcessingMessagePayload struct {
	ProcessedMessageID string `json:"processed_message_id"`
	JobID              string `json:"job_id"`
}

type CheckIdleJobsPayload struct {
	// Empty payload - agent checks all its jobs
}

type JobCompletePayload struct {
	JobID  string `json:"job_id"`
	Reason string `json:"reason"`
}
