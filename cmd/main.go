package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/gammazero/workerpool"
	"github.com/jessevdk/go-flags"
	"github.com/zishang520/engine.io-client-go/transports"
	"github.com/zishang520/engine.io/v2/types"
	"github.com/zishang520/socket.io-client-go/socket"

	"ccagent/clients"
	claudeclient "ccagent/clients/claude"
	cursorclient "ccagent/clients/cursor"
	"ccagent/core"
	"ccagent/core/env"
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
	messageHandler     *handlers.MessageHandler
	messageSender      *handlers.MessageSender
	connectionState    *handlers.ConnectionState
	gitUseCase         *usecases.GitUseCase
	appState           *models.AppState
	rotatingWriter     *log.RotatingWriter
	envManager         *env.EnvManager
	agentID            string
	agentsApiClient    *clients.AgentsApiClient
	wsURL              string
	ccagentAPIKey      string

	// Persistent worker pools reused across reconnects
	blockingWorkerPool *workerpool.WorkerPool
	instantWorkerPool  *workerpool.WorkerPool
}

// fetchAndSetToken fetches the token from API and sets it as environment variable
func fetchAndSetToken(agentsApiClient *clients.AgentsApiClient, envManager *env.EnvManager) error {
	log.Info("🔑 Fetching Anthropic token from API...")

	tokenResp, err := agentsApiClient.FetchToken()
	if err != nil {
		return fmt.Errorf("failed to fetch token: %w", err)
	}

	// Set the token as environment variable using EnvManager
	if err := envManager.Set(tokenResp.EnvKey, tokenResp.Token); err != nil {
		return fmt.Errorf("failed to set environment variable %s: %w", tokenResp.EnvKey, err)
	}

	log.Info("✅ Successfully fetched and set token (env key: %s, expires: %s)",
		tokenResp.EnvKey, tokenResp.ExpiresAt.Format(time.RFC3339))

	return nil
}

func NewCmdRunner(agentType, permissionMode, cursorModel string) (*CmdRunner, error) {
	log.Info("📋 Starting to initialize CmdRunner with agent: %s", agentType)

	// Create log directory for agent service
	configDir, err := env.GetConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}
	logDir := filepath.Join(configDir, "logs")

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

	// Initialize environment manager
	envManager, err := env.NewEnvManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create environment manager: %w", err)
	}

	// Start periodic refresh every 1 minute
	envManager.StartPeriodicRefresh(1 * time.Minute)

	// Get API key and WS URL for agents API client
	ccagentAPIKey := envManager.Get("CCAGENT_API_KEY")
	if ccagentAPIKey == "" {
		return nil, fmt.Errorf("CCAGENT_API_KEY environment variable is required but not set")
	}

	wsURL := envManager.Get("CCAGENT_WS_API_URL")
	if wsURL == "" {
		wsURL = "https://claudecontrol.onrender.com/socketio/"
	}

	// Extract base URL for API client (remove /socketio/ suffix)
	apiBaseURL := strings.TrimSuffix(wsURL, "/socketio/")
	agentsApiClient := clients.NewAgentsApiClient(ccagentAPIKey, apiBaseURL)
	log.Info("🔗 Configured agents API client with base URL: %s", apiBaseURL)

	// Fetch and set Anthropic token BEFORE initializing anything else
	if err := fetchAndSetToken(agentsApiClient, envManager); err != nil {
		return nil, fmt.Errorf("failed to fetch and set token: %w", err)
	}

	gitClient := clients.NewGitClient()

	// Determine state file path
	statePath := filepath.Join(configDir, "state.json")

	// Restore app state from persisted data
	appState, agentID, err := handlers.RestoreAppState(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to restore app state: %w", err)
	}

	// Initialize ConnectionState and MessageSender
	connectionState := handlers.NewConnectionState()
	messageSender := handlers.NewMessageSender(connectionState)

	gitUseCase := usecases.NewGitUseCase(gitClient, cliAgent, appState)

	messageHandler := handlers.NewMessageHandler(cliAgent, gitUseCase, appState, envManager, messageSender, agentsApiClient)

	// Create the CmdRunner instance
	cr := &CmdRunner{
		messageHandler:   messageHandler,
		messageSender:    messageSender,
		connectionState:  connectionState,
		gitUseCase:       gitUseCase,
		appState:         appState,
		envManager:       envManager,
		agentID:          agentID,
		agentsApiClient:  agentsApiClient,
		wsURL:            wsURL,
		ccagentAPIKey:    ccagentAPIKey,
	}

	// Initialize dual worker pools that persist for the app lifetime
	cr.blockingWorkerPool = workerpool.New(1) // sequential conversation processing
	cr.instantWorkerPool = workerpool.New(5)  // parallel PR status checks

	// Register GitHub token update hook
	envManager.RegisterReloadHook(gitUseCase.GithubTokenUpdateHook)
	log.Info("📎 Registered GitHub token update hook")

	// Recover in-progress jobs and queued messages on program startup (NOT on Socket.io reconnect)
	// This enables crash recovery - we only want to recover jobs once when the program starts
	handlers.RecoverJobs(
		appState,
		gitUseCase,
		cr.blockingWorkerPool,
		messageHandler,
	)

	log.Info("📋 Completed successfully - initialized CmdRunner with %s agent", agentType)
	return cr, nil
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
	Agent             string `long:"agent" description:"CLI agent to use (claude or cursor)" choice:"claude" choice:"cursor" default:"claude"`
	BypassPermissions bool   `long:"claude-bypass-permissions" description:"Use bypassPermissions mode for Claude (only applies when --agent=claude) (WARNING: Only use in controlled sandbox environments)"`
	CursorModel       string `long:"cursor-model" description:"Model to use with Cursor agent (only applies when --agent=cursor)" choice:"gpt-5" choice:"sonnet-4" choice:"sonnet-4-thinking"`
	Version           bool   `long:"version" short:"v" description:"Show version information"`
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

	// Handle version flag
	if opts.Version {
		fmt.Printf("%s\n", core.GetVersion())
		os.Exit(0)
	}

	// Always enable info level logging
	log.SetLevel(slog.LevelInfo)

	// Log startup information
	log.Info("🚀 ccagent starting - version %s", core.GetVersion())
	log.Info("⚙️  Configuration: agent=%s, permission_mode=%s", opts.Agent, func() string {
		if opts.BypassPermissions {
			return "bypassPermissions"
		}
		return "acceptEdits"
	}())
	if opts.Agent == "cursor" && opts.CursorModel != "" {
		log.Info("⚙️  Cursor model: %s", opts.CursorModel)
	}
	cwd, err := os.Getwd()
	if err == nil {
		log.Info("📁 Working directory: %s", cwd)
	}

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
	logPath, err := cmdRunner.setupProgramLogging()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error setting up program logging: %v\n", err)
		os.Exit(1)
	}
	log.Info("📝 Logging to: %s", logPath)

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

	log.Info("🌐 WebSocket URL: %s", cmdRunner.wsURL)
	log.Info("🔑 Agent ID: %s", cmdRunner.agentID)

	// Set up deferred cleanup
	defer func() {
		// Stop environment manager periodic refresh
		if cmdRunner.envManager != nil {
			cmdRunner.envManager.Stop()
		}

		// Close rotating writer to prevent file handle leak
		if cmdRunner.rotatingWriter != nil {
			if err := cmdRunner.rotatingWriter.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to close log files: %v\n", err)
			}
		}

		if cmdRunner.rotatingWriter != nil {
			fmt.Fprintf(
				os.Stderr,
				"\n📝 App execution finished, logs for this session are in %s\n",
				cmdRunner.rotatingWriter.GetCurrentLogPath(),
			)
		}

		// Stop persistent worker pools on shutdown
		if cmdRunner.blockingWorkerPool != nil {
			cmdRunner.blockingWorkerPool.StopWait()
		}
		if cmdRunner.instantWorkerPool != nil {
			cmdRunner.instantWorkerPool.StopWait()
		}
	}()

	// Start Socket.IO client with backoff retry
	err = cmdRunner.startSocketIOClientWithRetry(cmdRunner.wsURL, cmdRunner.ccagentAPIKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting WebSocket client after retries: %v\n", err)
		os.Exit(1)
	}
}

// startSocketIOClientWithRetry wraps startSocketIOClient with exponential backoff retry logic
func (cr *CmdRunner) startSocketIOClientWithRetry(serverURLStr, apiKey string) error {
	// Start token monitoring routine independently (runs for app lifetime)
	tokenCtx, tokenCancel := context.WithCancel(context.Background())
	defer tokenCancel()
	cr.startTokenMonitoringRoutine(tokenCtx, cr.blockingWorkerPool)

	// Configure exponential backoff with unlimited retries
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = 2 * time.Second
	expBackoff.MaxInterval = 10 * time.Second
	expBackoff.MaxElapsedTime = 0 // No time limit

	attempt := 0
	operation := func() error {
		attempt++
		log.Info("🔄 Connection attempt %d", attempt)

		err := cr.startSocketIOClient(serverURLStr, apiKey)
		if err != nil {
			log.Error("❌ Connection attempt %d failed: %v", attempt, err)
			return err
		}
		return nil
	}

	notify := func(err error, next time.Duration) {
		log.Info("⏳ Retrying in %v...", next)
	}

	err := backoff.RetryNotify(operation, expBackoff, notify)
	if err != nil {
		return fmt.Errorf("failed to connect after %d attempts: %w", attempt, err)
	}

	return nil
}

func (cr *CmdRunner) startSocketIOClient(serverURLStr, apiKey string) error {
	log.Info("📋 Starting to connect to Socket.IO server at %s", serverURLStr)

	// Set up global interrupt handling
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	defer signal.Stop(interrupt)

	// Set up Socket.IO client options
	opts := socket.DefaultOptions()
	opts.SetTransports(types.NewSet(transports.Polling, transports.WebSocket))

	// Disable automatic reconnection - handle reconnection externally with backoff
	opts.SetReconnection(false)

	// Get repository identifier for header
	gitClient := clients.NewGitClient()
	repoIdentifier, err := gitClient.GetRepositoryIdentifier()
	if err != nil {
		return fmt.Errorf("failed to get repository identifier: %w", err)
	}

	// Set authentication headers
	opts.SetExtraHeaders(map[string][]string{
		"X-CCAGENT-API-KEY": {apiKey},
		"X-CCAGENT-ID":      {cr.agentID},
		"X-CCAGENT-REPO":    {repoIdentifier},
	})

	manager := socket.NewManager(serverURLStr, opts)
	socketClient := manager.Socket("/", opts)

	// Start MessageSender goroutine
	go cr.messageSender.Run(socketClient)
	log.Info("📤 Started MessageSender goroutine")

	// Use persistent worker pools across reconnects
	blockingWorkerPool := cr.blockingWorkerPool
	instantWorkerPool := cr.instantWorkerPool

	// Track connection state for auth failure detection
	connected := make(chan bool, 1)
	connectionError := make(chan error, 1)
	runtimeErrorChan := make(chan error, 1) // Errors after successful connection

	// Connection event handlers
	err = socketClient.On("connect", func(args ...any) {
		log.Info("✅ Connected to Socket.IO server, socket ID: %s", socketClient.Id())
		cr.connectionState.SetConnected(true)
		connected <- true
	})
	utils.AssertInvariant(err == nil, fmt.Sprintf("Failed to set up connect handler: %v", err))

	err = socketClient.On("connect_error", func(args ...any) {
		log.Error("❌ Socket.IO connection error: %v", args)
		connectionError <- fmt.Errorf("socket.io connection error: %v", args)
	})
	utils.AssertInvariant(err == nil, fmt.Sprintf("Failed to set up connect_error handler: %v", err))

	err = socketClient.On("disconnect", func(args ...any) {
		log.Info("🔌 Socket.IO disconnected: %v", args)
		cr.connectionState.SetConnected(false)

		// Send disconnect error to trigger reconnection
		reason := "unknown"
		if len(args) > 0 {
			reason = fmt.Sprintf("%v", args[0])
		}

		select {
		case runtimeErrorChan <- fmt.Errorf("socket disconnected: %s", reason):
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
			// Persist message to queue BEFORE submitting to worker pool for crash recovery
			if err := cr.messageHandler.PersistQueuedMessage(msg); err != nil {
				log.Error("❌ Failed to persist queued message: %v", err)
			}

			// Conversation messages need sequential processing
			blockingWorkerPool.Submit(func() {
				cr.messageHandler.HandleMessage(msg)
			})
		case models.MessageTypeCheckIdleJobs:
			// PR status checks can run in parallel without blocking conversations
			instantWorkerPool.Submit(func() {
				cr.messageHandler.HandleMessage(msg)
			})
		default:
			// Fallback to blocking pool for any unhandled message types
			blockingWorkerPool.Submit(func() {
				cr.messageHandler.HandleMessage(msg)
			})
		}
	})
	utils.AssertInvariant(err == nil, fmt.Sprintf("Failed to set up cc_message handler: %v", err))

	// Wait for initial connection or detect auth failure
	// Wait up to 10 seconds for initial connection
	select {
	case <-connected:
		log.Info("✅ Successfully authenticated with Socket.IO server")
	case err := <-connectionError:
		socketClient.Disconnect()
		return err
	case <-time.After(10 * time.Second):
		socketClient.Disconnect()
		return fmt.Errorf("connection timeout - server may have rejected authentication")
	}

	// Connection appears stable if not immediately disconnected within 5s (legacy guard removed)
	time.AfterFunc(5*time.Second, func() {
		log.Info("✅ Connection appears stable, continuing normal operation")
	})

	// Start ping routine once connected
	pingCtx, pingCancel := context.WithCancel(context.Background())
	defer pingCancel()
	cr.startPingRoutine(pingCtx, socketClient, runtimeErrorChan)

	// Wait for interrupt signal or runtime error
	select {
	case <-interrupt:
		log.Info("🔌 Interrupt received, closing Socket.IO connection...")
		socketClient.Disconnect()
		return nil
	case err := <-runtimeErrorChan:
		log.Error("❌ Runtime error occurred: %v", err)
		socketClient.Disconnect()
		return err
	}
}

func (cr *CmdRunner) setupProgramLogging() (string, error) {
	// Get config directory
	configDir, err := env.GetConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get config directory: %w", err)
	}

	// Create logs directory
	logsDir := filepath.Join(configDir, "logs")

	// Set up rotating writer with 10MB file size limit
	rotatingWriter, err := log.NewRotatingWriter(log.RotatingWriterConfig{
		LogDir:      logsDir,
		MaxFileSize: 1024, // 10MB
		FilePrefix:  "ccagent",
		Stdout:      os.Stdout,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create rotating writer: %w", err)
	}

	// Store rotating writer reference for cleanup
	cr.rotatingWriter = rotatingWriter

	// Set the rotating writer as the log output
	log.SetWriter(rotatingWriter)

	return rotatingWriter.GetCurrentLogPath(), nil
}

func (cr *CmdRunner) startPingRoutine(ctx context.Context, socketClient *socket.Socket, runtimeErrorChan chan<- error) {
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
				// Check if socket is still connected
				if !socketClient.Connected() {
					log.Error("❌ Socket disconnected, stopping ping routine")
					select {
					case runtimeErrorChan <- fmt.Errorf("socket disconnected during ping"):
					default:
						// Channel full, ignore
					}
					return
				}

				log.Info("💓 Sending ping to server")
				if err := socketClient.Emit("ping"); err != nil {
					log.Error("❌ Failed to send ping: %v", err)
					select {
					case runtimeErrorChan <- fmt.Errorf("failed to send ping: %w", err):
					default:
						// Channel full, ignore
					}
					return
				}
			}
		}
	}()
}

func (cr *CmdRunner) startTokenMonitoringRoutine(ctx context.Context, blockingWorkerPool *workerpool.WorkerPool) {
	log.Info("🔑 Starting token monitoring routine (checks every 10 minutes)")
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Info("🔑 Token monitoring routine stopped")
				return
			case <-ticker.C:
				log.Info("🔍 Checking token expiration...")
				// Schedule token refresh check on blocking queue
				// This ensures it runs sequentially with other conversation messages
				blockingWorkerPool.Submit(func() {
					refreshMsg := models.BaseMessage{
						ID:      core.NewID("msg"),
						Type:    models.MessageTypeRefreshToken,
						Payload: models.RefreshTokenPayload{},
					}
					cr.messageHandler.HandleMessage(refreshMsg)
				})
			}
		}
	}()
}
