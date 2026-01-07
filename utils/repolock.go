package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
)

// RepoLock represents a lock on a git repository
type RepoLock struct {
	lockFile *flock.Flock
	lockPath string
}

// NewRepoLock creates a new repository lock for the given repository path
func NewRepoLock(repoPath string) (*RepoLock, error) {
	// Validate that the path exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("repository path does not exist: %s", repoPath)
	}

	// Create lock file in .git directory
	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("not a git repository (no .git directory): %s", repoPath)
	}

	lockPath := filepath.Join(gitDir, "ccagent.lock")

	// Create flock instance
	lockFile := flock.New(lockPath)

	return &RepoLock{
		lockFile: lockFile,
		lockPath: lockPath,
	}, nil
}

// TryLock attempts to acquire the repository lock
// Returns nil if successful, error if lock is already held or other error occurs
func (rl *RepoLock) TryLock() error {
	locked, err := rl.lockFile.TryLock()
	if err != nil {
		return fmt.Errorf("failed to try lock: %w", err)
	}

	if !locked {
		return fmt.Errorf("another ccagent instance is already working on this repository")
	}

	return nil
}

// Unlock releases the repository lock and removes the lock file
func (rl *RepoLock) Unlock() error {
	if rl.lockFile == nil {
		return nil
	}

	// Unlock the file
	err := rl.lockFile.Unlock()
	if err != nil {
		return fmt.Errorf("failed to unlock: %w", err)
	}

	// Remove the lock file
	if err := os.Remove(rl.lockPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove lock file: %w", err)
	}

	return nil
}

// GetLockPath returns the path to the lock file (for debugging/testing)
func (rl *RepoLock) GetLockPath() string {
	return rl.lockPath
}
