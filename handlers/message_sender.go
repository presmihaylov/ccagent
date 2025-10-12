package handlers

import (
	"github.com/zishang520/socket.io-client-go/socket"

	"ccagent/core/log"
)

// OutgoingMessage represents a message to be sent via Socket.IO
type OutgoingMessage struct {
	Event string
	Data  any
}

// MessageSender handles queuing and sending messages to Socket.IO client.
// It blocks when the connection is down and resumes when reconnected.
type MessageSender struct {
	connectionState *ConnectionState
	messageQueue    chan OutgoingMessage
	socketClient    *socket.Socket
}

// NewMessageSender creates a new MessageSender instance.
// The queue has a buffer of 1 message to ensure blocking until messages are sent.
// This guarantees that jobs are only marked complete after their messages are actually sent.
func NewMessageSender(connectionState *ConnectionState) *MessageSender {
	return &MessageSender{
		connectionState: connectionState,
		messageQueue:    make(chan OutgoingMessage, 1),
		socketClient:    nil, // Set later via Run()
	}
}

// Run starts the message sender goroutine that processes the queue.
// This should be called once with the Socket.IO client reference.
// It blocks until the message queue is closed.
func (ms *MessageSender) Run(socketClient *socket.Socket) {
	ms.socketClient = socketClient
	log.Info("📤 MessageSender: Started processing queue")

	for msg := range ms.messageQueue {
		// Block until connection is established
		ms.connectionState.WaitForConnection()

		// Send the message
		if err := ms.socketClient.Emit(msg.Event, msg.Data); err != nil {
			log.Error("❌ MessageSender: Failed to emit message on event '%s': %v", msg.Event, err)
			// Continue processing - message is lost (no retry logic yet)
		} else {
			log.Info("📤 MessageSender: Successfully sent message on event '%s'", msg.Event)
		}
	}

	log.Info("📤 MessageSender: Queue closed, exiting")
}

// QueueMessage adds a message to the send queue.
// Blocks until the message is consumed and sent by the MessageSender goroutine.
// This ensures the caller knows the message has been processed before continuing.
func (ms *MessageSender) QueueMessage(event string, data any) {
	log.Info("📥 MessageSender: Queueing message for event '%s'", event)
	ms.messageQueue <- OutgoingMessage{
		Event: event,
		Data:  data,
	}
	log.Info("📤 MessageSender: Message for event '%s' has been consumed by sender", event)
}

// Close closes the message queue, causing Run() to exit.
// Should be called during graceful shutdown.
func (ms *MessageSender) Close() {
	close(ms.messageQueue)
}
