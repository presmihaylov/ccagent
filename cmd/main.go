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
	codexclient "ccagent/clients/codex"
	cursorclient "ccagent/clients/cursor"
	opencodeclient "ccagent/clients/opencode"
	"ccagent/core"
	"ccagent/core/env"
	"ccagent/core/log"
	"ccagent/handlers"
	"ccagent/models"
	"ccagent/services"
	claudeservice "ccagent/services/claude"
	codexservice "ccagent/services/codex"
	cursorservice "ccagent/services/cursor"
	opencodeservice "ccagent/services/opencode"
	"ccagent/usecases"
	"ccagent/utils"
)

type CmdRunner struct {
	messageHandler  *handlers.MessageHandler
	messageSender   *handlers.MessageSender
	connectionState *handlers.ConnectionState
	gitUseCase      *usecases.GitUseCase
	appState        *models.AppState
	rotatingWriter  *log.RotatingWriter
	envManager      *env.EnvManager
	agentID         string
	agentsApiClient *clients.AgentsApiClient
	wsURL           string
	ccagentAPIKey   string

	// Persistent worker pools reused across reconnects
	blockingWorkerPool *workerpool.WorkerPool
	instantWorkerPool  *workerpool.WorkerPool
}

// validateModelForAgent checks if the specified model is compatible with the chosen agent
func validateModelForAgent(agentType, model string) error {
	// If no model specified, it's valid for all agents (they'll use defaults)
	if model == "" {
		return nil
	}

	switch agentType {
	case "claude":
		// Claude doesn't use model flags
		return fmt.Errorf("--model flag is not applicable for claude agent (claude uses the default model)")
	case "cursor":
		// Validate Cursor models
		validCursorModels := map[string]bool{
			"gpt-5":             true,
			"sonnet-4":          true,
			"sonnet-4-thinking": true,
		}
		if !validCursorModels[model] {
			return fmt.Errorf("--model '%s' is not valid for cursor agent (valid options: gpt-5, sonnet-4, sonnet-4-thinking)", model)
		}
	case "codex":
		// Codex accepts any model string (default: gpt-5)
		// No specific validation needed as it's flexible
	case "opencode":
		// OpenCode expects provider/model format (default: opencode/grok-code)
		if !strings.Contains(model, "/") {
			return fmt.Errorf("--model '%s' is not valid for opencode agent (expected format: provider/model, e.g., opencode/grok-code)", model)
		}
	default:
		return fmt.Errorf("unknown agent type: %s", agentType)
	}

	return nil
}

// fetchAndSetToken fetches the token from API and sets it as environment variable
func fetchAndSetToken(agentsApiClient *clients.AgentsApiClient, envManager *env.EnvManager) error {
	// Skip token operations for self-hosted installations
	if agentsApiClient.IsSelfHosted() {
		log.Info("🏠 Self-hosted installation detected, skipping token fetch")
		return nil
	}

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

func NewCmdRunner(agentType, permissionMode, model string) (*CmdRunner, error) {
	log.Info("📋 Starting to initialize CmdRunner with agent: %s", agentType)

	// Validate model compatibility with agent
	if err := validateModelForAgent(agentType, model); err != nil {
		return nil, err
	}

	// Create log directory for agent service
	configDir, err := env.GetConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}
	logDir := filepath.Join(configDir, "logs")

	// Initialize environment manager first
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

	// Get current working directory for Codex client
	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	// Create the appropriate CLI agent service (now with all dependencies available)
	cliAgent, err := createCLIAgent(agentType, permissionMode, model, logDir, workDir, agentsApiClient, envManager)
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
		messageHandler:  messageHandler,
		messageSender:   messageSender,
		connectionState: connectionState,
		gitUseCase:      gitUseCase,
		appState:        appState,
		envManager:      envManager,
		agentID:         agentID,
		agentsApiClient: agentsApiClient,
		wsURL:           wsURL,
		ccagentAPIKey:   ccagentAPIKey,
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
func createCLIAgent(
	agentType, permissionMode, model, logDir, workDir string,
	agentsApiClient *clients.AgentsApiClient,
	envManager *env.EnvManager,
) (services.CLIAgent, error) {
	// Apply default models when not specified
	if model == "" {
		switch agentType {
		case "codex":
			model = "gpt-5"
		case "opencode":
			model = "opencode/grok-code"
			// cursor and claude don't need defaults (cursor uses empty string, claude doesn't use models)
		}
	}

	switch agentType {
	case "claude":
		claudeClient := claudeclient.NewClaudeClient(permissionMode)
		return claudeservice.NewClaudeService(claudeClient, logDir, agentsApiClient, envManager), nil
	case "cursor":
		cursorClient := cursorclient.NewCursorClient()
		return cursorservice.NewCursorService(cursorClient, logDir, model), nil
	case "codex":
		codexClient := codexclient.NewCodexClient(permissionMode, workDir)
		return codexservice.NewCodexService(codexClient, logDir, model), nil
	case "opencode":
		opencodeClient := opencodeclient.NewOpenCodeClient()
		return opencodeservice.NewOpenCodeService(opencodeClient, logDir, model), nil
	default:
		return nil, fmt.Errorf("unsupported agent type: %s", agentType)
	}
}

type Options struct {
	//nolint
	Agent                       string `long:"agent" description:"CLI agent to use (claude, cursor, codex, or opencode)" choice:"claude" choice:"cursor" choice:"codex" choice:"opencode" default:"claude"`
	BypassPermissions           bool   `long:"bypass-permissions" description:"Use bypassPermissions mode for Claude/Codex (only applies when --agent=claude or --agent=codex) (WARNING: Only use in controlled sandbox environments)"`
	DeprecatedBypassPermissions bool   `long:"claude-bypass-permissions" hidden:"true"`
	Model                       string `long:"model" description:"Model to use (agent-specific: cursor: gpt-5/sonnet-4/sonnet-4-thinking, codex: any model string, opencode: provider/model format)"`
	Version                     bool   `long:"version" short:"v" description:"Show version information"`
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
	if opts.Model != "" {
		log.Info("⚙️  Model: %s", opts.Model)
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
	bypassPermissionsFlagUsed := opts.BypassPermissions || opts.DeprecatedBypassPermissions
	if bypassPermissionsFlagUsed {
		permissionMode = "bypassPermissions"
		if opts.DeprecatedBypassPermissions {
			fmt.Fprintf(
				os.Stderr,
				"Warning: --claude-bypass-permissions is deprecated. Use --bypass-permissions instead.\n",
			)
			fmt.Fprintf(
				os.Stderr,
				"Warning: --bypass-permissions flag should only be used in a controlled, sandbox environment. Otherwise, anyone from Slack will have access to your entire system\n",
			)
		} else {
			fmt.Fprintf(
				os.Stderr,
				"Warning: --bypass-permissions flag should only be used in a controlled, sandbox environment. Otherwise, anyone from Slack will have access to your entire system\n",
			)
		}
	}

	// OpenCode only supports bypassPermissions mode
	if opts.Agent == "opencode" && permissionMode != "bypassPermissions" {
		fmt.Fprintf(
			os.Stderr,
			"Error: OpenCode only supports bypassPermissions mode. Use --bypass-permissions flag.\n",
		)
		os.Exit(1)
	}

	cmdRunner, err := NewCmdRunner(opts.Agent, permissionMode, opts.Model)
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

	// Start token monitoring routine independently (runs for app lifetime)
	tokenCtx, tokenCancel := context.WithCancel(context.Background())
	defer tokenCancel()
	cmdRunner.startTokenMonitoringRoutine(tokenCtx, cmdRunner.blockingWorkerPool)

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
	// Skip token monitoring for self-hosted installations
	if cr.agentsApiClient.IsSelfHosted() {
		log.Info("🏠 Self-hosted installation detected, skipping token monitoring routine")
		return
	}

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
