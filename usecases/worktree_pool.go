package usecases

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"eksecd/clients"
	"eksecd/core/log"
)

// ErrPoolEmpty is returned when the pool has no available worktrees.
// This is an expected condition during initial fill or high load.
var ErrPoolEmpty = errors.New("pool is empty")

// ErrPoolStopping is returned when the pool is being shut down.
var ErrPoolStopping = errors.New("pool is stopping")

// PooledWorktree represents a pre-created worktree ready for use
type PooledWorktree struct {
	Path       string    // e.g., ~/.eksec_worktrees/pool-{uuid}
	BranchName string    // e.g., eksecd/pool-ready-{uuid}
	BaseCommit string    // Commit hash when created (for staleness check)
	CreatedAt  time.Time
}

// WorktreePool manages a pool of pre-created worktrees for fast job assignment.
// Pre-creating worktrees eliminates the 10-30+ second delay when starting jobs
// on large repositories (1GB+), providing instant worktree assignment.
type WorktreePool struct {
	ready         []PooledWorktree
	mutex         sync.Mutex
	gitClient     *clients.GitClient
	basePath      string        // ~/.eksec_worktrees/
	targetSize    int           // from WORKTREE_POOL_SIZE env
	replenishChan chan struct{} // signals replenisher
	stopChan      chan struct{} // for shutdown
	wg            sync.WaitGroup
}

// NewWorktreePool creates a new worktree pool.
// gitClient: the git client used for worktree operations
// basePath: the base directory where worktrees are stored (e.g., ~/.eksec_worktrees/)
// targetSize: the target number of worktrees to maintain in the pool
func NewWorktreePool(gitClient *clients.GitClient, basePath string, targetSize int) *WorktreePool {
	return &WorktreePool{
		ready:         make([]PooledWorktree, 0, targetSize),
		gitClient:     gitClient,
		basePath:      basePath,
		targetSize:    targetSize,
		replenishChan: make(chan struct{}, 1), // buffered to allow non-blocking sends
		stopChan:      make(chan struct{}),
	}
}

// Start begins the background replenisher goroutine that maintains the pool.
// This should be called after the pool is created and before acquiring worktrees.
// The context is used to signal graceful shutdown.
func (p *WorktreePool) Start(ctx context.Context) {
	p.wg.Add(1)
	go p.replenisherLoop(ctx)
}

// Stop gracefully shuts down the pool and waits for the background goroutine to finish.
// This should be called during application shutdown.
func (p *WorktreePool) Stop() {
	close(p.stopChan)
	p.wg.Wait()
}

// isStopping returns true if the pool has been signaled to stop.
func (p *WorktreePool) isStopping() bool {
	select {
	case <-p.stopChan:
		return true
	default:
		return false
	}
}

// Acquire gets a worktree from the pool and prepares it for a job.
// It renames the pool worktree directory and branch to match the job.
// If the pool is empty, returns an error so the caller can fall back to sync creation.
//
// Parameters:
//   - jobID: the unique identifier for the job (used as directory name)
//   - branchName: the target branch name for the job (e.g., "eksecd/adjective-noun-timestamp")
//
// Returns:
//   - worktreePath: the path to the ready-to-use worktree
//   - error: if the pool is empty or operations fail
func (p *WorktreePool) Acquire(jobID, branchName string) (string, error) {
	p.mutex.Lock()
	if len(p.ready) == 0 {
		p.mutex.Unlock()
		return "", ErrPoolEmpty
	}

	// Pop first available worktree
	pooledWT := p.ready[0]
	p.ready = p.ready[1:]
	poolSize := len(p.ready)
	p.mutex.Unlock()

	log.Info("🏊 Acquired worktree from pool (remaining: %d)", poolSize)

	// Signal replenisher (non-blocking)
	select {
	case p.replenishChan <- struct{}{}:
	default:
	}

	// Check staleness and refresh if needed
	currentCommit, err := p.getCurrentOriginCommit()
	if err != nil {
		log.Warn("⚠️ Failed to get current origin commit for staleness check: %v", err)
	} else if pooledWT.BaseCommit != currentCommit {
		log.Info("🔄 Worktree %s is stale, refreshing...", pooledWT.Path)
		if err := p.refreshWorktree(pooledWT.Path); err != nil {
			log.Warn("⚠️ Failed to refresh stale worktree: %v", err)
			// Continue anyway - it might still work
		}
	}

	// Move worktree: pool-{uuid} -> {jobID}
	// Use git worktree move instead of os.Rename to properly update git's internal tracking
	newPath := filepath.Join(p.basePath, jobID)
	if err := p.gitClient.MoveWorktree(pooledWT.Path, newPath); err != nil {
		// Failed to move - try to clean up and return error
		log.Error("❌ Failed to move worktree from %s to %s: %v", pooledWT.Path, newPath, err)
		// Attempt to remove the broken worktree
		p.cleanupFailedWorktree(pooledWT.Path, pooledWT.BranchName)
		return "", fmt.Errorf("failed to move worktree: %w", err)
	}

	// Rename branch: eksecd/pool-ready-{uuid} -> eksecd/{branchName}
	if err := p.renameBranch(newPath, pooledWT.BranchName, branchName); err != nil {
		log.Error("❌ Failed to rename branch from %s to %s: %v", pooledWT.BranchName, branchName, err)
		// Try to revert the worktree move
		if revertErr := p.gitClient.MoveWorktree(newPath, pooledWT.Path); revertErr != nil {
			log.Error("❌ Failed to revert worktree move: %v", revertErr)
		}
		return "", fmt.Errorf("failed to rename branch: %w", err)
	}

	log.Info("✅ Pool worktree ready: %s (branch: %s)", newPath, branchName)
	return newPath, nil
}

// GetPoolSize returns the current number of ready worktrees (thread-safe).
func (p *WorktreePool) GetPoolSize() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return len(p.ready)
}

// GetTargetSize returns the target pool size.
func (p *WorktreePool) GetTargetSize() int {
	return p.targetSize
}

// replenisherLoop is the background goroutine that maintains the pool.
// It fills the pool on startup, responds to acquire signals, and periodically
// refreshes stale worktrees.
func (p *WorktreePool) replenisherLoop(ctx context.Context) {
	defer p.wg.Done()

	// Initial fill on startup
	log.Info("🔄 Worktree pool: starting initial fill (target size: %d)", p.targetSize)
	p.fillToTarget(ctx)
	log.Info("✅ Worktree pool: initial fill complete (pool size: %d)", p.GetPoolSize())

	for {
		select {
		case <-ctx.Done():
			log.Info("🛑 Worktree pool replenisher stopped (context cancelled)")
			return
		case <-p.stopChan:
			log.Info("🛑 Worktree pool replenisher stopped (stop signal)")
			return
		case <-p.replenishChan:
			log.Debug("🔄 Worktree pool: replenish signal received")
			p.fillToTarget(ctx)
		case <-time.After(5 * time.Minute):
			// Periodic refresh of stale worktrees
			log.Debug("🔄 Worktree pool: periodic staleness check")
			p.refreshStaleWorktrees()
		}
	}
}

// fillToTarget creates worktrees until the pool reaches target size.
// It checks for context cancellation between each replenish attempt.
func (p *WorktreePool) fillToTarget(ctx context.Context) {
	for p.GetPoolSize() < p.targetSize {
		// Check for cancellation before each replenish
		select {
		case <-ctx.Done():
			log.Info("🛑 Worktree pool: fill interrupted by context cancellation")
			return
		case <-p.stopChan:
			log.Info("🛑 Worktree pool: fill interrupted by stop signal")
			return
		default:
		}

		if err := p.replenish(); err != nil {
			log.Error("❌ Worktree pool: failed to replenish: %v", err)
			return // back off on errors
		}
	}
}

// replenish creates one worktree and adds it to the pool.
func (p *WorktreePool) replenish() error {
	// Check for stop signal before starting
	if p.isStopping() {
		return ErrPoolStopping
	}

	// Generate unique ID
	id := uuid.New().String()[:8]
	wtPath := filepath.Join(p.basePath, fmt.Sprintf("pool-%s", id))
	branchName := fmt.Sprintf("eksecd/pool-ready-%s", id)

	// Ensure base directory exists
	if err := os.MkdirAll(p.basePath, 0755); err != nil {
		return fmt.Errorf("failed to create base directory: %w", err)
	}

	// Reset main repo to default branch before creating worktree to prevent
	// cross-pollination of changes between worktrees. This ensures the main
	// repository is in a clean, known state when spawning new worktrees.
	if err := p.resetMainRepoToDefaultBranch(); err != nil {
		log.Warn("⚠️ Failed to reset main repo before pool worktree creation: %v (continuing anyway)", err)
		// Continue anyway - worktree creation from origin/<default> might still work
	}

	// Fetch latest from origin (safe for concurrent calls)
	if err := p.gitClient.FetchOrigin(); err != nil {
		return fmt.Errorf("failed to fetch from origin: %w", err)
	}

	// Check for stop signal after slow network operation
	if p.isStopping() {
		return ErrPoolStopping
	}

	// Get current origin commit for staleness tracking
	baseCommit, err := p.getCurrentOriginCommit()
	if err != nil {
		return fmt.Errorf("failed to get origin commit: %w", err)
	}

	// Get default branch
	defaultBranch, err := p.gitClient.GetDefaultBranch()
	if err != nil {
		return fmt.Errorf("failed to get default branch: %w", err)
	}

	// Check for stop signal before creating worktree
	if p.isStopping() {
		return ErrPoolStopping
	}

	// Create worktree with new branch based on origin/<default-branch>
	baseRef := fmt.Sprintf("origin/%s", defaultBranch)
	if err := p.gitClient.CreateWorktree(wtPath, branchName, baseRef); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// Add to pool
	pooledWT := PooledWorktree{
		Path:       wtPath,
		BranchName: branchName,
		BaseCommit: baseCommit,
		CreatedAt:  time.Now(),
	}

	p.mutex.Lock()
	p.ready = append(p.ready, pooledWT)
	poolSize := len(p.ready)
	p.mutex.Unlock()

	log.Info("✅ Worktree pool: created pool worktree (pool size: %d)", poolSize)
	return nil
}

// refreshWorktree resets a worktree to the latest origin/main.
func (p *WorktreePool) refreshWorktree(wtPath string) error {
	// Fetch latest
	if err := p.gitClient.FetchOrigin(); err != nil {
		return fmt.Errorf("fetch failed: %w", err)
	}

	// Get default branch
	defaultBranch, err := p.gitClient.GetDefaultBranch()
	if err != nil {
		return fmt.Errorf("failed to get default branch: %w", err)
	}

	// Reset hard to origin/main
	if err := p.gitClient.ResetHardInWorktreeToRef(wtPath, fmt.Sprintf("origin/%s", defaultBranch)); err != nil {
		return fmt.Errorf("reset failed: %w", err)
	}

	return nil
}

// refreshStaleWorktrees checks and refreshes any stale worktrees in the pool.
func (p *WorktreePool) refreshStaleWorktrees() {
	currentCommit, err := p.getCurrentOriginCommit()
	if err != nil {
		log.Warn("⚠️ Failed to get current origin commit for staleness check: %v", err)
		return
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	refreshed := 0
	for i := range p.ready {
		if p.ready[i].BaseCommit != currentCommit {
			log.Debug("🔄 Refreshing stale worktree: %s", p.ready[i].Path)
			if err := p.refreshWorktree(p.ready[i].Path); err != nil {
				log.Warn("⚠️ Failed to refresh worktree %s: %v", p.ready[i].Path, err)
				continue
			}
			p.ready[i].BaseCommit = currentCommit
			refreshed++
		}
	}

	if refreshed > 0 {
		log.Info("✅ Worktree pool: refreshed %d stale worktrees", refreshed)
	}
}

// renameBranch renames a branch in a worktree.
func (p *WorktreePool) renameBranch(wtPath, oldBranch, newBranch string) error {
	return p.gitClient.RenameBranchInWorktree(wtPath, oldBranch, newBranch)
}

// getCurrentOriginCommit gets the current commit hash of origin/main.
func (p *WorktreePool) getCurrentOriginCommit() (string, error) {
	// Get default branch
	defaultBranch, err := p.gitClient.GetDefaultBranch()
	if err != nil {
		return "", err
	}

	return p.gitClient.GetOriginCommit(defaultBranch)
}

// resetMainRepoToDefaultBranch resets the main repository to the default branch.
// This prevents cross-pollination of changes when creating new worktrees.
func (p *WorktreePool) resetMainRepoToDefaultBranch() error {
	// Reset hard to discard any uncommitted changes
	if err := p.gitClient.ResetHard(); err != nil {
		return fmt.Errorf("failed to reset hard: %w", err)
	}

	// Clean untracked files
	if err := p.gitClient.CleanUntracked(); err != nil {
		return fmt.Errorf("failed to clean untracked: %w", err)
	}

	// Get default branch
	defaultBranch, err := p.gitClient.GetDefaultBranch()
	if err != nil {
		return fmt.Errorf("failed to get default branch: %w", err)
	}

	// Checkout default branch
	if err := p.gitClient.CheckoutBranch(defaultBranch); err != nil {
		return fmt.Errorf("failed to checkout default branch: %w", err)
	}

	// Pull latest changes
	if err := p.gitClient.PullLatest(); err != nil {
		return fmt.Errorf("failed to pull latest: %w", err)
	}

	return nil
}

// cleanupFailedWorktree attempts to clean up a worktree that failed during acquisition.
func (p *WorktreePool) cleanupFailedWorktree(wtPath, branchName string) {
	// Try to remove the worktree
	if err := p.gitClient.RemoveWorktree(wtPath); err != nil {
		log.Warn("⚠️ Failed to remove failed worktree %s: %v", wtPath, err)
	}

	// Try to delete the branch
	if err := p.gitClient.DeleteLocalBranch(branchName); err != nil {
		log.Warn("⚠️ Failed to delete branch %s: %v", branchName, err)
	}
}

// CleanupPool removes all pool worktrees. This should be called during shutdown
// if you want to clean up pool resources.
func (p *WorktreePool) CleanupPool() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	log.Info("🧹 Cleaning up worktree pool (%d worktrees)", len(p.ready))

	for _, wt := range p.ready {
		// Remove worktree
		if err := p.gitClient.RemoveWorktree(wt.Path); err != nil {
			log.Warn("⚠️ Failed to remove pool worktree %s: %v", wt.Path, err)
		}

		// Delete the branch
		if err := p.gitClient.DeleteLocalBranch(wt.BranchName); err != nil {
			log.Warn("⚠️ Failed to delete pool branch %s: %v", wt.BranchName, err)
		}
	}

	p.ready = nil
	return nil
}

// ReclaimOrphanedPoolWorktrees scans the base path for pool-* directories
// that aren't tracked and either reclaims them to the pool or removes them.
// This handles crash recovery scenarios.
func (p *WorktreePool) ReclaimOrphanedPoolWorktrees() error {
	log.Info("🔍 Scanning for orphaned pool worktrees")

	// Check if worktree directory exists
	if _, err := os.Stat(p.basePath); os.IsNotExist(err) {
		log.Info("ℹ️ Worktree base directory doesn't exist - nothing to reclaim")
		return nil
	}

	// List all directories in base path
	entries, err := os.ReadDir(p.basePath)
	if err != nil {
		return fmt.Errorf("failed to read worktree directory: %w", err)
	}

	// Get current commit for staleness tracking
	currentCommit, _ := p.getCurrentOriginCommit()

	reclaimedCount := 0
	removedCount := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Only process pool-* directories
		if !strings.HasPrefix(entry.Name(), "pool-") {
			continue
		}

		wtPath := filepath.Join(p.basePath, entry.Name())
		log.Info("🔍 Found orphaned pool worktree: %s", wtPath)

		// Check if it's a valid worktree
		if !p.gitClient.WorktreeExists(wtPath) {
			// Not a valid worktree, remove the directory
			log.Info("🗑️ Removing invalid pool directory: %s", wtPath)
			if err := os.RemoveAll(wtPath); err != nil {
				log.Warn("⚠️ Failed to remove invalid directory %s: %v", wtPath, err)
			} else {
				removedCount++
			}
			continue
		}

		// Get the branch name from the worktree
		branchName, err := p.gitClient.GetCurrentBranchInWorktree(wtPath)
		if err != nil {
			log.Warn("⚠️ Failed to get branch for worktree %s: %v", wtPath, err)
			continue
		}

		// Only reclaim if it has a pool-ready branch
		if !strings.HasPrefix(branchName, "eksecd/pool-ready-") {
			log.Info("⚠️ Worktree %s has non-pool branch %s, removing", wtPath, branchName)
			if err := p.gitClient.RemoveWorktree(wtPath); err != nil {
				log.Warn("⚠️ Failed to remove worktree: %v", err)
			} else {
				removedCount++
			}
			continue
		}

		// Reclaim to pool
		p.mutex.Lock()
		if len(p.ready) < p.targetSize {
			pooledWT := PooledWorktree{
				Path:       wtPath,
				BranchName: branchName,
				BaseCommit: currentCommit,
				CreatedAt:  time.Now(),
			}
			p.ready = append(p.ready, pooledWT)
			reclaimedCount++
			log.Info("♻️ Reclaimed pool worktree: %s", wtPath)
		} else {
			// Pool is full, remove the worktree
			p.mutex.Unlock()
			log.Info("🗑️ Pool full, removing excess worktree: %s", wtPath)
			if err := p.gitClient.RemoveWorktree(wtPath); err != nil {
				log.Warn("⚠️ Failed to remove excess worktree: %v", err)
			} else {
				removedCount++
			}
			continue
		}
		p.mutex.Unlock()
	}

	if reclaimedCount > 0 || removedCount > 0 {
		log.Info("✅ Pool worktree recovery: reclaimed %d, removed %d", reclaimedCount, removedCount)
	}

	return nil
}
