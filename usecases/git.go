package usecases

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/lucasepe/codename"

	"ccagent/clients"
	"ccagent/core/log"
	"ccagent/models"
	"ccagent/services"
)

type GitUseCase struct {
	gitClient     *clients.GitClient
	claudeService services.CLIAgent
	appState      *models.AppState
}

type CLIAgentResult struct {
	Output string
	Err    error
}

type AutoCommitResult struct {
	JustCreatedPR   bool
	PullRequestLink string
	PullRequestID   string // GitHub PR number (e.g., "123")
	CommitHash      string
	RepositoryURL   string
	BranchName      string
}

func NewGitUseCase(
	gitClient *clients.GitClient,
	claudeService services.CLIAgent,
	appState *models.AppState,
) *GitUseCase {
	return &GitUseCase{
		gitClient:     gitClient,
		claudeService: claudeService,
		appState:      appState,
	}
}

func (g *GitUseCase) ValidateGitEnvironment() error {
	log.Info("📋 Starting to validate Git environment")

	// Check if we're in a Git repository
	if err := g.gitClient.IsGitRepository(); err != nil {
		log.Error("❌ Not in a Git repository: %v", err)
		return fmt.Errorf("ccagent must be run from within a Git repository: %w", err)
	}

	// Check if we're at the Git repository root
	if err := g.gitClient.IsGitRepositoryRoot(); err != nil {
		log.Error("❌ Not at Git repository root: %v", err)
		return fmt.Errorf("ccagent must be run from the Git repository root: %w", err)
	}

	// Check if remote repository exists
	if err := g.gitClient.HasRemoteRepository(); err != nil {
		log.Error("❌ No remote repository configured: %v", err)
		return fmt.Errorf("git repository must have a remote configured: %w", err)
	}

	// Check if GitHub CLI is available (for PR creation)
	if err := g.gitClient.IsGitHubCLIAvailable(); err != nil {
		log.Error("❌ GitHub CLI not available: %v", err)
		return fmt.Errorf("GitHub CLI (gh) must be installed and configured: %w", err)
	}

	// Validate remote repository access credentials
	if err := g.gitClient.ValidateRemoteAccess(); err != nil {
		log.Error("❌ Remote repository access validation failed: %v", err)
		return fmt.Errorf("remote repository access validation failed: %w", err)
	}

	log.Info("✅ Git environment validation passed")
	log.Info("📋 Completed successfully - validated Git environment")
	return nil
}

// SwitchToJobBranch switches to the specified branch, discarding local changes and pulling latest from main
func (g *GitUseCase) SwitchToJobBranch(branchName string) error {
	log.Info("📋 Starting to switch to job branch: %s", branchName)

	// Step 1: Reset hard current branch to discard uncommitted changes
	if err := g.gitClient.ResetHard(); err != nil {
		log.Error("❌ Failed to reset hard: %v", err)
		return fmt.Errorf("failed to reset hard: %w", err)
	}

	// Step 2: Clean untracked files
	if err := g.gitClient.CleanUntracked(); err != nil {
		log.Error("❌ Failed to clean untracked files: %v", err)
		return fmt.Errorf("failed to clean untracked files: %w", err)
	}

	// Step 3: Get default branch and checkout to it
	defaultBranch, err := g.gitClient.GetDefaultBranch()
	if err != nil {
		log.Error("❌ Failed to get default branch: %v", err)
		return fmt.Errorf("failed to get default branch: %w", err)
	}

	if err := g.gitClient.CheckoutBranch(defaultBranch); err != nil {
		log.Error("❌ Failed to checkout default branch %s: %v", defaultBranch, err)
		return fmt.Errorf("failed to checkout default branch %s: %w", defaultBranch, err)
	}

	// Step 4: Pull latest changes
	if err := g.gitClient.PullLatest(); err != nil {
		log.Error("❌ Failed to pull latest changes: %v", err)
		return fmt.Errorf("failed to pull latest changes: %w", err)
	}

	// Step 5: Checkout target branch
	if err := g.gitClient.CheckoutBranch(branchName); err != nil {
		log.Error("❌ Failed to checkout target branch %s: %v", branchName, err)
		return fmt.Errorf("failed to checkout target branch %s: %w", branchName, err)
	}

	log.Info("✅ Successfully switched to job branch: %s", branchName)
	log.Info("📋 Completed successfully - switched to job branch")
	return nil
}

func (g *GitUseCase) PrepareForNewConversation(conversationHint string) (string, error) {
	log.Info("📋 Starting to prepare for new conversation")

	// Generate random branch name
	branchName, err := g.generateRandomBranchName()
	if err != nil {
		log.Error("❌ Failed to generate random branch name: %v", err)
		return "", fmt.Errorf("failed to generate branch name: %w", err)
	}

	log.Info("🌿 Generated branch name: %s", branchName)

	// Use the common branch switching logic to get to main and pull latest
	if err := g.resetAndPullDefaultBranch(); err != nil {
		log.Error("❌ Failed to reset and pull main: %v", err)
		return "", fmt.Errorf("failed to reset and pull main: %w", err)
	}

	// Create and checkout new branch
	if err := g.gitClient.CreateAndCheckoutBranch(branchName); err != nil {
		log.Error("❌ Failed to create and checkout new branch %s: %v", branchName, err)
		return "", fmt.Errorf("failed to create and checkout new branch %s: %w", branchName, err)
	}

	log.Info("✅ Successfully prepared for new conversation on branch: %s", branchName)
	log.Info("📋 Completed successfully - prepared for new conversation")
	return branchName, nil
}

// resetAndPullDefaultBranch is a helper that resets current branch, goes to main, and pulls latest
func (g *GitUseCase) resetAndPullDefaultBranch() error {
	log.Info("📋 Starting to reset and pull default branch")

	// Step 1: Reset hard current branch to discard uncommitted changes
	if err := g.gitClient.ResetHard(); err != nil {
		log.Error("❌ Failed to reset hard: %v", err)
		return fmt.Errorf("failed to reset hard: %w", err)
	}

	// Step 2: Clean untracked files
	if err := g.gitClient.CleanUntracked(); err != nil {
		log.Error("❌ Failed to clean untracked files: %v", err)
		return fmt.Errorf("failed to clean untracked files: %w", err)
	}

	// Step 3: Get default branch and checkout to it
	defaultBranch, err := g.gitClient.GetDefaultBranch()
	if err != nil {
		log.Error("❌ Failed to get default branch: %v", err)
		return fmt.Errorf("failed to get default branch: %w", err)
	}

	if err := g.gitClient.CheckoutBranch(defaultBranch); err != nil {
		log.Error("❌ Failed to checkout default branch %s: %v", defaultBranch, err)
		return fmt.Errorf("failed to checkout default branch %s: %w", defaultBranch, err)
	}

	// Step 4: Pull latest changes
	if err := g.gitClient.PullLatest(); err != nil {
		log.Error("❌ Failed to pull latest changes: %v", err)
		return fmt.Errorf("failed to pull latest changes: %w", err)
	}

	log.Info("✅ Successfully reset and pulled main")
	log.Info("📋 Completed successfully - reset and pulled main")
	return nil
}

func (g *GitUseCase) AutoCommitChangesIfNeeded(slackThreadLink string) (*AutoCommitResult, error) {
	log.Info("📋 Starting to auto-commit changes if needed")

	// Get current branch first (needed for both cases)
	currentBranch, err := g.gitClient.GetCurrentBranch()
	if err != nil {
		log.Error("❌ Failed to get current branch: %v", err)
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	// Check if there are any uncommitted changes
	hasChanges, err := g.gitClient.HasUncommittedChanges()
	if err != nil {
		log.Error("❌ Failed to check for uncommitted changes: %v", err)
		return nil, fmt.Errorf("failed to check for uncommitted changes: %w", err)
	}

	if !hasChanges {
		log.Info("ℹ️ No uncommitted changes found - skipping auto-commit")
		log.Info("📋 Completed successfully - no changes to commit")
		return &AutoCommitResult{
			JustCreatedPR:   false,
			PullRequestLink: "",
			PullRequestID:   "",
			CommitHash:      "",
			RepositoryURL:   "",
			BranchName:      currentBranch,
		}, nil
	}

	log.Info("✅ Uncommitted changes detected - proceeding with auto-commit")

	// Generate commit message using Claude
	commitMessage, err := g.generateCommitMessageWithClaude(currentBranch)
	if err != nil {
		log.Error("❌ Failed to generate commit message with Claude: %v", err)
		return nil, fmt.Errorf("failed to generate commit message with Claude: %w", err)
	}

	log.Info("📝 Generated commit message: %s", commitMessage)

	// Add all changes
	if err := g.gitClient.AddAll(); err != nil {
		log.Error("❌ Failed to add all changes: %v", err)
		return nil, fmt.Errorf("failed to add all changes: %w", err)
	}

	// Commit with message
	if err := g.gitClient.Commit(commitMessage); err != nil {
		log.Error("❌ Failed to commit changes: %v", err)
		return nil, fmt.Errorf("failed to commit changes: %w", err)
	}

	// Get commit hash after successful commit
	commitHash, err := g.gitClient.GetLatestCommitHash()
	if err != nil {
		log.Error("❌ Failed to get commit hash: %v", err)
		return nil, fmt.Errorf("failed to get commit hash: %w", err)
	}

	// Get repository URL for commit link
	repositoryURL, err := g.gitClient.GetRemoteURL()
	if err != nil {
		log.Error("❌ Failed to get repository URL: %v", err)
		return nil, fmt.Errorf("failed to get repository URL: %w", err)
	}

	// Push current branch to remote
	if err := g.gitClient.PushBranch(currentBranch); err != nil {
		log.Error("❌ Failed to push branch %s: %v", currentBranch, err)
		return nil, fmt.Errorf("failed to push branch %s: %w", currentBranch, err)
	}

	// Handle PR creation/update
	prResult, err := g.handlePRCreationOrUpdate(currentBranch, slackThreadLink)
	if err != nil {
		log.Error("❌ Failed to handle PR creation/update: %v", err)
		return nil, fmt.Errorf("failed to handle PR creation/update: %w", err)
	}

	// Update the result with commit information
	prResult.CommitHash = commitHash
	prResult.RepositoryURL = repositoryURL

	// Extract and store PR ID from the PR URL if available
	if prResult.PullRequestLink != "" {
		prResult.PullRequestID = g.gitClient.ExtractPRIDFromURL(prResult.PullRequestLink)
	}

	log.Info("✅ Successfully auto-committed and pushed changes")
	log.Info("📋 Completed successfully - auto-committed changes")
	return prResult, nil
}

func (g *GitUseCase) generateRandomBranchName() (string, error) {
	log.Info("🎲 Generating random branch name")

	rng, err := codename.DefaultRNG()
	if err != nil {
		return "", fmt.Errorf("failed to create random generator: %w", err)
	}

	randomName := codename.Generate(rng, 0)
	timestamp := time.Now().Format("20060102-150405")
	finalBranchName := fmt.Sprintf("ccagent/%s-%s", randomName, timestamp)

	log.Info("🎲 Generated random name: %s", finalBranchName)
	return finalBranchName, nil
}

func (g *GitUseCase) generateCommitMessageWithClaude(branchName string) (string, error) {
	log.Info("🤖 Asking Claude to generate commit message")

	prompt := CommitMessageGenerationPrompt(branchName)

	result, err := g.claudeService.StartNewConversationWithDisallowedTools(prompt, []string{"Bash(gh:*)"})
	if err != nil {
		return "", fmt.Errorf("claude failed to generate commit message: %w", err)
	}

	return strings.TrimSpace(result.Output), nil
}

func (g *GitUseCase) handlePRCreationOrUpdate(branchName, slackThreadLink string) (*AutoCommitResult, error) {
	log.Info("📋 Starting to handle PR creation or update for branch: %s", branchName)

	// Check if a PR already exists for this branch
	hasExistingPR, err := g.gitClient.HasExistingPR(branchName)
	if err != nil {
		log.Error("❌ Failed to check for existing PR: %v", err)
		return nil, fmt.Errorf("failed to check for existing PR: %w", err)
	}

	if hasExistingPR {
		log.Info("✅ Existing PR found for branch %s - changes have been pushed", branchName)

		// Get the PR URL for the existing PR
		prURL, err := g.gitClient.GetPRURL(branchName)
		if err != nil {
			log.Error("❌ Failed to get PR URL for existing PR: %v", err)
			// Continue without the URL rather than failing
			prURL = ""
		}

		// Update PR title and description based on new changes
		if err := g.updatePRTitleAndDescriptionIfNeeded(branchName, slackThreadLink); err != nil {
			log.Error("❌ Failed to update PR title/description: %v", err)
			// Log error but don't fail the entire operation
		}

		log.Info("📋 Completed successfully - updated existing PR")
		return &AutoCommitResult{
			JustCreatedPR:   false,
			PullRequestLink: prURL,
			PullRequestID:   g.gitClient.ExtractPRIDFromURL(prURL),
			CommitHash:      "", // Will be filled in by caller
			RepositoryURL:   "", // Will be filled in by caller
			BranchName:      branchName,
		}, nil
	}

	log.Info("🆕 No existing PR found - creating new PR")

	// Generate PR title and body using Claude in parallel
	titleChan := make(chan CLIAgentResult)
	bodyChan := make(chan CLIAgentResult)

	// Start PR title generation
	go func() {
		output, err := g.generatePRTitleWithClaude(branchName)
		titleChan <- CLIAgentResult{Output: output, Err: err}
	}()

	// Start PR body generation
	go func() {
		output, err := g.generatePRBodyWithClaude(branchName, slackThreadLink)
		bodyChan <- CLIAgentResult{Output: output, Err: err}
	}()

	// Wait for both to complete and collect results
	titleRes := <-titleChan
	bodyRes := <-bodyChan

	// Check for errors
	if titleRes.Err != nil {
		log.Error("❌ Failed to generate PR title with Claude: %v", titleRes.Err)
		return nil, fmt.Errorf("failed to generate PR title with Claude: %w", titleRes.Err)
	}

	if bodyRes.Err != nil {
		log.Error("❌ Failed to generate PR body with Claude: %v", bodyRes.Err)
		return nil, fmt.Errorf("failed to generate PR body with Claude: %w", bodyRes.Err)
	}

	prTitle := titleRes.Output
	prBody := bodyRes.Output

	log.Info("📋 Generated PR title: %s", prTitle)

	// Get default branch for PR base
	defaultBranch, err := g.gitClient.GetDefaultBranch()
	if err != nil {
		log.Error("❌ Failed to get default branch: %v", err)
		return nil, fmt.Errorf("failed to get default branch: %w", err)
	}

	// Create pull request
	prURL, err := g.gitClient.CreatePullRequest(prTitle, prBody, defaultBranch)
	if err != nil {
		log.Error("❌ Failed to create pull request: %v", err)
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	log.Info("✅ Successfully created PR: %s", prTitle)
	log.Info("📋 Completed successfully - created new PR")
	return &AutoCommitResult{
		JustCreatedPR:   true,
		PullRequestLink: prURL,
		PullRequestID:   g.gitClient.ExtractPRIDFromURL(prURL),
		CommitHash:      "", // Will be filled in by caller
		RepositoryURL:   "", // Will be filled in by caller
		BranchName:      branchName,
	}, nil
}

func (g *GitUseCase) generatePRTitleWithClaude(branchName string) (string, error) {
	log.Info("🤖 Asking Claude to generate PR title")

	// Get default branch to compare against
	defaultBranch, err := g.gitClient.GetDefaultBranch()
	if err != nil {
		return "", fmt.Errorf("failed to get default branch: %w", err)
	}

	// Get commit messages specific to this branch
	commitMessages, err := g.gitClient.GetBranchCommitMessages(branchName, defaultBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get branch commit messages: %w", err)
	}

	// Get diff summary
	diffSummary, err := g.gitClient.GetBranchDiffSummary(branchName, defaultBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get branch diff summary: %w", err)
	}

	// Get actual diff content for better context
	diffContent, err := g.gitClient.GetBranchDiffContent(branchName, defaultBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get branch diff content: %w", err)
	}

	// Build commit info for context
	commitInfo := "No commits found"
	if len(commitMessages) > 0 {
		commitInfo = fmt.Sprintf("Recent commits:\n%s", strings.Join(commitMessages, "\n"))
	}

	prompt := PRTitleGenerationPrompt(branchName, commitInfo, diffSummary, diffContent)

	result, err := g.claudeService.StartNewConversationWithDisallowedTools(prompt, []string{"Bash(gh:*)"})
	if err != nil {
		return "", fmt.Errorf("claude failed to generate PR title: %w", err)
	}

	return strings.TrimSpace(result.Output), nil
}

func (g *GitUseCase) generatePRBodyWithClaude(branchName, slackThreadLink string) (string, error) {
	log.Info("🤖 Asking Claude to generate PR body")

	// Get default branch to compare against
	defaultBranch, err := g.gitClient.GetDefaultBranch()
	if err != nil {
		return "", fmt.Errorf("failed to get default branch: %w", err)
	}

	// Get commit messages specific to this branch
	commitMessages, err := g.gitClient.GetBranchCommitMessages(branchName, defaultBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get branch commit messages: %w", err)
	}

	// Get diff summary
	diffSummary, err := g.gitClient.GetBranchDiffSummary(branchName, defaultBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get branch diff summary: %w", err)
	}

	// Get actual diff content for better context
	diffContent, err := g.gitClient.GetBranchDiffContent(branchName, defaultBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get branch diff content: %w", err)
	}

	// Build commit info for context
	commitInfo := "No commits found"
	if len(commitMessages) > 0 {
		commitInfo = strings.Join(commitMessages, "\n- ")
		commitInfo = "- " + commitInfo // Add bullet to first item
	}

	prompt := PRDescriptionGenerationPrompt(branchName, commitInfo, diffSummary, diffContent)

	result, err := g.claudeService.StartNewConversationWithDisallowedTools(prompt, []string{"Bash(gh:*)"})
	if err != nil {
		return "", fmt.Errorf("claude failed to generate PR body: %w", err)
	}

	// Append footer with Slack thread link
	cleanBody := strings.TrimSpace(result.Output)
	finalBody := cleanBody + fmt.Sprintf(
		"\n\n---\nGenerated by [Claude Control](https://claudecontrol.com) from this [slack thread](%s)",
		slackThreadLink,
	)

	return finalBody, nil
}

func (g *GitUseCase) ValidateAndRestorePRDescriptionFooter(slackThreadLink string) error {
	log.Info("📋 Starting to validate and restore PR description footer")

	// Get current branch
	currentBranch, err := g.gitClient.GetCurrentBranch()
	if err != nil {
		log.Error("❌ Failed to get current branch: %v", err)
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Check if a PR exists for this branch
	hasExistingPR, err := g.gitClient.HasExistingPR(currentBranch)
	if err != nil {
		log.Error("❌ Failed to check for existing PR: %v", err)
		return fmt.Errorf("failed to check for existing PR: %w", err)
	}

	if !hasExistingPR {
		log.Info("ℹ️ No existing PR found - skipping footer validation")
		log.Info("📋 Completed successfully - no PR to validate")
		return nil
	}

	// Get current PR description
	currentDescription, err := g.gitClient.GetPRDescription(currentBranch)
	if err != nil {
		log.Error("❌ Failed to get PR description: %v", err)
		return fmt.Errorf("failed to get PR description: %w", err)
	}

	// Check if the expected footer pattern is present (using regex to handle different permalinks)
	footerPattern := `---\s*\n.*Generated by \[Claude Control\]\(https://claudecontrol\.com\) from this \[slack thread\]\([^)]+\)`

	matched, err := regexp.MatchString(footerPattern, currentDescription)
	if err != nil {
		log.Error("❌ Failed to match footer pattern: %v", err)
		return fmt.Errorf("failed to match footer pattern: %w", err)
	}

	if matched {
		log.Info("✅ PR description already has correct Claude Control footer")
		log.Info("📋 Completed successfully - footer validation passed")
		return nil
	}

	log.Info("🔧 PR description missing Claude Control footer - restoring it")

	// Remove any existing footer lines to avoid duplicates
	lines := strings.Split(currentDescription, "\n")
	var cleanedLines []string
	inFooterSection := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check if we've hit a footer section
		if strings.Contains(trimmedLine, "Generated by Claude Control") ||
			strings.Contains(trimmedLine, "Generated by Claude Code") {
			inFooterSection = true
			continue
		}

		// Skip separator lines that are part of footer
		if trimmedLine == "---" {
			// Look ahead to see if this separator is followed by footer content
			isFooterSeparator := false
			for i := len(cleanedLines); i < len(lines)-1; i++ {
				nextLine := strings.TrimSpace(lines[i+1])
				if nextLine == "" {
					continue
				}
				if strings.Contains(nextLine, "Generated by Claude") {
					isFooterSeparator = true
				}
				break
			}

			if isFooterSeparator || inFooterSection {
				continue
			}
		}

		// Skip empty lines in footer section
		if inFooterSection && trimmedLine == "" {
			continue
		}

		cleanedLines = append(cleanedLines, line)
	}

	// Remove trailing empty lines
	for len(cleanedLines) > 0 && strings.TrimSpace(cleanedLines[len(cleanedLines)-1]) == "" {
		cleanedLines = cleanedLines[:len(cleanedLines)-1]
	}

	// Add the correct footer
	expectedFooter := fmt.Sprintf(
		"Generated by [Claude Control](https://claudecontrol.com) from this [slack thread](%s)",
		slackThreadLink,
	)
	restoredDescription := strings.Join(cleanedLines, "\n")
	if restoredDescription != "" {
		// Check if description already ends with a separator
		if strings.HasSuffix(strings.TrimSpace(restoredDescription), "---") {
			restoredDescription += "\n" + expectedFooter
		} else {
			restoredDescription += "\n\n---\n" + expectedFooter
		}
	} else {
		restoredDescription = "---\n" + expectedFooter
	}

	// Update the PR description
	if err := g.gitClient.UpdatePRDescription(currentBranch, restoredDescription); err != nil {
		log.Error("❌ Failed to update PR description: %v", err)
		return fmt.Errorf("failed to update PR description: %w", err)
	}

	log.Info("✅ Successfully restored Claude Control footer to PR description")
	log.Info("📋 Completed successfully - restored PR description footer")
	return nil
}

func (g *GitUseCase) CheckPRStatus(branchName string) (string, error) {
	log.Info("📋 Starting to check PR status for branch: %s", branchName)

	// First check if a PR exists for this branch
	hasExistingPR, err := g.gitClient.HasExistingPR(branchName)
	if err != nil {
		log.Error("❌ Failed to check for existing PR for branch %s: %v", branchName, err)
		return "", fmt.Errorf("failed to check for existing PR: %w", err)
	}

	if !hasExistingPR {
		log.Info("📋 No PR found for branch %s", branchName)
		return "no_pr", nil
	}

	// Get PR status using GitHub CLI
	prStatus, err := g.gitClient.GetPRState(branchName)
	if err != nil {
		log.Error("❌ Failed to get PR state for branch %s: %v", branchName, err)
		return "", fmt.Errorf("failed to get PR state: %w", err)
	}

	log.Info("📋 Completed successfully - PR status for branch %s: %s", branchName, prStatus)
	return prStatus, nil
}

func (g *GitUseCase) CheckPRStatusByID(prID string) (string, error) {
	log.Info("📋 Starting to check PR status by ID: %s", prID)

	// Get PR status directly by PR ID using GitHub CLI
	prStatus, err := g.gitClient.GetPRStateByID(prID)
	if err != nil {
		log.Error("❌ Failed to get PR state for PR ID %s: %v", prID, err)
		return "", fmt.Errorf("failed to get PR state by ID: %w", err)
	}

	log.Info("📋 Completed successfully - PR status for ID %s: %s", prID, prStatus)
	return prStatus, nil
}

func (g *GitUseCase) CleanupStaleBranches() error {
	log.Info("📋 Starting to cleanup stale ccagent branches")

	// Get all local branches
	localBranches, err := g.gitClient.GetLocalBranches()
	if err != nil {
		log.Error("❌ Failed to get local branches: %v", err)
		return fmt.Errorf("failed to get local branches: %w", err)
	}

	// Get current branch to avoid deleting it
	currentBranch, err := g.gitClient.GetCurrentBranch()
	if err != nil {
		log.Error("❌ Failed to get current branch: %v", err)
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Get default branch to avoid deleting it
	defaultBranch, err := g.gitClient.GetDefaultBranch()
	if err != nil {
		log.Error("❌ Failed to get default branch: %v", err)
		return fmt.Errorf("failed to get default branch: %w", err)
	}

	// Get all tracked job branches from app state
	trackedJobs := g.appState.GetAllJobs()
	trackedBranches := make(map[string]bool)
	for _, jobData := range trackedJobs {
		if jobData.BranchName != "" {
			trackedBranches[jobData.BranchName] = true
		}
	}

	// Filter branches for cleanup
	var branchesToDelete []string
	protectedBranches := []string{"main", "master", currentBranch, defaultBranch}

	for _, branch := range localBranches {
		// Only process ccagent/ branches
		if !strings.HasPrefix(branch, "ccagent/") {
			continue
		}

		// Skip protected branches
		isProtected := false
		for _, protected := range protectedBranches {
			if branch == protected {
				isProtected = true
				break
			}
		}
		if isProtected {
			log.Info("⚠️ Skipping protected branch: %s", branch)
			continue
		}

		// Skip tracked branches
		if trackedBranches[branch] {
			log.Info("⚠️ Skipping tracked branch: %s", branch)
			continue
		}

		// This branch is stale - mark for deletion
		branchesToDelete = append(branchesToDelete, branch)
	}

	if len(branchesToDelete) == 0 {
		log.Info("✅ No stale ccagent branches found")
		log.Info("📋 Completed successfully - no stale branches to cleanup")
		return nil
	}

	log.Info("🧹 Found %d stale ccagent branches to delete", len(branchesToDelete))

	// Delete each stale branch
	deletedCount := 0
	for _, branch := range branchesToDelete {
		if err := g.gitClient.DeleteLocalBranch(branch); err != nil {
			log.Error("❌ Failed to delete stale branch %s: %v", branch, err)
			// Continue with other branches even if one fails
			continue
		}
		deletedCount++
		log.Info("🗑️ Deleted stale branch: %s", branch)
	}

	log.Info("✅ Successfully deleted %d out of %d stale ccagent branches", deletedCount, len(branchesToDelete))
	log.Info("📋 Completed successfully - cleaned up stale branches")
	return nil
}

func (g *GitUseCase) updatePRTitleAndDescriptionIfNeeded(branchName, slackThreadLink string) error {
	log.Info("📋 Starting to update PR title and description if needed for branch: %s", branchName)

	// Get current PR title and description
	currentTitle, err := g.gitClient.GetPRTitle(branchName)
	if err != nil {
		log.Error("❌ Failed to get current PR title: %v", err)
		return fmt.Errorf("failed to get current PR title: %w", err)
	}

	currentDescription, err := g.gitClient.GetPRDescription(branchName)
	if err != nil {
		log.Error("❌ Failed to get current PR description: %v", err)
		return fmt.Errorf("failed to get current PR description: %w", err)
	}

	// Generate updated PR title and description using Claude in parallel
	titleUpdateChan := make(chan CLIAgentResult)
	descriptionUpdateChan := make(chan CLIAgentResult)

	// Start updated PR title generation
	go func() {
		output, err := g.generateUpdatedPRTitleWithClaude(branchName, currentTitle)
		titleUpdateChan <- CLIAgentResult{Output: output, Err: err}
	}()

	// Start updated PR description generation
	go func() {
		output, err := g.generateUpdatedPRDescriptionWithClaude(
			branchName,
			currentDescription,
			slackThreadLink,
		)
		descriptionUpdateChan <- CLIAgentResult{Output: output, Err: err}
	}()

	// Wait for both to complete and collect results
	titleUpdateRes := <-titleUpdateChan
	descriptionUpdateRes := <-descriptionUpdateChan

	// Check for errors
	if titleUpdateRes.Err != nil {
		log.Error("❌ Failed to generate updated PR title with Claude: %v", titleUpdateRes.Err)
		return fmt.Errorf("failed to generate updated PR title with Claude: %w", titleUpdateRes.Err)
	}

	if descriptionUpdateRes.Err != nil {
		log.Error("❌ Failed to generate updated PR description with Claude: %v", descriptionUpdateRes.Err)
		return fmt.Errorf("failed to generate updated PR description with Claude: %w", descriptionUpdateRes.Err)
	}

	updatedTitle := titleUpdateRes.Output
	updatedDescription := descriptionUpdateRes.Output

	// Update title if it has changed
	if strings.TrimSpace(updatedTitle) != strings.TrimSpace(currentTitle) {
		log.Info("🔄 PR title has changed, updating...")
		if err := g.gitClient.UpdatePRTitle(branchName, updatedTitle); err != nil {
			log.Error("❌ Failed to update PR title: %v", err)
			return fmt.Errorf("failed to update PR title: %w", err)
		}
		log.Info("✅ Successfully updated PR title")
	} else {
		log.Info("ℹ️ PR title remains the same - no update needed")
	}

	// Update description if it has changed
	if strings.TrimSpace(updatedDescription) != strings.TrimSpace(currentDescription) {
		log.Info("🔄 PR description has changed, updating...")
		if err := g.gitClient.UpdatePRDescription(branchName, updatedDescription); err != nil {
			log.Error("❌ Failed to update PR description: %v", err)
			return fmt.Errorf("failed to update PR description: %w", err)
		}
		log.Info("✅ Successfully updated PR description")
	} else {
		log.Info("ℹ️ PR description remains the same - no update needed")
	}

	log.Info("📋 Completed successfully - updated PR title and description if needed")
	return nil
}

func (g *GitUseCase) generateUpdatedPRTitleWithClaude(branchName, currentTitle string) (string, error) {
	log.Info("🤖 Asking Claude to generate updated PR title")

	// Get default branch to compare against
	defaultBranch, err := g.gitClient.GetDefaultBranch()
	if err != nil {
		return "", fmt.Errorf("failed to get default branch: %w", err)
	}

	// Get recent commits since the branch was created
	commitMessages, err := g.gitClient.GetBranchCommitMessages(branchName, defaultBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get branch commit messages: %w", err)
	}

	// Get diff summary
	diffSummary, err := g.gitClient.GetBranchDiffSummary(branchName, defaultBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get branch diff summary: %w", err)
	}

	// Build commit info for context
	commitInfo := "No commits found"
	if len(commitMessages) > 0 {
		commitInfo = fmt.Sprintf("All commits on this branch:\n%s", strings.Join(commitMessages, "\n"))
	}

	prompt := PRTitleUpdatePrompt(currentTitle, branchName, commitInfo, diffSummary)

	result, err := g.claudeService.StartNewConversationWithDisallowedTools(prompt, []string{"Bash(gh:*)"})
	if err != nil {
		return "", fmt.Errorf("claude failed to generate updated PR title: %w", err)
	}

	return strings.TrimSpace(result.Output), nil
}

func (g *GitUseCase) generateUpdatedPRDescriptionWithClaude(
	branchName, currentDescription, slackThreadLink string,
) (string, error) {
	log.Info("🤖 Asking Claude to generate updated PR description")

	// Get default branch to compare against
	defaultBranch, err := g.gitClient.GetDefaultBranch()
	if err != nil {
		return "", fmt.Errorf("failed to get default branch: %w", err)
	}

	// Get recent commits since the branch was created
	commitMessages, err := g.gitClient.GetBranchCommitMessages(branchName, defaultBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get branch commit messages: %w", err)
	}

	// Get diff summary
	diffSummary, err := g.gitClient.GetBranchDiffSummary(branchName, defaultBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get branch diff summary: %w", err)
	}

	// Build commit info for context
	commitInfo := "No commits found"
	if len(commitMessages) > 0 {
		commitInfo = strings.Join(commitMessages, "\n- ")
		commitInfo = "- " + commitInfo // Add bullet to first item
	}

	// Remove existing footer from current description for analysis
	currentDescriptionClean := g.removeFooterFromDescription(currentDescription)

	prompt := PRDescriptionUpdatePrompt(currentDescriptionClean, branchName, commitInfo, diffSummary)

	result, err := g.claudeService.StartNewConversationWithDisallowedTools(prompt, []string{"Bash(gh:*)"})
	if err != nil {
		return "", fmt.Errorf("claude failed to generate updated PR description: %w", err)
	}

	// Append footer with Slack thread link
	cleanBody := strings.TrimSpace(result.Output)
	finalBody := cleanBody + fmt.Sprintf(
		"\n\n---\nGenerated by [Claude Control](https://claudecontrol.com) from this [slack thread](%s)",
		slackThreadLink,
	)

	return finalBody, nil
}

func (g *GitUseCase) removeFooterFromDescription(description string) string {
	// Remove the Claude Control footer to get clean description for analysis
	footerPattern := `---\s*\n.*Generated by \[Claude Control\]\(https://claudecontrol\.com\) from this \[slack thread\]\([^)]+\)`

	// Use regex to remove the footer section
	re := regexp.MustCompile(footerPattern)
	cleanDescription := re.ReplaceAllString(description, "")

	// Clean up any trailing whitespace
	return strings.TrimSpace(cleanDescription)
}
