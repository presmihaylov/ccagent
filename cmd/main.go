package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/gammazero/workerpool"
	"github.com/jessevdk/go-flags"
	"github.com/zishang520/engine.io-client-go/transports"
	"github.com/zishang520/engine.io/v2/types"
	"github.com/zishang520/socket.io-client-go/socket"

	"ccagent/clients"
	claudeclient "ccagent/clients/claude"
	cursorclient "ccagent/clients/cursor"
	"ccagent/core"
	"ccagent/core/log"
	"ccagent/handlers"
	"ccagent/models"
	"ccagent/services"
	claudeservice "ccagent/services/claude"
	cursorservice "ccagent/services/cursor"
	"ccagent/usecases"
	"ccagent/utils"
)

type CmdRunner struct {
	messageHandler *handlers.MessageHandler
	gitUseCase     *usecases.GitUseCase
	logFilePath    string
	agentID        string
	reconnectChan  chan struct{}
}

func NewCmdRunner(agentType, permissionMode, cursorModel string) (*CmdRunner, error) {
	log.Info("📋 Starting to initialize CmdRunner with agent: %s", agentType)

	// Create log directory for agent service
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	logDir := filepath.Join(homeDir, ".config", "ccagent", "logs")

	// Create the appropriate CLI agent service
	cliAgent, err := createCLIAgent(agentType, permissionMode, cursorModel, logDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create CLI agent: %w", err)
	}

	// Cleanup old session logs (older than 7 days)
	err = cliAgent.CleanupOldLogs(7)
	if err != nil {
		log.Error("Warning: Failed to cleanup old session logs: %v", err)
		// Don't exit - this is not critical for agent operation
	}

	gitClient := clients.NewGitClient()
	appState := models.NewAppState()
	gitUseCase := usecases.NewGitUseCase(gitClient, cliAgent, appState)
	messageHandler := handlers.NewMessageHandler(cliAgent, gitUseCase, appState)

	agentID := core.NewID("ccaid")
	log.Info("🆔 Using persistent agent ID: %s", agentID)

	log.Info("📋 Completed successfully - initialized CmdRunner with %s agent", agentType)
	return &CmdRunner{
		messageHandler: messageHandler,
		gitUseCase:     gitUseCase,
		agentID:        agentID,
		reconnectChan:  make(chan struct{}, 1),
	}, nil
}

// createCLIAgent creates the appropriate CLI agent based on the agent type
func createCLIAgent(agentType, permissionMode, cursorModel, logDir string) (services.CLIAgent, error) {
	switch agentType {
	case "claude":
		claudeClient := claudeclient.NewClaudeClient(permissionMode)
		return claudeservice.NewClaudeService(claudeClient, logDir), nil
	case "cursor":
		cursorClient := cursorclient.NewCursorClient()
		return cursorservice.NewCursorService(cursorClient, logDir, cursorModel), nil
	default:
		return nil, fmt.Errorf("unsupported agent type: %s", agentType)
	}
}

type Options struct {
	//nolint
	Agent string `long:"agent" description:"CLI agent to use (claude or cursor)" choice:"claude" choice:"cursor" default:"claude"`
	BypassPermissions bool `long:"claude-bypass-permissions" description:"Use bypassPermissions mode for Claude (only applies when --agent=claude) (WARNING: Only use in controlled sandbox environments)"`
	CursorModel string `long:"cursor-model" description:"Model to use with Cursor agent (only applies when --agent=cursor)" choice:"gpt-5" choice:"sonnet-4" choice:"sonnet-4-thinking"`
}

func main() {
	var opts Options
	parser := flags.NewParser(&opts, flags.Default)

	_, err := parser.Parse()
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Always enable info level logging
	log.SetLevel(slog.LevelInfo)

	// Acquire directory lock to prevent multiple instances in same directory
	dirLock, err := utils.NewDirLock()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory lock: %v\n", err)
		os.Exit(1)
	}

	if err := dirLock.TryLock(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Ensure lock is released on program exit
	defer func() {
		if unlockErr := dirLock.Unlock(); unlockErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to release directory lock: %v\n", unlockErr)
		}
	}()

	// Validate CCAGENT_API_KEY environment variable
	ccagentAPIKey := os.Getenv("CCAGENT_API_KEY")
	if ccagentAPIKey == "" {
		fmt.Fprintf(os.Stderr, "Error: CCAGENT_API_KEY environment variable is required but not set\n")
		os.Exit(1)
	}

	// Determine permission mode based on flag
	permissionMode := "acceptEdits"
	if opts.BypassPermissions {
		permissionMode = "bypassPermissions"
		fmt.Fprintf(
			os.Stderr,
			"Warning: --claude-bypass-permissions flag should only be used in a controlled, sandbox environment. Otherwise, anyone from Slack will have access to your entire system\n",
		)
	}

	cmdRunner, err := NewCmdRunner(opts.Agent, permissionMode, opts.CursorModel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing CmdRunner: %v\n", err)
		os.Exit(1)
	}

	// Setup program-wide logging from start
	logFilePath, err := cmdRunner.setupProgramLogging()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error setting up program logging: %v\n", err)
		os.Exit(1)
	}
	cmdRunner.logFilePath = logFilePath

	// Validate Git environment before starting
	err = cmdRunner.gitUseCase.ValidateGitEnvironment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Git environment validation failed: %v\n", err)
		os.Exit(1)
	}

	// Cleanup stale ccagent branches
	err = cmdRunner.gitUseCase.CleanupStaleBranches()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to cleanup stale branches: %v\n", err)
		// Don't exit - this is not critical for agent operation
	}

	// Get WebSocket URL from environment variable with default fallback
	wsURL := os.Getenv("CCAGENT_WS_API_URL")
	if wsURL == "" {
		wsURL = "https://claudecontrol.onrender.com/socketio/"
	}

	// Set up deferred cleanup
	defer func() {
		fmt.Fprintf(
			os.Stderr,
			"\n📝 App execution finished, logs for this session are stored in %s\n",
			cmdRunner.logFilePath,
		)
	}()

	// Start Socket.IO client
	err = cmdRunner.startSocketIOClient(wsURL, ccagentAPIKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting WebSocket client: %v\n", err)
		os.Exit(1)
	}
}

func (cr *CmdRunner) startSocketIOClient(serverURLStr, apiKey string) error {
	log.Info("📋 Starting to connect to Socket.IO server at %s", serverURLStr)

	// Set up global interrupt handling
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Set up Socket.IO client options
	opts := socket.DefaultOptions()
	opts.SetTransports(types.NewSet(transports.Polling, transports.WebSocket))

	// Set authentication headers
	opts.SetExtraHeaders(map[string][]string{
		"X-CCAGENT-API-KEY": {apiKey},
		"X-CCAGENT-ID":      {cr.agentID},
	})

	manager := socket.NewManager(serverURLStr, opts)
	socketClient := manager.Socket("/", opts)

	// Initialize dual worker pools
	// Blocking worker pool: 1 worker for sequential conversation processing
	blockingWorkerPool := workerpool.New(1)
	defer blockingWorkerPool.StopWait()

	// Instant worker pool: 5 workers for parallel PR status checking
	instantWorkerPool := workerpool.New(5)
	defer instantWorkerPool.StopWait()

	// Track connection state for auth failure detection
	connected := make(chan bool, 1)
	disconnected := make(chan string, 1)

	// Connection event handlers
	err := socketClient.On("connect", func(args ...any) {
		log.Info("✅ Connected to Socket.IO server, socket ID: %s", socketClient.Id())
		connected <- true
	})
	utils.AssertInvariant(err == nil, fmt.Sprintf("Failed to set up connect handler: %v", err))

	err = socketClient.On("connect_error", func(args ...any) {
		log.Error("❌ Socket.IO connection error: %v", args)
		os.Exit(1)
	})
	utils.AssertInvariant(err == nil, fmt.Sprintf("Failed to set up connect_error handler: %v", err))

	err = socketClient.On("disconnect", func(args ...any) {
		log.Info("🔌 Socket.IO disconnected: %v", args)

		// Send disconnect reason to the channel
		reason := "unknown"

		select {
		case disconnected <- reason:
		default:
			// Channel full, ignore
		}
	})
	utils.AssertInvariant(err == nil, fmt.Sprintf("Failed to set up disconnect handler: %v", err))

	// Set up message handler for cc_message event
	err = socketClient.On("cc_message", func(data ...any) {
		if len(data) == 0 {
			log.Info("❌ No data received for cc_message event")
			return
		}

		var msg models.BaseMessage
		msgBytes, err := json.Marshal(data[0])
		if err != nil {
			log.Info("❌ Failed to marshal message data: %v", err)
			return
		}

		err = json.Unmarshal(msgBytes, &msg)
		if err != nil {
			log.Info("❌ Failed to unmarshal message data: %v", err)
			return
		}

		log.Info("📨 Received message type: %s", msg.Type)

		// Route messages to appropriate worker pool
		switch msg.Type {
		case models.MessageTypeStartConversation, models.MessageTypeUserMessage:
			// Conversation messages need sequential processing
			blockingWorkerPool.Submit(func() {
				cr.messageHandler.HandleMessage(msg, socketClient)
			})
		case models.MessageTypeCheckIdleJobs:
			// PR status checks can run in parallel without blocking conversations
			instantWorkerPool.Submit(func() {
				cr.messageHandler.HandleMessage(msg, socketClient)
			})
		default:
			// Fallback to blocking pool for any unhandled message types
			blockingWorkerPool.Submit(func() {
				cr.messageHandler.HandleMessage(msg, socketClient)
			})
		}
	})
	utils.AssertInvariant(err == nil, fmt.Sprintf("Failed to set up cc_message handler: %v", err))

	// Built-in reconnection handlers
	err = manager.On("reconnect", func(...any) {
		log.Info("✅ Reconnected to Socket.IO server")
	})
	utils.AssertInvariant(err == nil, fmt.Sprintf("Failed to set up reconnect handler: %v", err))

	err = manager.On("reconnect_error", func(errs ...any) {
		log.Info("❌ Socket.IO reconnection error: %v", errs)
	})
	utils.AssertInvariant(err == nil, fmt.Sprintf("Failed to set up reconnect_error handler: %v", err))

	err = manager.On("reconnect_failed", func(errs ...any) {
		log.Info("❌ Socket.IO reconnection failed: %v", errs)
	})
	utils.AssertInvariant(err == nil, fmt.Sprintf("Failed to set up reconnect_failed handler: %v", err))

	// Wait for initial connection or detect auth failure
	go func() {
		// Wait up to 10 seconds for initial connection
		select {
		case <-connected:
			log.Info("✅ Successfully authenticated with Socket.IO server")
		case <-time.After(10 * time.Second):
			log.Error("❌ Connection timeout - server may have rejected authentication. Please check your CCAGENT_API_KEY environment variable")
			os.Exit(1)
		}

		// After successful connection, watch for immediate disconnection
		select {
		case reason := <-disconnected:
			log.Error("❌ Disconnected immediately after connection for unknown reason: %s", reason)
			os.Exit(1)
		case <-time.After(5 * time.Second):
			// No immediate disconnection - connection appears stable
			log.Info("✅ Connection appears stable, continuing normal operation")
		}
	}()

	// Start ping routine once connected
	pingCtx, pingCancel := context.WithCancel(context.Background())
	defer pingCancel()
	cr.startPingRoutine(pingCtx, socketClient)

	// Wait for interrupt signal
	<-interrupt
	log.Info("🔌 Interrupt received, closing Socket.IO connection...")

	socketClient.Disconnect()
	return nil
}

func (cr *CmdRunner) setupProgramLogging() (string, error) {
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Create ~/.config/ccagent/logs directory if it doesn't exist
	logsDir := filepath.Join(homeDir, ".config", "ccagent", "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Create log file with timestamp only
	timestamp := time.Now().Format("20060102-150405")
	logFileName := fmt.Sprintf("%s.log", timestamp)
	logFilePath := filepath.Join(logsDir, logFileName)

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to create log file: %w", err)
	}

	// Always write to both stdout and file
	writer := io.MultiWriter(os.Stdout, logFile)
	log.SetWriter(writer)

	return logFilePath, nil
}

func (cr *CmdRunner) startPingRoutine(ctx context.Context, socketClient *socket.Socket) {
	log.Info("📋 Starting ping routine")
	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Info("📋 Ping routine stopped")
				return
			case <-ticker.C:
				log.Info("💓 Sending ping to server")
				if err := socketClient.Emit("ping"); err != nil {
					log.Error("❌ Failed to send ping: %v", err)
				}
			}
		}
	}()
}
