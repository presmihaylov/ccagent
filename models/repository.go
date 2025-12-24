package models

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ccagent/clients"
	"ccagent/utils"
)

// RepositoryContext represents a single repository in a multi-repo configuration
type RepositoryContext struct {
	Path       string             // Absolute path to repository
	Identifier string             // org/repo-name from git remote
	GitClient  *clients.GitClient // Dedicated git client for this repo
	Lock       *utils.DirLock     // Directory lock for this repo
}

// MultiRepoConfig manages configuration for potentially multiple repositories
type MultiRepoConfig struct {
	Repositories []*RepositoryContext
	Primary      *RepositoryContext // First repository in list
}

// NewMultiRepoConfig creates a multi-repo configuration from comma-separated paths
// If paths is empty, uses current working directory as single repo
func NewMultiRepoConfig(repoPaths string) (*MultiRepoConfig, error) {
	var paths []string

	if repoPaths == "" {
		// Default to current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current working directory: %w", err)
		}
		paths = []string{cwd}
	} else {
		// Parse comma-separated paths
		paths = strings.Split(repoPaths, ",")
	}

	// Validate and create repository contexts
	var repoContexts []*RepositoryContext
	for _, path := range paths {
		// Trim whitespace
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}

		// Convert to absolute path
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for '%s': %w", path, err)
		}

		// Verify path exists
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("repository path does not exist: %s", absPath)
		}

		// Create git client for this repository
		gitClient := clients.NewGitClient()
		gitClient.SetWorkingDirectory(absPath)

		// Validate it's a git repository
		if err := gitClient.IsGitRepository(); err != nil {
			return nil, fmt.Errorf("path is not a git repository: %s (%w)", absPath, err)
		}

		// Get repository identifier
		identifier, err := gitClient.GetRepositoryIdentifier()
		if err != nil {
			return nil, fmt.Errorf("failed to get repository identifier for %s: %w", absPath, err)
		}

		// Create directory lock
		lock := utils.NewDirLockWithPath(absPath)

		repoContext := &RepositoryContext{
			Path:       absPath,
			Identifier: identifier,
			GitClient:  gitClient,
			Lock:       lock,
		}

		repoContexts = append(repoContexts, repoContext)
	}

	if len(repoContexts) == 0 {
		return nil, fmt.Errorf("no valid repositories specified")
	}

	return &MultiRepoConfig{
		Repositories: repoContexts,
		Primary:      repoContexts[0],
	}, nil
}

// IsSingleRepo returns true if this configuration has only one repository
func (m *MultiRepoConfig) IsSingleRepo() bool {
	return len(m.Repositories) == 1
}

// GetPaths returns all repository paths
func (m *MultiRepoConfig) GetPaths() []string {
	paths := make([]string, len(m.Repositories))
	for i, repo := range m.Repositories {
		paths[i] = repo.Path
	}
	return paths
}

// GetIdentifiers returns all repository identifiers (org/repo-name format)
func (m *MultiRepoConfig) GetIdentifiers() []string {
	identifiers := make([]string, len(m.Repositories))
	for i, repo := range m.Repositories {
		identifiers[i] = repo.Identifier
	}
	return identifiers
}

// TryLockAll attempts to acquire directory locks for all repositories
func (m *MultiRepoConfig) TryLockAll() error {
	// Try to lock all repositories
	lockedRepos := []*RepositoryContext{}
	for _, repo := range m.Repositories {
		if err := repo.Lock.TryLock(); err != nil {
			// Release all previously acquired locks before returning error
			for _, lockedRepo := range lockedRepos {
				_ = lockedRepo.Lock.Unlock()
			}
			return fmt.Errorf("failed to lock repository %s: %w", repo.Path, err)
		}
		lockedRepos = append(lockedRepos, repo)
	}
	return nil
}

// UnlockAll releases directory locks for all repositories
func (m *MultiRepoConfig) UnlockAll() error {
	var lastErr error
	for _, repo := range m.Repositories {
		if err := repo.Lock.Unlock(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
